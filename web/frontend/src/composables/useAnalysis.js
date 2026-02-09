import { ref, onUnmounted } from 'vue'
import { analyzeApi, analysesApi } from '@/lib/api'
import { getWebSocket } from '@/lib/websocket'

const MAX_PERSISTED_STEPS_LOAD = 200

const MAX_LOGS = 500
const MAX_LIVE_STEPS = 50
const LS_KEY = 'wizards-qa-running-analysis'

// Map granular step names to coarse status for backward compat
const STEP_TO_STATUS = {
  scouting: 'scouting',
  scouted: 'scouting',
  scouted_detail: 'scouting',
  fallback: 'scouting',
  analyzing: 'analyzing',
  analyzed: 'analyzing',
  analyzed_detail: 'analyzing',
  scenarios: 'analyzing',
  scenarios_done: 'analyzing',
  flows: 'generating',
  flows_done: 'generating',
  saving: 'generating',
  // Agent mode steps
  agent_start: 'analyzing',
  agent_step: 'analyzing',
  agent_action: 'analyzing',
  agent_done: 'analyzing',
  agent_synthesize: 'analyzing',
  synthesis_retry: 'analyzing',
  agent_reasoning: 'analyzing',
  agent_step_detail: 'analyzing',
  agent_screenshot: 'analyzing',
  user_hint: 'analyzing',
  flows_retry: 'generating',
}

