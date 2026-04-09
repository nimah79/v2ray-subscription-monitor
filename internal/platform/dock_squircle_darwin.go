//go:build darwin

package platform

import (
	"bytes"
	"image"
	"image/png"
	"math"
)

const dockIconRaster = 512

// squircleDockIconPNG scales the image to dockIconRaster and applies a superellipse (n≈4) alpha
// mask so the Dock tile matches the rounded “squircle” look of other macOS app icons.
func squircleDockIconPNG(data []byte) []byte {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	sb := src.Bounds()
	if sb.Dx() <= 0 || sb.Dy() <= 0 {
		return nil
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dockIconRaster, dockIconRaster))
	sx0, sy0 := sb.Min.X, sb.Min.Y
	sw, sh := sb.Dx(), sb.Dy()
	for y := 0; y < dockIconRaster; y++ {
		sy := sy0 + y*sh/dockIconRaster
		for x := 0; x < dockIconRaster; x++ {
			sx := sx0 + x*sw/dockIconRaster
			dst.Set(x, y, src.At(sx, sy))
		}
	}

	c := float64(dockIconRaster-1) / 2
	edge := 0.07 // soft rim in normalized superellipse space
	for y := 0; y < dockIconRaster; y++ {
		for x := 0; x < dockIconRaster; x++ {
			nx := (float64(x) - c) / c
			ny := (float64(y) - c) / c
			d := math.Pow(math.Abs(nx), 4) + math.Pow(math.Abs(ny), 4)

			i := y*dst.Stride + x*4
			a0 := float64(dst.Pix[i+3]) / 255

			var f float64
			switch {
			case d >= 1:
				f = 0
			case d > 1-edge:
				f = (1 - d) / edge
			default:
				f = 1
			}

			na := uint8(math.Round(a0 * f * 255))
			if na == 0 {
				dst.Pix[i] = 0
				dst.Pix[i+1] = 0
				dst.Pix[i+2] = 0
				dst.Pix[i+3] = 0
				continue
			}
			dst.Pix[i+3] = na
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil
	}
	return buf.Bytes()
}
