import { onUnmounted } from 'vue'
import { getWebSocket } from '@/lib/websocket'

/**
 * Register WebSocket event listeners that auto-cleanup on component unmount.
 * @param {Record<string, Function>} handlers - Map of event name to handler function
 * @returns {{ cleanup: Function }} - Manual cleanup function if needed before unmount
 */
export function useWsListeners(handlers) {
  const ws = getWebSocket()
  ws.connect()
  const cleanups = Object.entries(handlers).map(([event, fn]) => ws.on(event, fn))

  function cleanup() {
    cleanups.forEach(off => off())
  }

  onUnmounted(cleanup)

  return { cleanup }
}
