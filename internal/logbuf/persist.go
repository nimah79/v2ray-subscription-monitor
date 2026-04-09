package logbuf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const defaultLogSubdir = "V2Ray Subscription Monitor"
const defaultLogFileName = "subscription-requests.jsonl"

// DefaultLogFilePath returns the default JSONL log path under os.UserConfigDir().
func DefaultLogFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultLogSubdir, defaultLogFileName), nil
}

// NewPersistent creates a buffer that loads existing entries from path and saves on each change.
func NewPersistent(maxEntries int, maxAge time.Duration, path string) (*Buffer, error) {
	b := New(maxEntries, maxAge)
	if path == "" {
		return b, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.path = path
	if err := b.loadFromFileLocked(); err != nil {
		return nil, err
	}
	b.pruneLocked(time.Now())
	return b, nil
}

type persistEntry struct {
	T   string `json:"t"`
	OK  bool   `json:"ok"`
	U   uint64 `json:"u,omitempty"`
	Tot uint64 `json:"tot,omitempty"`
	Msg string `json:"msg,omitempty"`
	Lat string `json:"lat"`
}

func entryToPersist(e Entry) persistEntry {
	return persistEntry{
		T:   e.Time.UTC().Format(time.RFC3339Nano),
		OK:  e.OK,
		U:   e.Used,
		Tot: e.Total,
		Msg: e.Message,
		Lat: e.Latency.String(),
	}
}

func entryFromPersist(p persistEntry) (Entry, error) {
	t, err := time.Parse(time.RFC3339Nano, p.T)
	if err != nil {
		t, err = time.Parse(time.RFC3339, p.T)
		if err != nil {
			return Entry{}, err
		}
	}
	d, err := time.ParseDuration(p.Lat)
	if err != nil {
		d = 0
	}
	return Entry{
		Time:    t.In(time.Local),
		OK:      p.OK,
		Used:    p.U,
		Total:   p.Tot,
		Message: p.Msg,
		Latency: d,
	}, nil
}

func (b *Buffer) loadFromFileLocked() error {
	if b.path == "" {
		return nil
	}
	data, err := os.ReadFile(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	var loaded []Entry
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var p persistEntry
		if err := json.Unmarshal(line, &p); err != nil {
			continue
		}
		ent, err := entryFromPersist(p)
		if err != nil {
			continue
		}
		loaded = append(loaded, ent)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].Time.Before(loaded[j].Time)
	})
	b.entries = loaded
	return nil
}

func (b *Buffer) persistLocked() {
	if b.path == "" {
		return
	}
	_ = b.saveToFileLocked()
}

func (b *Buffer) saveToFileLocked() error {
	if b.path == "" {
		return nil
	}
	var buf bytes.Buffer
	for _, e := range b.entries {
		line, err := json.Marshal(entryToPersist(e))
		if err != nil {
			return err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	_ = os.Remove(b.path)
	if err := os.Rename(tmp, b.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SetPath changes the on-disk log file. If the file exists and is non-empty, entries are
// replaced from disk; otherwise current in-memory entries are kept and written to the new path.
func (b *Buffer) SetPath(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.path = path
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	st, err := os.Stat(path)
	if err == nil && st.Size() > 0 {
		if err := b.loadFromFileLocked(); err != nil {
			return err
		}
		b.pruneLocked(time.Now())
	}
	return b.saveToFileLocked()
}
