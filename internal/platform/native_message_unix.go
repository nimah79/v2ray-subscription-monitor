//go:build !darwin && !windows

package platform

import (
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
)

var zenityX11Parent atomic.Uint64

// SetUnixZenityParentX11 sets an optional X11 window ID for zenity --attach (transient for centering).
// Clear with 0 on Wayland or when the main window is hidden to tray.
func SetUnixZenityParentX11(wid uint64) {
	zenityX11Parent.Store(wid)
}

// ShowNativeInfo tries zenity or kdialog (common on Linux desktop). Returns false if unavailable.
func ShowNativeInfo(title, message string) bool {
	if p, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--warning", "--title", title, "--no-wrap", "--text", message}
		if wid := zenityX11Parent.Load(); wid != 0 && os.Getenv("WAYLAND_DISPLAY") == "" {
			args = append(args, "--attach", strconv.FormatUint(wid, 10))
		}
		cmd := exec.Command(p, args...)
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
		return true
	}
	if p, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command(p, "--title", title, "--msgbox", message)
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
		return true
	}
	return false
}
