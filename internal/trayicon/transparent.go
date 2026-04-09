package trayicon

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"fyne.io/fyne/v2"
)

var (
	transparentOnce sync.Once
	transparentRes  fyne.Resource
)

// TransparentTrayIcon returns a small fully transparent PNG so the tray slot stays clickable
// but shows no visible glyph (idle state when a custom icon is disabled).
func TransparentTrayIcon() fyne.Resource {
	transparentOnce.Do(func() {
		const n = 16
		img := image.NewNRGBA(image.Rect(0, 0, n, n))
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		transparentRes = fyne.NewStaticResource("tray_transparent.png", buf.Bytes())
	})
	return transparentRes
}
