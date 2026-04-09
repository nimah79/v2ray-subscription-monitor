package assets

import (
	_ "embed"
	"sync"

	"fyne.io/fyne/v2"
)

//go:embed icons/v2ray.svg
var V2RaySVG []byte

//go:embed icons/v2ray-subscription-monitor.png
var appIconPNG []byte

var (
	appIconOnce sync.Once
	appIcon     fyne.Resource
)

// AppIcon returns the embedded application icon (dock / taskbar / window).
func AppIcon() fyne.Resource {
	appIconOnce.Do(func() {
		appIcon = fyne.NewStaticResource("v2ray-subscription-monitor.png", appIconPNG)
	})
	return appIcon
}
