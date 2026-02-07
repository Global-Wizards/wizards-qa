import { ref, onUnmounted } from 'vue'
import { getWebSocket } from '@/lib/websocket'

export function useTestExecution() {
  const status = ref('idle') // idle, running, completed, failed
  const logs = ref([])
  const progress = ref([])
  const result = ref(null)

  let cleanups = []

  function startExecution(testId) {
    status.value = 'running'
    logs.value = []
    progress.value = []
    result.value = null

    const ws = getWebSocket()
    ws.connect()

    const offStarted = ws.on('test_started', (data) => {
      if (data.testId === testId) {
        status.value = 'running'
      }
    })

    const offProgress = ws.on('test_progress', (data) => {
      if (data.testId === testId) {
        if (data.line) {
          logs.value = [...logs.value, data.line]
        }
        if (data.flowName) {
          const existing = progress.value.find((p) => p.flowName === data.flowName)
          if (existing) {
            existing.status = data.status
          } else {
            progress.value = [...progress.value, { flowName: data.flowName, status: data.status }]
          }
        }
      }
    })

    const offCompleted = ws.on('test_completed', (data) => {
      if (data.testId === testId) {
        status.value = 'completed'
        result.value = data
      }
    })

    const offFailed = ws.on('test_failed', (data) => {
      if (data.testId === testId) {
        status.value = 'failed'
        result.value = data
      }
    })

    cleanups = [offStarted, offProgress, offCompleted, offFailed]
  }

  function stopListening() {
    cleanups.forEach((fn) => fn())
    cleanups = []
  }

  onUnmounted(() => {
    stopListening()
  })

  return {
    status,
    logs,
    progress,
    result,
    startExecution,
    stopListening,
  }
}
