//go:build darwin

package platform

/*
#cgo CFLAGS: -Wall -Wno-unused-parameter
#cgo LDFLAGS: -framework Foundation -framework AppKit

void platform_apply_activation_policy(long policy);
void platform_sync_activation_policy(long policy);
*/
import "C"

// SetTrayOnlyMode uses NSApplicationActivationPolicyAccessory so the app stays out of the
// Dock while the window is closed to the menu-bar/tray. Call with false when showing the GUI again.
func SetTrayOnlyMode(trayOnly bool) {
	var p C.long
	if trayOnly {
		p = 1 // NSApplicationActivationPolicyAccessory
	} else {
		p = 0 // NSApplicationActivationPolicyRegular
	}
	C.platform_apply_activation_policy(p)
}

// SetTrayOnlyModeSync applies the activation policy immediately on the AppKit main thread
// (or synchronously from a worker thread). Use before Quit when the app must leave accessory
// mode without waiting for the next async dispatch tick.
func SetTrayOnlyModeSync(trayOnly bool) {
	var p C.long
	if trayOnly {
		p = 1
	} else {
		p = 0
	}
	C.platform_sync_activation_policy(p)
}
