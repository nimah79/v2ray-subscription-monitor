package main

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"v2ray-subscription-data-usage-monitor/assets"
	"v2ray-subscription-data-usage-monitor/internal/logbuf"
	"v2ray-subscription-data-usage-monitor/internal/platform"
	"v2ray-subscription-data-usage-monitor/internal/subscription"
	"v2ray-subscription-data-usage-monitor/internal/trayicon"
	"v2ray-subscription-data-usage-monitor/internal/trayquit"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/systray"
)

const prefURL = "subscription_url"
const prefInterval = "interval_seconds"
const prefWarnAmount = "warn_threshold_amount"
const prefWarnUnit = "warn_threshold_unit"
const prefMaxLogs = "max_log_entries"
const prefMaxAgeH = "max_log_age_hours"
const prefTimeout = "request_timeout_seconds"
const prefTrayV2rayIcon = "tray_v2ray_style_icon"
const prefFailAlertAfter = "fail_alert_after_consecutive"
const prefLogFilePath = "log_file_path"

const appID = "io.github.v2ray-subscription-data-usage-monitor"

// appTitle is the human-readable name: window title, tray menu header, and app metadata (taskbar/dock where supported).
const appTitle = "V2Ray Subscription Monitor"

// darwinMainInTray: main window was closed to tray; Dock / activation must call showMainWindow (GLFW has no focused window).
var (
	darwinDockRestoreMu sync.Mutex
	darwinMainInTray    bool
)

type fetchState struct {
	mu         sync.Mutex
	fetching   bool
	lastErr    string
	used       uint64
	total      uint64
	expireUnix int64
}

func (s *fetchState) snapshot() (fetching bool, err string, used, total uint64, exp int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fetching, s.lastErr, s.used, s.total, s.expireUnix
}

func formatBytes(n uint64) string {
	if n == 0 {
		return "0 B"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit && exp < 4; v /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}[exp]
	return fmt.Sprintf("%.2f %s", float64(n)/float64(div), suffix)
}

// formatBytesCompact removes spaces from formatBytes for compact tray text.
func formatBytesCompact(n uint64) string {
	return strings.ReplaceAll(formatBytes(n), " ", "")
}

// parseWarnThresholdBytes converts a positive amount and unit "MB" or "GB" to bytes. Empty/zero amount => 0, true.
func parseWarnThresholdBytes(amountStr, unit string) (uint64, bool) {
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" || amountStr == "0" {
		return 0, true
	}
	v, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "GB":
		const maxGB = float64(1 << 20) // 1M GiB — sanity cap
		if v > maxGB {
			return 0, false
		}
		b := v * 1024 * 1024 * 1024
		if b >= float64(1<<63) {
			return 0, false
		}
		return uint64(b), true
	case "MB":
		const maxMB = float64(1 << 30) // large sanity cap
		if v > maxMB {
			return 0, false
		}
		b := v * 1024 * 1024
		if b >= float64(1<<63) {
			return 0, false
		}
		return uint64(b), true
	default:
		return 0, false
	}
}

func formatExpire(unix int64) string {
	// Many providers omit unknown expiry; some send 0 for "none".
	if unix <= 0 {
		return "—"
	}
	t := time.Unix(unix, 0).UTC()
	return t.Format(time.RFC3339)
}

