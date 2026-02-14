import { ref, onUnmounted } from 'vue'

/**
 * Shared timer composable for tracking elapsed seconds.
 * @returns {{ elapsed: import('vue').Ref<number>, start: (from?: Date) => void, stop: () => void, reset: () => void }}
 */
export function useTimer() {
  const elapsed = ref(0)
  let startedAt = null
  let interval = null

  function start(from) {
    stop()
    startedAt = from || new Date()
    elapsed.value = Math.floor((Date.now() - startedAt.getTime()) / 1000)
    interval = setInterval(() => {
      elapsed.value = Math.floor((Date.now() - startedAt.getTime()) / 1000)
    }, 1000)
  }

  function stop() {
    if (interval) {
      clearInterval(interval)
      interval = null
    }
  }

  function reset() {
    stop()
    elapsed.value = 0
    startedAt = null
  }

  onUnmounted(stop)

  return { elapsed, start, stop, reset }
}
