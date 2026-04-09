//go:build darwin

package platform

/*
#cgo CFLAGS: -Wall -Wno-unused-parameter
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework Carbon

void platform_register_did_become_active(void);
*/
import "C"

import (
	"sync"
	"sync/atomic"
)

//export platformAppDidBecomeActiveGo
func platformAppDidBecomeActiveGo() {
	becomeMu.Lock()
	cb := becomeCb
	becomeMu.Unlock()
	if cb != nil {
		cb()
	}
}

var (
	becomeMu         sync.Mutex
	becomeCb         func()
	becomeRegistered atomic.Bool
)

// SetOnApplicationDidBecomeActive registers a callback when the user activates the app via the Dock
// (kAEReopenApplication), not for every NSApplicationDidBecomeActive (e.g. NSAlert dismiss).
// Passing nil clears the callback; the notification observer is registered once.
func SetOnApplicationDidBecomeActive(fn func()) {
	becomeMu.Lock()
	becomeCb = fn
	becomeMu.Unlock()
	if fn != nil && !becomeRegistered.Swap(true) {
		C.platform_register_did_become_active()
	}
}
