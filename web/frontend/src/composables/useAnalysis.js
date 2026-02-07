import { ref, onUnmounted } from 'vue'
import { analyzeApi } from '@/lib/api'
import { getWebSocket } from '@/lib/websocket'

const MAX_LOGS = 500

export function useAnalysis() {
  const status = ref('idle') // idle, scouting, analyzing, generating, complete, error
  const analysisId = ref(null)
  const pageMeta = ref(null)
  const analysis = ref(null)
  const flows = ref([])
  const error = ref(null)
  const logs = ref([])

  let cleanups = []

  async function start(gameUrl) {
    status.value = 'scouting'
    analysisId.value = null
    pageMeta.value = null
    analysis.value = null
    flows.value = []
    error.value = null
    logs.value = []

    const ws = getWebSocket()
    ws.connect()

    const offProgress = ws.on('analysis_progress', (data) => {
      const progress = data
      const progressData = progress.data || {}

      if (analysisId.value && progressData.analysisId && progressData.analysisId !== analysisId.value) {
        return
      }

      if (progress.message) {
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), progress.message]
      }

      switch (progress.step) {
        case 'scouting':
          status.value = 'scouting'
          break
        case 'analyzing':
          status.value = 'analyzing'
          break
        case 'generating':
          status.value = 'generating'
          break
      }
    })

    const offCompleted = ws.on('analysis_completed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      const result = data.result || {}
      pageMeta.value = result.pageMeta || null
      analysis.value = result.analysis || null
      flows.value = result.flows || []
      status.value = 'complete'
    })

    const offFailed = ws.on('analysis_failed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      error.value = data.error || 'Analysis failed'
      status.value = 'error'
    })

    cleanups = [offProgress, offCompleted, offFailed]

    try {
      const response = await analyzeApi.start(gameUrl)
      analysisId.value = response.analysisId
    } catch (err) {
      error.value = err.message || 'Failed to start analysis'
      status.value = 'error'
    }
  }

  function reset() {
    stopListening()
    status.value = 'idle'
    analysisId.value = null
    pageMeta.value = null
    analysis.value = null
    flows.value = []
    error.value = null
    logs.value = []
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
    analysisId,
    pageMeta,
    analysis,
    flows,
    error,
    logs,
    start,
    reset,
    stopListening,
  }
}
