package logbuf

import (
	"sync"
	"time"
)

// Entry is one probe log line.
type Entry struct {
	Time    time.Time
	OK      bool
	Used    uint64
	Total   uint64
	Message string
	Latency time.Duration
}

// Buffer keeps a bounded ring of entries and applies age/size retention.
type Buffer struct {
	mu         sync.Mutex
	entries    []Entry
	maxEntries int
	maxAge     time.Duration // 0 = no age limit
	path       string        // empty = in-memory only (no disk I/O)
}

func New(maxEntries int, maxAge time.Duration) *Buffer {
	if maxEntries < 1 {
		maxEntries = 100
	}
	return &Buffer{
		entries:    make([]Entry, 0, maxEntries+8),
		maxEntries: maxEntries,
		maxAge:     maxAge,
		path:       "",
	}
}

func (b *Buffer) SetPolicy(maxEntries int, maxAge time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if maxEntries < 1 {
		maxEntries = 100
	}
	b.maxEntries = maxEntries
	b.maxAge = maxAge
	b.pruneLocked(time.Now())
	b.persistLocked()
}

func (b *Buffer) Append(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.entries = append(b.entries, e)
	b.pruneLocked(now)
	b.persistLocked()
}

func (b *Buffer) Snapshot() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, len(b.entries))
	copy(out, b.entries)
	return out
}

func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = b.entries[:0]
	b.persistLocked()
}

func (b *Buffer) pruneLocked(now time.Time) {
	if b.maxAge > 0 {
		i := 0
		cutoff := now.Add(-b.maxAge)
		for i < len(b.entries) && b.entries[i].Time.Before(cutoff) {
			i++
		}
		if i > 0 {
			b.entries = append([]Entry(nil), b.entries[i:]...)
		}
	}
	over := len(b.entries) - b.maxEntries
	if over > 0 {
		b.entries = b.entries[over:]
	}
}