func main() {
	app.SetMetadata(fyne.AppMetadata{
		ID:   appID,
		Name: appTitle,
	})
	a := app.NewWithID(appID)
	appIcon := assets.AppIcon()
	a.SetIcon(appIcon)

	platform.EnsureNSApplication()
	platform.SetDockIconFromPNG(appIcon.Content())

	applyDockIcon := func() {
		platform.SetDockIconFromPNG(appIcon.Content())
	}
	a.Lifecycle().SetOnStarted(func() {
		applyDockIcon()
		// One frame later: systray init can briefly replace the Dock tile before OnStarted runs.
		go func() {
			time.Sleep(24 * time.Millisecond)
			fyne.Do(applyDockIcon)
		}()
	})

	w := a.NewWindow(appTitle)
	w.SetIcon(appIcon)

	var logsWindow fyne.Window

	showMainWindow := func() {
		if runtime.GOOS == "darwin" {
			darwinDockRestoreMu.Lock()
			darwinMainInTray = false
			darwinDockRestoreMu.Unlock()
		}
		platform.SetTrayOnlyMode(false)
		applyDockIcon()
		w.Show()
		w.RequestFocus()
	}
	hideMainToTray := func() {
		if runtime.GOOS == "darwin" {
			darwinDockRestoreMu.Lock()
			darwinMainInTray = true
			darwinDockRestoreMu.Unlock()
		}
		if logsWindow != nil {
			logsWindow.Hide()
		}
		w.Hide()
		platform.SetTrayOnlyMode(true)
	}

	prefs := a.Preferences()
	state := &fetchState{}
	timeoutSec := prefs.IntWithFallback(prefTimeout, 45)
	if timeoutSec < 1 {
		timeoutSec = 45
	}
	httpClient := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	initMaxAgeH := prefs.IntWithFallback(prefMaxAgeH, 0)
	var initMaxAge time.Duration
	if initMaxAgeH > 0 {
		initMaxAge = time.Duration(initMaxAgeH) * time.Hour
	}

	defaultLogPath, defaultLogPathErr := logbuf.DefaultLogFilePath()
	logPathEntry := widget.NewEntry()
	logPathEntry.SetText(strings.TrimSpace(prefs.String(prefLogFilePath)))
	if defaultLogPathErr == nil {
		logPathEntry.SetPlaceHolder(defaultLogPath)
	} else {
		logPathEntry.SetPlaceHolder("subscription-requests.jsonl")
	}

	maxLogInit := prefs.IntWithFallback(prefMaxLogs, 500)
	startLogPath := strings.TrimSpace(logPathEntry.Text)
	if startLogPath == "" && defaultLogPathErr == nil {
		startLogPath = defaultLogPath
	}
	var logs *logbuf.Buffer
	if startLogPath != "" {
		var err error
		logs, err = logbuf.NewPersistent(maxLogInit, initMaxAge, startLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "v2ray subscription monitor: log file %q: %v\n", startLogPath, err)
			logs = logbuf.New(maxLogInit, initMaxAge)
		}
	} else {
		if defaultLogPathErr != nil {
			fmt.Fprintf(os.Stderr, "v2ray subscription monitor: default log path: %v\n", defaultLogPathErr)
		}
		logs = logbuf.New(maxLogInit, initMaxAge)
	}

	urlEntry := widget.NewEntry()
	urlEntry.SetText(prefs.String(prefURL))
	urlEntry.SetPlaceHolder("https://…")

	intervalEntry := widget.NewEntry()
	intervalEntry.SetText(strconv.Itoa(prefs.IntWithFallback(prefInterval, 300)))
	intervalEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		if n < 5 {
			return fmt.Errorf("minimum 5 seconds")
		}
		if n > 86400 {
			return fmt.Errorf("maximum 86400 seconds (24h)")
		}
		return nil
	}

	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetText(strconv.Itoa(prefs.IntWithFallback(prefTimeout, 45)))
	timeoutEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		if n < 1 {
			return fmt.Errorf("minimum 1 second")
		}
		if n > 3600 {
			return fmt.Errorf("maximum 3600 seconds (1h)")
		}
		return nil
	}

	warnUnitSelect := widget.NewSelect([]string{"MB", "GB"}, nil)
	// Fyne Select MinSize uses PlaceHolder width; default "(Select one)" is much wider than "MB"/"GB".
	warnUnitSelect.PlaceHolder = "GB"

	warnAmountEntry := widget.NewEntry()
	warnAmountEntry.SetPlaceHolder("0 = off")
	if a := prefs.FloatWithFallback(prefWarnAmount, 0); a > 0 {
		warnAmountEntry.SetText(strconv.FormatFloat(a, 'f', -1, 64))
	}
	warnAmountEntry.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" || s == "0" {
			return nil
		}
		_, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("enter a number, or 0 to disable")
		}
		_, ok := parseWarnThresholdBytes(s, warnUnitSelect.Selected)
		if !ok {
			return fmt.Errorf("invalid amount for %s", warnUnitSelect.Selected)
		}
		return nil
	}

	warnUnit := prefs.StringWithFallback(prefWarnUnit, "GB")
	if warnUnit != "MB" && warnUnit != "GB" {
		warnUnit = "GB"
	}
	warnUnitSelect.OnChanged = func(string) {
		_ = warnAmountEntry.Validate()
	}
	warnUnitSelect.SetSelected(warnUnit)

	maxLogEntry := widget.NewEntry()
	maxLogEntry.SetText(strconv.Itoa(prefs.IntWithFallback(prefMaxLogs, 500)))
	maxLogEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		if n < 10 || n > 100000 {
			return fmt.Errorf("use 10–100000")
		}
		return nil
	}

	maxAgeEntry := widget.NewEntry()
	maxAgeEntry.SetPlaceHolder("0 = unlimited by age")
	maxAgeEntry.SetText(strconv.Itoa(prefs.IntWithFallback(prefMaxAgeH, 0)))
	maxAgeEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		if n < 0 || n > 24*365 {
			return fmt.Errorf("invalid hours")
		}
		return nil
	}

	failAlertEntry := widget.NewEntry()
	failAlertEntry.SetPlaceHolder("0 = off")
	failAlertEntry.SetText(strconv.Itoa(prefs.IntWithFallback(prefFailAlertAfter, 3)))
	failAlertEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		if n < 0 || n > 10000 {
			return fmt.Errorf("use 0 (off) or 1–10000")
		}
		return nil
	}

	v2rayTrayCheck := widget.NewCheck("V2Ray-style tray icon", nil)
	v2rayTrayCheck.Checked = prefs.BoolWithFallback(prefTrayV2rayIcon, true)

	usedLabel := widget.NewLabel("Used: —")
	totalLabel := widget.NewLabel("Total: —")
	expireLabel := widget.NewLabel("Expiry (UTC): —")
	progress := widget.NewProgressBar()
	progress.TextFormatter = func() string {
		d := progress.Max - progress.Min
		if d <= 0 {
			return "0.0%"
		}
		pct := (progress.Value - progress.Min) / d * 100
		return fmt.Sprintf("%.1f%%", pct)
	}
	progress.Hide()
	statusLabel := widget.NewLabel("")

	var logRows []string
	logList := widget.NewList(
		func() int { return len(logRows) },
		func() fyne.CanvasObject { return widget.NewLabel("log") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(logRows[i])
			o.(*widget.Label).Wrapping = fyne.TextWrapOff
		},
	)

	logsWindow = a.NewWindow("Request log")
	logsWindow.Resize(fyne.NewSize(920, 680))
	logsWindow.SetIcon(w.Icon())

	var warningLatch bool
	var consecutiveFails int
	var failAlertLatch bool
	var failStreakMu sync.Mutex
	var pollCancel context.CancelFunc
	var pollMu sync.Mutex

	stopPolling := func() {
		pollMu.Lock()
		if pollCancel != nil {
			pollCancel()
			pollCancel = nil
		}
		pollMu.Unlock()
	}

	// Fyne's GLFW driver only runs trayStop (systray nativeEnd + Quit) from driver.Quit when
	// curWindow != nil. After hiding the main window to the tray, GLFW focus can be nil, so
	// App.Quit would skip systray teardown. Mirror RunWithExternalLoop's end callback here,
	// on the UI thread, before Quit (trayquit is idempotent if the driver also runs trayStop).
	quitApplication := func() {
		stopPolling()
		platform.SetTrayOnlyModeSync(false)
		fyne.Do(func() {
			if logsWindow != nil {
				logsWindow.Hide()
			}
			w.Show()
			w.RequestFocus()
			trayquit.TearDownSystrayForExternalLoop()
			a.Quit()
		})
	}

	trayMenu := func() *fyne.Menu {
		titleItem := fyne.NewMenuItem(appTitle, nil)
		titleItem.Disabled = true
		return fyne.NewMenu("",
			titleItem,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("Settings", theme.SettingsIcon(), func() { showMainWindow() }),
			fyne.NewMenuItemWithIcon("Quit", theme.LogoutIcon(), func() { quitApplication() }),
		)
	}

	refreshLogs := func() {
		ents := logs.Snapshot()
		logRows = logRows[:0]
		for _, e := range ents {
			ts := e.Time.Local().Format(time.RFC3339Nano)
			if e.OK {
				logRows = append(logRows, fmt.Sprintf("%s  %s  used %s  total %s  %s",
					ts, "OK", formatBytes(e.Used), formatBytes(e.Total), e.Latency))
			} else {
				logRows = append(logRows, fmt.Sprintf("%s  ERR  %s  %s",
					ts, e.Message, e.Latency))
			}
		}
		logList.Refresh()
	}

	updateSummary := func() {
		fetching, errStr, used, total, exp := state.snapshot()
		// Fetching and errors use theme icons in the tray only (no emoji in labels).
		usedLabel.SetText("Used: " + formatBytes(used))
		totalLabel.SetText("Total: " + formatBytes(total))
		expireLabel.SetText("Expiry (UTC): " + formatExpire(exp))

		if total > 0 {
			progress.Show()
			max := float64(total)
			if float64(used) > max {
				max = float64(used)
			}
			progress.Max = max
			progress.SetValue(float64(used))
		} else {
			progress.Hide()
		}
		if errStr != "" && !fetching {
			statusLabel.SetText("Last error: " + errStr)
		} else if fetching {
			statusLabel.SetText("Fetching…")
		} else {
			statusLabel.SetText("")
		}
	}

	updateSystray := func() {
		desk, ok := a.(desktop.App)
		if !ok {
			return
		}
		fetching, errStr, used, total, _ := state.snapshot()
		u := formatBytesCompact(used)
		t := formatBytesCompact(total)
		title := u + "/" + t

		// Visible quota without opening the menu: title beside the icon (macOS, Linux).
		// Windows has no tray title; SetTooltip shows the same text on hover.
		systray.SetTitle(title)
		tooltip := title
		if fetching {
			tooltip = "Updating… " + title
		} else if errStr != "" {
			tooltip = title + " — " + errStr
		}
		systray.SetTooltip(tooltip)

		desk.SetSystemTrayMenu(trayMenu())

		switch {
		case fetching:
			desk.SetSystemTrayIcon(theme.ViewRefreshIcon())
		case errStr != "":
			desk.SetSystemTrayIcon(theme.ErrorIcon())
		default:
			if v2rayTrayCheck.Checked {
				desk.SetSystemTrayIcon(trayicon.V2RayStyle())
			} else {
				desk.SetSystemTrayIcon(trayicon.TransparentTrayIcon())
			}
		}
	}

	runOnMain := func(fn func()) {
		fyne.Do(fn)
	}

	v2rayTrayCheck.OnChanged = func(bool) { runOnMain(updateSystray) }

	a.Settings().AddListener(func(fyne.Settings) {
		trayicon.InvalidateCache()
		runOnMain(updateSystray)
	})

	evalWarning := func(used, total uint64, thresholdBytes uint64) {
		if thresholdBytes == 0 {
			return
		}
		if used >= thresholdBytes {
			if !warningLatch {
				warningLatch = true
				msg := fmt.Sprintf("Used data is %s (warning threshold %s). Total quota %s.",
					formatBytes(used), formatBytes(thresholdBytes), formatBytes(total))
				dialog.ShowInformation("Usage warning", msg, w)
			}
		} else {
			warningLatch = false
		}
	}

	doFetch := func() {
		url := strings.TrimSpace(urlEntry.Text)
		if url == "" {
			return
		}
		warnThresh, _ := parseWarnThresholdBytes(warnAmountEntry.Text, warnUnitSelect.Selected)
		if timeoutEntry.Validate() == nil {
			tv, _ := strconv.Atoi(strings.TrimSpace(timeoutEntry.Text))
			httpClient.Timeout = time.Duration(tv) * time.Second
		}

		state.mu.Lock()
		state.fetching = true
		state.mu.Unlock()
		runOnMain(func() {
			updateSummary()
			updateSystray()
		})

		start := time.Now()
		res := subscription.Fetch(url, httpClient)
		dur := time.Since(start)

		state.mu.Lock()
		state.fetching = false
		if res.Err != nil {
			state.lastErr = res.Err.Error()
		} else {
			state.lastErr = ""
			state.used = res.Stats.Used()
			state.total = res.Stats.Total
			state.expireUnix = res.Stats.Expire
		}
		state.mu.Unlock()

		ent := logbuf.Entry{Time: time.Now(), Latency: dur}
		if res.Err != nil {
			ent.OK = false
			ent.Message = res.Err.Error()
			if res.StatusCode != 0 {
				ent.Message = fmt.Sprintf("%s (HTTP %d)", ent.Message, res.StatusCode)
			}
		} else {
			ent.OK = true
			ent.Used = res.Stats.Used()
			ent.Total = res.Stats.Total
			ent.Message = "ok"
		}
		logs.Append(ent)

		failThr := prefs.IntWithFallback(prefFailAlertAfter, 3)
		if n, err := strconv.Atoi(strings.TrimSpace(failAlertEntry.Text)); err == nil && n >= 0 {
			failThr = n
		}
		var showFailAlert bool
		var failAlertMsg string
		failStreakMu.Lock()
		if res.Err != nil {
			consecutiveFails++
			if failThr > 0 && consecutiveFails >= failThr && !failAlertLatch {
				failAlertLatch = true
				showFailAlert = true
				failAlertMsg = fmt.Sprintf("The subscription update request failed %d times in a row.\n\nLast error: %s",
					consecutiveFails, ent.Message)
			}
		} else {
			consecutiveFails = 0
			failAlertLatch = false
		}
		failStreakMu.Unlock()

		runOnMain(func() {
			refreshLogs()
			updateSummary()
			updateSystray()
			if res.Err == nil {
				evalWarning(res.Stats.Used(), res.Stats.Total, warnThresh)
			} else if showFailAlert {
				dialog.ShowInformation("Update failed", failAlertMsg, w)
			}
		})
	}

	applyPrefs := func() error {
		if err := intervalEntry.Validate(); err != nil {
			return err
		}
		if err := timeoutEntry.Validate(); err != nil {
			return err
		}
		if err := warnAmountEntry.Validate(); err != nil {
			return err
		}
		if err := maxLogEntry.Validate(); err != nil {
			return err
		}
		if err := maxAgeEntry.Validate(); err != nil {
			return err
		}
		if err := failAlertEntry.Validate(); err != nil {
			return err
		}
		iv, _ := strconv.Atoi(strings.TrimSpace(intervalEntry.Text))
		tv, _ := strconv.Atoi(strings.TrimSpace(timeoutEntry.Text))
		warnAmtStr := strings.TrimSpace(warnAmountEntry.Text)
		var warnAmt float64
		if warnAmtStr != "" && warnAmtStr != "0" {
			warnAmt, _ = strconv.ParseFloat(warnAmtStr, 64)
		}
		mv, _ := strconv.Atoi(strings.TrimSpace(maxLogEntry.Text))
		av, _ := strconv.Atoi(strings.TrimSpace(maxAgeEntry.Text))
		fv, _ := strconv.Atoi(strings.TrimSpace(failAlertEntry.Text))
		lp := strings.TrimSpace(logPathEntry.Text)

		prefs.SetString(prefURL, urlEntry.Text)
		prefs.SetInt(prefInterval, iv)
		prefs.SetInt(prefTimeout, tv)
		prefs.SetFloat(prefWarnAmount, warnAmt)
		prefs.SetString(prefWarnUnit, warnUnitSelect.Selected)
		prefs.SetInt(prefMaxLogs, mv)
		prefs.SetInt(prefMaxAgeH, av)
		prefs.SetInt(prefFailAlertAfter, fv)
		prefs.SetBool(prefTrayV2rayIcon, v2rayTrayCheck.Checked)
		prefs.SetString(prefLogFilePath, lp)

		resolvedLogPath := lp
		if resolvedLogPath == "" {
			if defaultLogPathErr != nil {
				return fmt.Errorf("default log file path is unavailable: %w", defaultLogPathErr)
			}
			resolvedLogPath = defaultLogPath
		}
		if err := logs.SetPath(resolvedLogPath); err != nil {
			return fmt.Errorf("log file: %w", err)
		}

		var maxAge time.Duration
		if av > 0 {
			maxAge = time.Duration(av) * time.Hour
		}
		logs.SetPolicy(mv, maxAge)
		httpClient.Timeout = time.Duration(tv) * time.Second
		refreshLogs()
		runOnMain(updateSystray)
		return nil
	}

	startPolling := func() {
		if err := applyPrefs(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		iv, _ := strconv.Atoi(strings.TrimSpace(intervalEntry.Text))

		pollMu.Lock()
		if pollCancel != nil {
			pollCancel()
			pollCancel = nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		pollCancel = cancel
		pollMu.Unlock()

		go func() {
			doFetch()
			tick := time.NewTicker(time.Duration(iv) * time.Second)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					doFetch()
				}
			}
		}()
	}

	applyBtn := widget.NewButton("Apply (save & poll)", func() { startPolling() })
	fetchNowBtn := widget.NewButton("Fetch once", func() { go doFetch() })
	clearLogBtn := widget.NewButton("Clear logs", func() {
		logs.Clear()
		refreshLogs()
	})
	closeLogsBtn := widget.NewButton("Close", func() { logsWindow.Hide() })
	showLogsBtn := widget.NewButton("Logs", func() {
		refreshLogs()
		logsWindow.Show()
		logsWindow.RequestFocus()
	})

	browseLogPathBtn := widget.NewButton("Browse…", func() {
		d := dialog.NewFileSave(func(out fyne.URIWriteCloser, err error) {
			if err != nil || out == nil {
				return
			}
			u := out.URI()
			_ = out.Close()
			if u != nil && u.Scheme() == "file" {
				logPathEntry.SetText(u.Path())
			}
		}, w)
		name := "subscription-requests.jsonl"
		if defaultLogPathErr == nil {
			name = filepath.Base(defaultLogPath)
		}
		d.SetFileName(name)
		d.Show()
	})

	logsWindow.SetContent(container.NewBorder(
		widget.NewLabel("Newest entries at the bottom."),
		container.NewHBox(clearLogBtn, closeLogsBtn, layout.NewSpacer()),
		nil, nil,
		container.NewVScroll(logList),
	))

	subscriptionBlock := container.NewVBox(
		widget.NewLabelWithStyle("Subscription URL", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		urlEntry,
	)
	settingsGrid := container.NewGridWithColumns(3,
		container.NewVBox(widget.NewLabel("Interval (s)"), intervalEntry),
		container.NewVBox(widget.NewLabel("Timeout (s)"), timeoutEntry),
		container.NewVBox(
			widget.NewLabel("Warn used ≥ (0=off)"),
			container.NewBorder(nil, nil, nil, warnUnitSelect, warnAmountEntry),
		),
		container.NewVBox(widget.NewLabel("Max log entries"), maxLogEntry),
		container.NewVBox(widget.NewLabel("Log max age (h, 0=∞)"), maxAgeEntry),
		container.NewVBox(widget.NewLabel("Alert after N fails (0=off)"), failAlertEntry),
	)
	logFileRow := container.NewVBox(
		widget.NewLabel("Log file (blank = default path, persisted across restarts)"),
		container.NewBorder(nil, nil, nil, browseLogPathBtn, logPathEntry),
	)
	formTop := container.NewVBox(
		subscriptionBlock,
		settingsGrid,
		logFileRow,
		v2rayTrayCheck,
		container.NewHBox(applyBtn, fetchNowBtn, showLogsBtn, layout.NewSpacer()),
	)

	// While fetching (total==0), the progress bar is hidden; when data arrives it shows and
	// is taller than one status line. Stack a same-size transparent fill under the bar so
	// layout height stays stable without extra empty space below the status line.
	progressSlotFill := canvas.NewRectangle(color.Transparent)
	progressSlotFill.SetMinSize(progress.MinSize())
	progressSlot := container.NewStack(progressSlotFill, progress)

	summary := container.NewVBox(
		container.NewGridWithColumns(3, usedLabel, totalLabel, expireLabel),
		progressSlot,
		statusLabel,
	)

	mainCol := container.NewVBox(formTop, summary)
	scroll := container.NewVScroll(mainCol)
	colMin := mainCol.MinSize()
	const prefWinWidth float32 = 760
	innerW := fyne.Max(colMin.Width, prefWinWidth)
	// Do not set the scroll viewport's min height to the full form — that blocks shrinking
	// and disables vertical scroll. Keep a modest floor so the window can still resize smaller.
	const minScrollViewH float32 = 200
	scroll.SetMinSize(fyne.NewSize(fyne.Max(colMin.Width, 360), minScrollViewH))
	w.SetContent(scroll)
	pad := theme.Padding() * 2
	w.Resize(fyne.NewSize(innerW+pad, colMin.Height+pad))
	w.CenterOnScreen()

	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayMenu(trayMenu())
	}

	w.SetCloseIntercept(func() {
		hideMainToTray()
	})

	// Quit: Cmd+Q (macOS) / Alt+F4 (Windows & Linux). Close window: Cmd+W (macOS) / Ctrl+W (elsewhere).
	registerWindowShortcuts := func(win fyne.Window, onCloseWindow func()) {
		c := win.Canvas()
		if runtime.GOOS == "darwin" {
			c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) {
				quitApplication()
			})
		} else {
			c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF4, Modifier: fyne.KeyModifierAlt}, func(fyne.Shortcut) {
				quitApplication()
			})
		}
		c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierShortcutDefault}, func(fyne.Shortcut) {
			onCloseWindow()
		})
	}
	registerWindowShortcuts(w, hideMainToTray)
	registerWindowShortcuts(logsWindow, func() { logsWindow.Hide() })

	go func() {
		time.Sleep(200 * time.Millisecond)
		runOnMain(updateSystray)
	}()

	w.SetOnDropped(func(_ fyne.Position, u []fyne.URI) {
		if len(u) == 0 {
			return
		}
		urlEntry.SetText(u[0].String())
	})

	// Auto-start if URL was saved
	if strings.TrimSpace(urlEntry.Text) != "" {
		go func() {
			time.Sleep(100 * time.Millisecond)
			runOnMain(startPolling)
		}()
	} else {
		runOnMain(func() {
			refreshLogs()
			updateSummary()
			updateSystray()
		})
	}

	if runtime.GOOS == "darwin" {
		platform.SetOnApplicationDidBecomeActive(func() {
			fyne.Do(func() {
				darwinDockRestoreMu.Lock()
				need := darwinMainInTray
				darwinDockRestoreMu.Unlock()
				if need {
					showMainWindow()
				}
			})
		})
	}

	w.ShowAndRun()
	stopPolling()
}
