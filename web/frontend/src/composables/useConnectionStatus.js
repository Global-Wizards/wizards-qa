import { ref, onMounted, onUnmounted } from 'vue'
import { getWebSocket } from '@/lib/websocket'

const wsConnected = ref(true)
const wsReconnecting = ref(false)

let initialized = false

export function useConnectionStatus() {
  onMounted(() => {
    if (initialized) return
    initialized = true

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
  })

  return { wsConnected, wsReconnecting }
}
