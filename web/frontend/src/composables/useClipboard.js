import { ref, onUnmounted } from 'vue'

/**
 * Reusable clipboard copy composable with auto-reset "copied" state.
 * Returns { copied, copy } where copy(text) writes to clipboard and
 * sets copied=true for 2 seconds.
 */
export function useClipboard() {
  const copied = ref(false)
  let timeoutId = null

  async function copy(text) {
    try {
      await navigator.clipboard.writeText(text)
      copied.value = true
      if (timeoutId != null) clearTimeout(timeoutId)
      timeoutId = setTimeout(() => { copied.value = false }, 2000)
    } catch {
      // clipboard API not available
    }
  }

  onUnmounted(() => {
    if (timeoutId != null) clearTimeout(timeoutId)
  })

  return { copied, copy }
}
