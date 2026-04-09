//go:build !android && !ios && (darwin || windows || linux || freebsd || openbsd || netbsd)

package trayquit

import (
	"fyne.io/systray"
	_ "unsafe"
)

//go:linkname systrayNativeEnd fyne.io/systray.nativeEnd
func systrayNativeEnd()

// TearDownSystrayForExternalLoop mirrors fyne.io/systray.RunWithExternalLoop's end callback
// (nativeEnd + Quit). The GLFW driver only invokes that from App.Quit when curWindow != nil;
// after "close to tray" focus can be nil, so tray teardown is skipped and the process can crash
// or hang. Call from the UI thread immediately before fyne.App.Quit. Re-running when the driver
// does call trayStop is safe (Quit and exit hooks are idempotent).
func TearDownSystrayForExternalLoop() {
	systrayNativeEnd()
	systray.Quit()
}
