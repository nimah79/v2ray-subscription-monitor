//go:build android || ios || (!darwin && !windows && !linux && !freebsd && !openbsd && !netbsd)

package trayquit

// TearDownSystrayForExternalLoop is a no-op on platforms without a native systray implementation.
func TearDownSystrayForExternalLoop() {}
