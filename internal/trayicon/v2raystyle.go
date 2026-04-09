package trayicon

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"

	"v2ray-subscription-data-usage-monitor/assets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const trayRasterSize = 128

var (
	iconMu   sync.Mutex
	cacheHex string
	cacheRes fyne.Resource
)

// InvalidateCache clears the rasterized tray icon so the next V2RayStyle() call rebuilds
// (e.g. after theme / light-dark changes).
func InvalidateCache() {
	iconMu.Lock()
	defer iconMu.Unlock()
	cacheHex = ""
	cacheRes = nil
}

func foregroundHex() string {
	c := theme.ForegroundColor()
	r16, g16, b16, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r16>>8), uint8(g16>>8), uint8(b16>>8))
}

func applyStrokeColor(svgSrc []byte, hex string) []byte {
	s := string(svgSrc)
	s = strings.ReplaceAll(s, `stroke="#000"`, `stroke="`+hex+`"`)
	s = strings.ReplaceAll(s, `stroke="#000000"`, `stroke="`+hex+`"`)
	return []byte(s)
}

func rasterizeV2RayPNG(hex string) ([]byte, error) {
	data := applyStrokeColor(assets.V2RaySVG, hex)
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	w, h := trayRasterSize, trayRasterSize
	vw, vh := int(icon.ViewBox.W), int(icon.ViewBox.H)
	if vw <= 0 {
		vw = w
	}
	if vh <= 0 {
		vh = h
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	icon.SetTarget(0, 0, float64(w), float64(h))

	scanner := rasterx.NewScannerGV(vw, vh, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// V2RayStyle returns a cached PNG tray icon from v2ray.svg with stroke color matching
// the current theme foreground (light on dark themes, dark on light themes).
func V2RayStyle() fyne.Resource {
	hex := foregroundHex()

	iconMu.Lock()
	defer iconMu.Unlock()

	if hex == cacheHex && cacheRes != nil {
		return cacheRes
	}

	pngBytes, err := rasterizeV2RayPNG(hex)
	if err != nil {
		return assets.AppIcon()
	}

	cacheHex = hex
	cacheRes = fyne.NewStaticResource("v2ray-tray.png", pngBytes)
	return cacheRes
}
