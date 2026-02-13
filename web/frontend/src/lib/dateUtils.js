import { format, formatDistanceToNowStrict } from 'date-fns'

/**
 * Format a date string for display (e.g., "Jan 5, 10:30 AM").
 */
export function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    return format(new Date(dateStr), 'MMM d, h:mm a')
  } catch {
    return dateStr
  }
}

/**
 * Return a human-readable relative time string (e.g., "2h ago").
 */
export function timeAgo(timestamp) {
  if (!timestamp) return '-'
  try {
    const date = new Date(timestamp)
    const diff = Date.now() - date.getTime()
    if (diff < 60000) return 'just now'
    const str = formatDistanceToNowStrict(date)
    return str
      .replace(/ seconds?/, 's')
      .replace(/ minutes?/, 'm')
      .replace(/ hours?/, 'h')
      .replace(/ days?/, 'd')
      .replace(/ months?/, 'mo')
      .replace(/ years?/, 'y')
      + ' ago'
  } catch {
    return '-'
  }
}

/**
 * Return a full locale-formatted timestamp string, suitable for tooltips.
 */
export function fullTimestamp(timestamp) {
  if (!timestamp) return ''
  try {
    return format(new Date(timestamp), 'PPpp')
  } catch {
    return ''
  }
}
