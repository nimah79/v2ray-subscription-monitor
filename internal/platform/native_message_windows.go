//go:build windows

package platform

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

const mbOK = 0x0
const mbIconWarning = 0x30

var messageOwnerHWND atomic.Uintptr

// SetWindowsMessageOwner sets the HWND passed to MessageBoxW as the owner (centering, Z-order).
// Use 0 while the main window is hidden to tray so dismissing the dialog does not restore that window.
func SetWindowsMessageOwner(hwnd uintptr) {
	messageOwnerHWND.Store(hwnd)
}

// ShowNativeInfo shows a modal warning MessageBox.
func ShowNativeInfo(title, message string) bool {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return false
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return false
	}
	hwnd := messageOwnerHWND.Load()
	procMessageBoxW.Call(hwnd,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(mbOK|mbIconWarning),
	)
	return true
}