export function useAnalysis() {
  const status = ref('idle') // idle, scouting, analyzing, generating, complete, error
  const currentStep = ref('') // granular step name
  const analysisId = ref(null)
  const pageMeta = ref(null)
  const analysis = ref(null)
  const flows = ref([])
  const agentSteps = ref([])
  const agentMode = ref(false)
  const error = ref(null)
  const logs = ref([])
  const startTime = ref(null)
  const elapsedSeconds = ref(0)
  const stepTimings = ref({}) // { scouting: {start, end}, analyzing: {start, end}, ... }

  const failedStep = ref(null) // last step name when analysis failed

  // Live agent exploration state
  const liveAgentSteps = ref([])
  const latestScreenshot = ref(null)
  const agentReasoning = ref('')
  const userHints = ref([])
  const hintCooldown = ref(false)
  const agentStepCurrent = ref(0)
  const agentStepTotal = ref(0)

  // Persisted agent steps (loaded from API after completion/failure)
  const persistedAgentSteps = ref([])

  let hintCooldownTimer = null

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

  async function sendHint(message) {
    if (!analysisId.value || hintCooldown.value || !message?.trim()) return
    try {
      await analyzeApi.sendHint(analysisId.value, message.trim())
      userHints.value = [...userHints.value, { message: message.trim(), sentAt: Date.now() }]
      hintCooldown.value = true
      if (hintCooldownTimer) clearTimeout(hintCooldownTimer)
      hintCooldownTimer = setTimeout(() => { hintCooldown.value = false }, 5000)
    } catch {
      // 410/404 = analysis ended, just disable
      hintCooldown.value = false
    }
  }

  async function loadPersistedSteps(id) {
    if (!id) return
    try {
      const data = await analysesApi.steps(id)
      persistedAgentSteps.value = (data.steps || []).slice(0, MAX_PERSISTED_STEPS_LOAD)
    } catch {
      // API may not be available yet — ignore
    }
  }

  function clearLocalStorage() {
    localStorage.removeItem(LS_KEY)
  }

  function setupListeners() {
    // Clean up any previous listeners to prevent accumulation
    stopListening()
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

      // Parse agent step counter (format: "Step X/Y: ...")
      if (progress.step === 'agent_step' && progress.message) {
        const match = progress.message.match(/^Step (\d+)\/(\d+)/)
        if (match) {
          agentStepCurrent.value = parseInt(match[1], 10)
          agentStepTotal.value = parseInt(match[2], 10)
        }
      }

      // Track granular step and timings
      if (progress.step) {
        const now = Date.now()
        // End previous step
        if (currentStep.value && stepTimings.value[currentStep.value]) {
          stepTimings.value[currentStep.value].end = now
        }
        // Start new step
        stepTimings.value = { ...stepTimings.value, [progress.step]: { start: now, end: null } }

        currentStep.value = progress.step
        const coarseStatus = STEP_TO_STATUS[progress.step]
        if (coarseStatus) {
          status.value = coarseStatus
        }
      }
    })

    // Live agent event listeners
    const offStepDetail = ws.on('agent_step_detail', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      const newStep = {
        stepNumber: data.stepNumber,
        toolName: data.toolName,
        input: data.input,
        result: data.result,
        error: data.error,
        durationMs: data.durationMs,
        type: 'tool',
        timestamp: Date.now(),
      }
      liveAgentSteps.value = [...liveAgentSteps.value.slice(-(MAX_LIVE_STEPS - 1)), newStep]
    })

    const offAgentReasoning = ws.on('agent_reasoning', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      agentReasoning.value = data.text
    })

    const offAgentScreenshot = ws.on('agent_screenshot', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      latestScreenshot.value = data.imageData
      // Mark the most recent live step as having a screenshot (don't store full base64)
      if (liveAgentSteps.value.length > 0) {
        const last = liveAgentSteps.value[liveAgentSteps.value.length - 1]
        if (!last.hasScreenshot) {
          const updated = [...liveAgentSteps.value]
          updated[updated.length - 1] = { ...last, hasScreenshot: true }
          liveAgentSteps.value = updated
        }
      }
    })

    const offUserHint = ws.on('agent_user_hint', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      // Add hint to live timeline
      const hintEntry = {
        type: 'hint',
        message: data.message,
        timestamp: Date.now(),
      }
      liveAgentSteps.value = [...liveAgentSteps.value.slice(-(MAX_LIVE_STEPS - 1)), hintEntry]
    })

    const offCompleted = ws.on('analysis_completed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      // End the last step's timing
      const now = Date.now()
      if (currentStep.value && stepTimings.value[currentStep.value]) {
        stepTimings.value = { ...stepTimings.value, [currentStep.value]: { ...stepTimings.value[currentStep.value], end: now } }
      }

      const result = data.result || {}
      pageMeta.value = result.pageMeta || null
      analysis.value = result.analysis || null
      flows.value = result.flows || []
      agentSteps.value = result.agentSteps || []
      agentMode.value = result.mode === 'agent'
      status.value = 'complete'
      currentStep.value = 'complete'
      latestScreenshot.value = null
      agentReasoning.value = ''
      stopElapsedTimer()
      clearLocalStorage()

      // Load persisted steps from API (has screenshots, reasoning, etc.)
      loadPersistedSteps(data.analysisId)
    })

    const offFailed = ws.on('analysis_failed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return

      // End the last step's timing
      const now = Date.now()
      if (currentStep.value && stepTimings.value[currentStep.value]) {
        stepTimings.value = { ...stepTimings.value, [currentStep.value]: { ...stepTimings.value[currentStep.value], end: now } }
      }

      error.value = data.error || 'Analysis failed'
      failedStep.value = currentStep.value || null
      status.value = 'error'
      stopElapsedTimer()
      clearLocalStorage()

      // Keep liveAgentSteps for debug — do NOT clear them
      // Load persisted steps from API
      loadPersistedSteps(data.analysisId)
    })

    cleanups = [offProgress, offStepDetail, offAgentReasoning, offAgentScreenshot, offUserHint, offCompleted, offFailed]
  }

  async function start(gameUrl, projectId, useAgentMode = false, profileParams = {}) {
    status.value = 'scouting'
    currentStep.value = 'scouting'
    analysisId.value = null
    pageMeta.value = null
    analysis.value = null
    flows.value = []
    agentSteps.value = []
    agentMode.value = useAgentMode
    error.value = null
    failedStep.value = null
    logs.value = []
    stepTimings.value = {}
    liveAgentSteps.value = []
    latestScreenshot.value = null
    agentReasoning.value = ''
    userHints.value = []
    hintCooldown.value = false
    agentStepCurrent.value = 0
    agentStepTotal.value = 0

    startElapsedTimer()
    setupListeners()

    try {
      const response = await analyzeApi.start(gameUrl, projectId, useAgentMode, profileParams)
      analysisId.value = response.analysisId

      // Persist to localStorage so we can recover
      localStorage.setItem(LS_KEY, JSON.stringify({
        analysisId: response.analysisId,
        gameUrl,
        startedAt: startTime.value || Date.now(),
        agentMode: useAgentMode,
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

        // Restore agent mode from persisted state or infer from step name
        const agentStepNames = ['agent_start', 'agent_step', 'agent_action', 'agent_done', 'agent_synthesize', 'synthesis_retry', 'agent_reasoning', 'agent_step_detail', 'agent_screenshot']
        if (parsed.agentMode || agentStepNames.includes(statusData.step)) {
          agentMode.value = true
        }

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
          agentSteps.value = fullData.result.agentSteps || []
          agentMode.value = fullData.result.mode === 'agent'
        }
        status.value = 'complete'
        currentStep.value = 'complete'
        clearLocalStorage()

        // Load persisted agent steps
        loadPersistedSteps(parsed.analysisId)

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
    agentSteps.value = []
    agentMode.value = false
    error.value = null
    failedStep.value = null
    logs.value = []
    elapsedSeconds.value = 0
    startTime.value = null
    stepTimings.value = {}
    liveAgentSteps.value = []
    latestScreenshot.value = null
    agentReasoning.value = ''
    userHints.value = []
    hintCooldown.value = false
    agentStepCurrent.value = 0
    agentStepTotal.value = 0
    persistedAgentSteps.value = []
    if (hintCooldownTimer) {
      clearTimeout(hintCooldownTimer)
      hintCooldownTimer = null
    }
  }

  function stopListening() {
    cleanups.forEach((fn) => fn())
    cleanups = []
  }

  onUnmounted(() => {
    stopListening()
    stopElapsedTimer()
    if (hintCooldownTimer) clearTimeout(hintCooldownTimer)
  })

  return {
    status,
    currentStep,
    analysisId,
    pageMeta,
    analysis,
    flows,
    agentSteps,
    agentMode,
    error,
    logs,
    elapsedSeconds,
    stepTimings,
    formatElapsed,
    start,
    reset,
    tryRecover,
    stopListening,
    // Live agent exploration
    liveAgentSteps,
    latestScreenshot,
    agentReasoning,
    userHints,
    hintCooldown,
    agentStepCurrent,
    agentStepTotal,
    sendHint,
    // Failed step tracking
    failedStep,
    // Persisted agent steps
    persistedAgentSteps,
    loadPersistedSteps,
  }
}
