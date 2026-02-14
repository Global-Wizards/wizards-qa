/**
 * Format elapsed seconds as a human-readable string.
 * @param {number} seconds
 * @returns {string} e.g. "45s", "2m 05s"
 */
export function formatElapsed(seconds) {
  if (seconds < 60) return `${seconds}s`
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}m ${s.toString().padStart(2, '0')}s`
}
