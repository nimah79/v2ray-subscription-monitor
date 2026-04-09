//go:build !darwin

package platform

// SetTrayOnlyMode requests that the app not take a normal taskbar/Dock presence while only
// the tray is in use. Implemented for macOS; no-op elsewhere.
func SetTrayOnlyMode(trayOnly bool) {}

// SetTrayOnlyModeSync is the synchronous variant on macOS; elsewhere it matches SetTrayOnlyMode.
func SetTrayOnlyModeSync(trayOnly bool) {
	SetTrayOnlyMode(trayOnly)
}
