/**
 * Format a date string for display (e.g., "Jan 5, 10:30 AM").
 */
export function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateStr
  }
}

/**
 * Return a human-readable relative time string (e.g., "2h ago").
 */
export function timeAgo(timestamp) {
  if (!timestamp) return '-'
  const now = Date.now()
  const diff = now - new Date(timestamp).getTime()
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return new Date(timestamp).toLocaleDateString()
}

/**
 * Return a full locale-formatted timestamp string, suitable for tooltips.
 */
export function fullTimestamp(timestamp) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleString()
}
