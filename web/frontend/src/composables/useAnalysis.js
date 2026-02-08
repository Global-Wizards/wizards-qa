import { ref, onUnmounted } from 'vue'
import { analyzeApi, analysesApi } from '@/lib/api'
import { getWebSocket } from '@/lib/websocket'

const MAX_LOGS = 500
const LS_KEY = 'wizards-qa-running-analysis'

// Map granular step names to coarse status for backward compat
const STEP_TO_STATUS = {
  scouting: 'scouting',
  scouted: 'scouting',
  analyzing: 'analyzing',
  analyzed: 'analyzing',
  scenarios: 'analyzing',
  scenarios_done: 'analyzing',
  flows: 'generating',
  flows_done: 'generating',
  saving: 'generating',
}

export function useAnalysis() {
  const status = ref('idle') // idle, scouting, analyzing, generating, complete, error
  const currentStep = ref('') // granular step name
  const analysisId = ref(null)
  const pageMeta = ref(null)
  const analysis = ref(null)
  const flows = ref([])
  const error = ref(null)
  const logs = ref([])
  const startTime = ref(null)
  const elapsedSeconds = ref(0)

  let cleanups = []
  let elapsedTimer = null

  function startElapsedTimer() {
    stopElapsedTimer()
    startTime.value = Date.now()
    elapsedSeconds.value = 0
    elapsedTimer = setInterval(() => {
      elapsedSeconds.value = Math.floor((Date.now() - startTime.value) / 1000)
    }, 1000)
  }

  function stopElapsedTimer() {
    if (elapsedTimer) {
      clearInterval(elapsedTimer)
      elapsedTimer = null
    }
  }

  function formatElapsed(seconds) {
    if (seconds < 60) return `${seconds}s`
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}m ${s}s`
  }

  function saveToLocalStorage() {
    if (analysisId.value) {
      localStorage.setItem(LS_KEY, JSON.stringify({
        analysisId: analysisId.value,
        gameUrl: '', // will be set by caller
        startedAt: startTime.value || Date.now(),
      }))
    }
  }

  function clearLocalStorage() {
    localStorage.removeItem(LS_KEY)
  }

  function setupListeners() {
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

      // Extract partial data from progress events when available
      if (progressData.pageMeta) {
        pageMeta.value = progressData.pageMeta
      }
      if (progressData.analysis) {
        analysis.value = progressData.analysis
      }

      // Track granular step
      if (progress.step) {
        currentStep.value = progress.step
        const coarseStatus = STEP_TO_STATUS[progress.step]
        if (coarseStatus) {
          status.value = coarseStatus
        }
      }
    })

    const offCompleted = ws.on('analysis_completed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      const result = data.result || {}
      pageMeta.value = result.pageMeta || null
      analysis.value = result.analysis || null
      flows.value = result.flows || []
      status.value = 'complete'
      currentStep.value = 'complete'
      stopElapsedTimer()
      clearLocalStorage()
    })

    const offFailed = ws.on('analysis_failed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      error.value = data.error || 'Analysis failed'
      status.value = 'error'
      currentStep.value = ''
      stopElapsedTimer()
      clearLocalStorage()
    })

    cleanups = [offProgress, offCompleted, offFailed]
  }

  async function start(gameUrl) {
    status.value = 'scouting'
    currentStep.value = 'scouting'
    analysisId.value = null
    pageMeta.value = null
    analysis.value = null
    flows.value = []
    error.value = null
    logs.value = []

    startElapsedTimer()
    setupListeners()

    try {
      const response = await analyzeApi.start(gameUrl)
      analysisId.value = response.analysisId

      // Persist to localStorage so we can recover
      localStorage.setItem(LS_KEY, JSON.stringify({
        analysisId: response.analysisId,
        gameUrl,
        startedAt: startTime.value || Date.now(),
      }))
    } catch (err) {
      error.value = err.message || 'Failed to start analysis'
      status.value = 'error'
      stopElapsedTimer()
      clearLocalStorage()
    }
  }

  /**
   * Try to recover a running or completed analysis from localStorage.
   * Returns { recovered: true, status: 'running'|'completed' } or false.
   */
  async function tryRecover() {
    const saved = localStorage.getItem(LS_KEY)
    if (!saved) return false

    let parsed
    try {
      parsed = JSON.parse(saved)
    } catch {
      clearLocalStorage()
      return false
    }

    if (!parsed.analysisId) {
      clearLocalStorage()
      return false
    }

    try {
      const statusData = await analysesApi.status(parsed.analysisId)

      if (statusData.status === 'running') {
        // Reconnect to running analysis
        analysisId.value = parsed.analysisId
        status.value = STEP_TO_STATUS[statusData.step] || 'analyzing'
        currentStep.value = statusData.step || ''

        // Resume elapsed timer from original start
        startTime.value = parsed.startedAt || Date.now()
        elapsedSeconds.value = Math.floor((Date.now() - startTime.value) / 1000)
        elapsedTimer = setInterval(() => {
          elapsedSeconds.value = Math.floor((Date.now() - startTime.value) / 1000)
        }, 1000)

        logs.value = ['Reconnected to running analysis...']
        setupListeners()

        return { recovered: true, status: 'running', gameUrl: parsed.gameUrl }
      }

      if (statusData.status === 'completed') {
        // Load full result
        const fullData = await analysesApi.get(parsed.analysisId)
        analysisId.value = parsed.analysisId
        if (fullData.result) {
          pageMeta.value = fullData.result.pageMeta || null
          analysis.value = fullData.result.analysis || null
          flows.value = fullData.result.flows || []
        }
        status.value = 'complete'
        currentStep.value = 'complete'
        clearLocalStorage()

        return { recovered: true, status: 'completed', gameUrl: parsed.gameUrl }
      }

      // Failed or unknown — clear
      clearLocalStorage()
      return false
    } catch {
      clearLocalStorage()
      return false
    }
  }

  function reset() {
    stopListening()
    stopElapsedTimer()
    clearLocalStorage()
    status.value = 'idle'
    currentStep.value = ''
    analysisId.value = null
    pageMeta.value = null
    analysis.value = null
    flows.value = []
    error.value = null
    logs.value = []
    elapsedSeconds.value = 0
    startTime.value = null
  }

  function stopListening() {
    cleanups.forEach((fn) => fn())
    cleanups = []
  }

  onUnmounted(() => {
    stopListening()
    stopElapsedTimer()
  })

  return {
    status,
    currentStep,
    analysisId,
    pageMeta,
    analysis,
    flows,
    error,
    logs,
    elapsedSeconds,
    formatElapsed,
    start,
    reset,
    tryRecover,
    stopListening,
  }
}
