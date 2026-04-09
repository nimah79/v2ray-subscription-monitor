//go:build darwin

package platform

/*
#cgo CFLAGS: -Wall -Wno-unused-parameter
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework CoreFoundation

#include <stdlib.h>
void platform_show_native_info(const char *title, const char *message);
*/
import "C"

import (
	"sync"
	"unsafe"
)

//export goAfterNativeModalDialog
func goAfterNativeModalDialog() {
	afterNativeMu.Lock()
	fn := afterNativeCb
	afterNativeMu.Unlock()
	if fn != nil {
		fn()
	}
}

var (
	afterNativeMu sync.Mutex
	afterNativeCb func()
)

// SetAfterNativeModal sets a callback run on the AppKit main thread after each native modal dismisses
// (e.g. to restore NSApplicationActivationPolicyAccessory when appropriate). Only used on macOS.
func SetAfterNativeModal(fn func()) {
	afterNativeMu.Lock()
	afterNativeCb = fn
	afterNativeMu.Unlock()
}

// ShowNativeInfo shows a modal warning dialog (NSAlert). Safe from any goroutine; marshals to AppKit main.
func ShowNativeInfo(title, message string) bool {
	if title == "" {
		title = " "
	}
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))
	C.platform_show_native_info(ct, cm)
	return true
}
