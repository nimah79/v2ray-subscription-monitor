//go:build !darwin

package platform

// SetOnApplicationDidBecomeActive is a no-op on non-macOS platforms.
func SetOnApplicationDidBecomeActive(fn func()) {}
