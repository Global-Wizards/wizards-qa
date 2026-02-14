import { ref } from 'vue'
import { getWebSocket } from '@/lib/websocket'

const wsConnected = ref(true)
const wsReconnecting = ref(false)

// Register WS listeners at module level since refs are module-level singletons.
// This prevents the memory leak from registering new listeners every time the
// composable is used in a component, while still updating the shared refs.
const ws = getWebSocket()
ws.on('connected', () => {
  wsConnected.value = true
  wsReconnecting.value = false
})
ws.on('disconnected', () => {
  wsConnected.value = false
})
ws.on('reconnecting', () => {
  wsReconnecting.value = true
})
ws.on('connection_lost', () => {
  wsReconnecting.value = false
  wsConnected.value = false
})

export function useConnectionStatus() {
  return { wsConnected, wsReconnecting }
}
