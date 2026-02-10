package scout

// ViewportPreset defines a device viewport configuration.
type ViewportPreset struct {
	Name             string
	Label            string
	Category         string
	Width            int
	Height           int
	DevicePixelRatio float64
}

// viewportPresets is the Go-side lookup table matching the frontend presets.
var viewportPresets = []ViewportPreset{
	// Desktop
	{Name: "desktop-hd", Label: "Desktop HD", Category: "Desktop", Width: 1920, Height: 1080, DevicePixelRatio: 1},
	{Name: "desktop-std", Label: "Desktop Standard", Category: "Desktop", Width: 1280, Height: 720, DevicePixelRatio: 1},
	{Name: "desktop-lg", Label: "Desktop Large", Category: "Desktop", Width: 2560, Height: 1440, DevicePixelRatio: 1},
	{Name: "desktop-4k", Label: "Desktop 4K", Category: "Desktop", Width: 3840, Height: 2160, DevicePixelRatio: 1},
	{Name: "laptop-13", Label: "Laptop 13\"", Category: "Desktop", Width: 1440, Height: 900, DevicePixelRatio: 2},
	{Name: "laptop-15", Label: "Laptop 15\"", Category: "Desktop", Width: 1536, Height: 864, DevicePixelRatio: 2},
	{Name: "macbook-air", Label: "MacBook Air", Category: "Desktop", Width: 1440, Height: 900, DevicePixelRatio: 2},
	{Name: "macbook-pro-16", Label: "MacBook Pro 16\"", Category: "Desktop", Width: 1728, Height: 1117, DevicePixelRatio: 2},

	// iOS — iPhone
	{Name: "iphone-16-pro-max", Label: "iPhone 16 Pro Max", Category: "iOS", Width: 440, Height: 956, DevicePixelRatio: 3},
	{Name: "iphone-16-pro", Label: "iPhone 16 Pro", Category: "iOS", Width: 402, Height: 874, DevicePixelRatio: 3},
	{Name: "iphone-16", Label: "iPhone 16", Category: "iOS", Width: 393, Height: 852, DevicePixelRatio: 3},
	{Name: "iphone-15", Label: "iPhone 15", Category: "iOS", Width: 393, Height: 852, DevicePixelRatio: 3},
	{Name: "iphone-se", Label: "iPhone SE", Category: "iOS", Width: 375, Height: 667, DevicePixelRatio: 2},
	{Name: "iphone-14-plus", Label: "iPhone 14 Plus", Category: "iOS", Width: 428, Height: 926, DevicePixelRatio: 3},
	{Name: "iphone-13-mini", Label: "iPhone 13 Mini", Category: "iOS", Width: 375, Height: 812, DevicePixelRatio: 3},

	// iOS — iPad
	{Name: "ipad-pro-12", Label: "iPad Pro 12.9\"", Category: "iOS", Width: 1024, Height: 1366, DevicePixelRatio: 2},
	{Name: "ipad-pro-11", Label: "iPad Pro 11\"", Category: "iOS", Width: 834, Height: 1194, DevicePixelRatio: 2},
	{Name: "ipad-air", Label: "iPad Air", Category: "iOS", Width: 820, Height: 1180, DevicePixelRatio: 2},
	{Name: "ipad-mini", Label: "iPad Mini", Category: "iOS", Width: 744, Height: 1133, DevicePixelRatio: 2},
	{Name: "ipad-10th", Label: "iPad 10th Gen", Category: "iOS", Width: 810, Height: 1080, DevicePixelRatio: 2},

	// Android — Phones
	{Name: "pixel-9-pro", Label: "Pixel 9 Pro", Category: "Android", Width: 412, Height: 892, DevicePixelRatio: 2.625},
	{Name: "pixel-9", Label: "Pixel 9", Category: "Android", Width: 412, Height: 892, DevicePixelRatio: 2.625},
	{Name: "samsung-s24-ultra", Label: "Samsung S24 Ultra", Category: "Android", Width: 412, Height: 915, DevicePixelRatio: 3.5},
	{Name: "samsung-s24", Label: "Samsung S24", Category: "Android", Width: 360, Height: 780, DevicePixelRatio: 3},
	{Name: "samsung-a54", Label: "Samsung A54", Category: "Android", Width: 412, Height: 915, DevicePixelRatio: 2.625},
	{Name: "oneplus-12", Label: "OnePlus 12", Category: "Android", Width: 412, Height: 915, DevicePixelRatio: 3.5},

	// Android — Tablets
	{Name: "samsung-tab-s9", Label: "Samsung Tab S9", Category: "Android", Width: 800, Height: 1280, DevicePixelRatio: 2},
	{Name: "pixel-tablet", Label: "Pixel Tablet", Category: "Android", Width: 800, Height: 1280, DevicePixelRatio: 2},
}

// DefaultViewportName is the default viewport preset name.
const DefaultViewportName = "desktop-std"

// GetViewportByName returns the viewport preset with the given name, or nil if not found.
func GetViewportByName(name string) *ViewportPreset {
	for i := range viewportPresets {
		if viewportPresets[i].Name == name {
			return &viewportPresets[i]
		}
	}
	return nil
}
