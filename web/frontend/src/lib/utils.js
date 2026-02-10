import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}

/**
 * Truncate a URL for display, showing host + shortened path.
 */
export function truncateUrl(urlStr, maxLen = 50) {
  if (!urlStr || urlStr.length <= maxLen) return urlStr
  try {
    const url = new URL(urlStr)
    const host = url.hostname.replace(/^www\./, '')
    const path = url.pathname
    const shortPath = path.length > 20 ? path.slice(0, 17) + '...' : path
    const result = host + shortPath
    return result.length > maxLen ? result.slice(0, maxLen - 3) + '...' : result
  } catch {
    return urlStr.slice(0, maxLen - 3) + '...'
  }
}

/**
 * Validate whether a string is a valid http/https URL.
 */
export function isValidUrl(str) {
  try {
    const url = new URL(str)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

/**
 * Trigger a file download from in-memory content.
 */
/**
 * Map finding severity to a tier: 'positive', 'suggestion', or 'bug'.
 */
export function findingTier(severity) {
  if (severity === 'positive') return 'positive'
  if (severity === 'suggestion' || severity === 'minor') return 'suggestion'
  return 'bug' // critical, major, or unknown
}

/**
 * Map finding severity to Badge variant.
 * Note: FindingsTab uses tier-based styling directly; this is used elsewhere.
 */
export function severityVariant(severity) {
  switch (severity) {
    case 'critical': return 'destructive'
    case 'major': return 'default'
    case 'minor':
    case 'suggestion': return 'secondary'
    case 'positive': return 'outline'
    default: return 'secondary'
  }
}

/**
 * Trigger a file download from in-memory content.
 */
export function downloadBlob(content, filename, mimeType = 'text/plain') {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
