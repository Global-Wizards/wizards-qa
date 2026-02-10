export const VIEWPORT_PRESETS = [
  // Desktop
  { name: 'desktop-hd', label: 'Desktop HD', category: 'Desktop', width: 1920, height: 1080, devicePixelRatio: 1 },
  { name: 'desktop-std', label: 'Desktop Standard', category: 'Desktop', width: 1280, height: 720, devicePixelRatio: 1 },
  { name: 'desktop-lg', label: 'Desktop Large', category: 'Desktop', width: 2560, height: 1440, devicePixelRatio: 1 },
  { name: 'desktop-4k', label: 'Desktop 4K', category: 'Desktop', width: 3840, height: 2160, devicePixelRatio: 1 },
  { name: 'laptop-13', label: 'Laptop 13"', category: 'Desktop', width: 1440, height: 900, devicePixelRatio: 2 },
  { name: 'laptop-15', label: 'Laptop 15"', category: 'Desktop', width: 1536, height: 864, devicePixelRatio: 2 },
  { name: 'macbook-air', label: 'MacBook Air', category: 'Desktop', width: 1440, height: 900, devicePixelRatio: 2 },
  { name: 'macbook-pro-16', label: 'MacBook Pro 16"', category: 'Desktop', width: 1728, height: 1117, devicePixelRatio: 2 },

  // iPhone
  { name: 'iphone-16-pro-max', label: 'iPhone 16 Pro Max', category: 'iPhone', width: 440, height: 956, devicePixelRatio: 3 },
  { name: 'iphone-16-pro', label: 'iPhone 16 Pro', category: 'iPhone', width: 402, height: 874, devicePixelRatio: 3 },
  { name: 'iphone-16', label: 'iPhone 16', category: 'iPhone', width: 393, height: 852, devicePixelRatio: 3 },
  { name: 'iphone-15', label: 'iPhone 15', category: 'iPhone', width: 393, height: 852, devicePixelRatio: 3 },
  { name: 'iphone-se', label: 'iPhone SE', category: 'iPhone', width: 375, height: 667, devicePixelRatio: 2 },
  { name: 'iphone-14-plus', label: 'iPhone 14 Plus', category: 'iPhone', width: 428, height: 926, devicePixelRatio: 3 },
  { name: 'iphone-13-mini', label: 'iPhone 13 Mini', category: 'iPhone', width: 375, height: 812, devicePixelRatio: 3 },

  // iPad
  { name: 'ipad-pro-12', label: 'iPad Pro 12.9"', category: 'iPad', width: 1024, height: 1366, devicePixelRatio: 2 },
  { name: 'ipad-pro-11', label: 'iPad Pro 11"', category: 'iPad', width: 834, height: 1194, devicePixelRatio: 2 },
  { name: 'ipad-air', label: 'iPad Air', category: 'iPad', width: 820, height: 1180, devicePixelRatio: 2 },
  { name: 'ipad-mini', label: 'iPad Mini', category: 'iPad', width: 744, height: 1133, devicePixelRatio: 2 },
  { name: 'ipad-10th', label: 'iPad 10th Gen', category: 'iPad', width: 810, height: 1080, devicePixelRatio: 2 },

  // Android Phones
  { name: 'pixel-9-pro', label: 'Pixel 9 Pro', category: 'Android', width: 412, height: 892, devicePixelRatio: 2.625 },
  { name: 'pixel-9', label: 'Pixel 9', category: 'Android', width: 412, height: 892, devicePixelRatio: 2.625 },
  { name: 'samsung-s24-ultra', label: 'Samsung S24 Ultra', category: 'Android', width: 412, height: 915, devicePixelRatio: 3.5 },
  { name: 'samsung-s24', label: 'Samsung S24', category: 'Android', width: 360, height: 780, devicePixelRatio: 3 },
  { name: 'samsung-a54', label: 'Samsung A54', category: 'Android', width: 412, height: 915, devicePixelRatio: 2.625 },
  { name: 'oneplus-12', label: 'OnePlus 12', category: 'Android', width: 412, height: 915, devicePixelRatio: 3.5 },

  // Android Tablets
  { name: 'samsung-tab-s9', label: 'Samsung Tab S9', category: 'Android Tablet', width: 800, height: 1280, devicePixelRatio: 2 },
  { name: 'pixel-tablet', label: 'Pixel Tablet', category: 'Android Tablet', width: 800, height: 1280, devicePixelRatio: 2 },
]

// Subset shown in "Recommended" group at the top of the selector
export const RECOMMENDED_VIEWPORTS = [
  'desktop-std',
  'desktop-hd',
  'iphone-16-pro',
  'ipad-pro-11',
  'samsung-s24-ultra',
]

export const DEFAULT_VIEWPORT = 'desktop-std' // 1280x720

export function getViewportByName(name) {
  return VIEWPORT_PRESETS.find(p => p.name === name) || null
}

export function getViewportCategories() {
  const cats = [...new Set(VIEWPORT_PRESETS.map(p => p.category))]
  return cats.map(cat => ({ name: cat, presets: VIEWPORT_PRESETS.filter(p => p.category === cat) }))
}

export function getRecommendedViewports() {
  return RECOMMENDED_VIEWPORTS.map(name => getViewportByName(name)).filter(Boolean)
}
