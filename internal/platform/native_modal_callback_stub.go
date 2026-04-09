//go:build !darwin

package platform

// SetAfterNativeModal is a no-op outside macOS.
func SetAfterNativeModal(fn func()) {}
