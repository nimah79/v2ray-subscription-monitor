//go:build !darwin

package platform

// EnsureNSApplication is a no-op on non-macOS platforms.
func EnsureNSApplication() {}

// SetDockIconFromPNG is a no-op on non-macOS platforms.
func SetDockIconFromPNG([]byte) {}
