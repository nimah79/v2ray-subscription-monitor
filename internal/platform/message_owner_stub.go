//go:build !windows

package platform

// SetWindowsMessageOwner is implemented in native_message_windows.go.
func SetWindowsMessageOwner(hwnd uintptr) {}
