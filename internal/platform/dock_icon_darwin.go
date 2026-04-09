//go:build darwin

package platform

/*
#cgo CFLAGS: -Wall -Wno-unused-parameter
#cgo LDFLAGS: -framework Foundation -framework AppKit

void platform_ensure_ns_application(void);
void platform_set_dock_icon_from_png(const void *bytes, size_t len);
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// EnsureNSApplication initializes NSApplication and sets activation policy to Regular so the
// Dock tile can appear before Fyne’s GLFW/systray startup finishes.
func EnsureNSApplication() {
	C.platform_ensure_ns_application()
}

// SetDockIconFromPNG sets the macOS Dock / app switcher icon from PNG bytes.
// Fyne's app/window SetIcon does not update the Dock for a non-.app binary; this uses NSApplication.
// The image is composited with a squircle alpha mask for a standard macOS icon silhouette.
func SetDockIconFromPNG(png []byte) {
	if len(png) == 0 {
		return
	}
	out := squircleDockIconPNG(png)
	if len(out) == 0 {
		out = png
	}
	C.platform_set_dock_icon_from_png(unsafe.Pointer(&out[0]), C.size_t(len(out)))
	runtime.KeepAlive(out)
}
