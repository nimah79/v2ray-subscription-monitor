//go:build darwin

package platform

/*
#cgo CFLAGS: -Wall -Wno-unused-parameter
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework Carbon

void platform_register_did_become_active(void);
void platform_register_quit_apple_event(void);
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

//export platformQuitAppleEventGo
func platformQuitAppleEventGo() {
	quitMu.Lock()
	fn := quitCb
	quitMu.Unlock()
	if fn != nil {
		fn()
	}
}

var (
	becomeMu         sync.Mutex
	becomeCb         func()
	becomeRegistered atomic.Bool
	quitMu           sync.Mutex
	quitCb           func()
	quitRegistered   atomic.Bool
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

// SetQuitAppleEventHandler registers handling for Dock “Quit” and Cmd+Q (kAEQuitApplication).
// Use the same teardown as the systray Quit item; pass nil to clear.
func SetQuitAppleEventHandler(fn func()) {
	quitMu.Lock()
	quitCb = fn
	quitMu.Unlock()
	if fn != nil && !quitRegistered.Swap(true) {
		C.platform_register_quit_apple_event()
	}
}
