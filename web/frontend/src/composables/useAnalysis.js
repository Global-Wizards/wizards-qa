import { ref, onUnmounted } from 'vue'
import { analyzeApi, analysesApi, authUrl } from '@/lib/api'
import { getWebSocket } from '@/lib/websocket'
import { MAX_LOGS } from '@/lib/constants'

const MAX_PERSISTED_STEPS_LOAD = 200
const MAX_LIVE_STEPS = 50
const LS_KEY = 'wizards-qa-running-analysis'

// Map granular step names to coarse status for backward compat
const STEP_TO_STATUS = {
  queued: 'queued',
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
  flows_prompt: 'generating',
  flows_calling: 'generating',
  flows_parsing: 'generating',
  flows_validating: 'generating',
  flows_done: 'generating',
  saving: 'generating',
  // Test plan creation steps
  test_plan: 'creating_test_plan',
  test_plan_checking: 'creating_test_plan',
  test_plan_flows: 'creating_test_plan',
  test_plan_saving: 'creating_test_plan',
  test_plan_done: 'creating_test_plan',
  // Browser test execution steps
  testing: 'testing',
  testing_started: 'testing',
  testing_done: 'testing',
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
  agent_adaptive: 'analyzing',
  agent_timeout_extend: 'analyzing',
  user_hint: 'analyzing',
  flows_retry: 'generating',
  resuming: 'scouting',
  // Multi-device batch step
  device_transition: 'scouting',
}

export function useAnalysis() {
  const status = ref('idle') // idle, queued, scouting, analyzing, generating, creating_test_plan, complete, error
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

  // Latest progress step message (for showing live detail during flow generation, etc.)
  const latestStepMessage = ref('')

  // Auto-created test plan ID (set on analysis completion)
  const autoTestPlanId = ref(null)

  // Multi-device batch results
  const devices = ref([])

  // Multi-device progress tracking
  const currentDeviceIndex = ref(0)
  const currentDeviceTotal = ref(0)
  const currentDeviceCategory = ref('')

  // Browser test execution state (inline during analysis)
  const testRunId = ref(null)
  const testStepScreenshots = ref([])
  const testFlowProgress = ref([])

  let hintCooldownTimer = null
  let statusPollInterval = null

  let cleanups = []
  let elapsedTimer = null

  // Reset all reactive state to initial values
  function resetState() {
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
    latestStepMessage.value = ''
    persistedAgentSteps.value = []
    autoTestPlanId.value = null
    devices.value = []
    testRunId.value = null
    testStepScreenshots.value = []
    testFlowProgress.value = []
    currentDeviceIndex.value = 0
    currentDeviceTotal.value = 0
    currentDeviceCategory.value = ''
  }

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

  function startStatusPolling() {
    stopStatusPolling()
    statusPollInterval = setInterval(async () => {
      if (!analysisId.value) return
      // Only poll when in an active (non-terminal) state
      const activeStates = ['queued', 'scouting', 'analyzing', 'generating', 'creating_test_plan', 'testing']
      if (!activeStates.includes(status.value)) return
      try {
        const statusData = await analysesApi.status(analysisId.value)
        if (statusData.status === 'completed') {
          stopStatusPolling()
          const fullData = await analysesApi.get(analysisId.value)
          if (fullData.result) {
            pageMeta.value = fullData.result.pageMeta || null
            analysis.value = fullData.result.analysis || null
            flows.value = fullData.result.flows || []
            agentSteps.value = fullData.result.agentSteps || []
            agentMode.value = fullData.result.mode === 'agent'
            devices.value = fullData.result.devices || []
          }
          autoTestPlanId.value = fullData.testPlanId || null
          status.value = 'complete'
          currentStep.value = 'complete'
          latestScreenshot.value = null
          agentReasoning.value = ''
          stopElapsedTimer()
          clearLocalStorage()
          loadPersistedSteps(analysisId.value)
        } else if (statusData.status === 'failed') {
          stopStatusPolling()
          error.value = statusData.error || 'Analysis failed'
          failedStep.value = currentStep.value || null
          status.value = 'error'
          stopElapsedTimer()
          clearLocalStorage()
          loadPersistedSteps(analysisId.value)
        }
      } catch (err) {
        // 404 means analysis no longer exists (e.g., after server restart) — stop polling
        if (err?.response?.status === 404) {
          stopStatusPolling()
          error.value = 'Analysis no longer available (server may have restarted)'
          failedStep.value = currentStep.value || null
          status.value = 'error'
          stopElapsedTimer()
          clearLocalStorage()
        }
        // Other errors (network timeout, 5xx) — ignore and retry next interval
      }
    }, 15000)
  }

  function stopStatusPolling() {
    if (statusPollInterval) {
      clearInterval(statusPollInterval)
      statusPollInterval = null
    }
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
        latestStepMessage.value = progress.message
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

      // Extract testRunId from testing_started progress event
      if (progress.step === 'testing_started' && progressData.testId) {
        testRunId.value = progressData.testId
      }

      // Track device context from batch progress events
      if (progressData.deviceIndex != null) {
        currentDeviceIndex.value = progressData.deviceIndex
      }
      if (progressData.deviceTotal != null) {
        currentDeviceTotal.value = progressData.deviceTotal
      }
      if (progressData.device) {
        currentDeviceCategory.value = progressData.device
      }

      // Track granular step and timings
      if (progress.step) {
        // Skip device_transition from updating currentStep to prevent
        // the progress phase timeline from visually regressing when
        // the next device starts back at 'scouting'
        if (progress.step === 'device_transition') {
          // Still log the message but don't update step/status
        } else {
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
      }
    })

    // Live agent event listeners
    const offStepDetail = ws.on('agent_step_detail', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      // Deduplicate: skip steps already loaded from DB on reconnect
      if (liveAgentSteps.value.some(s => s.type === 'tool' && s.stepNumber === data.stepNumber)) return
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

      // Debug log entry for agent step
      const durLabel = data.durationMs != null ? ` (${data.durationMs}ms)` : ''
      const errLabel = data.error ? ` ERROR: ${data.error.slice(0, 120)}` : ''
      logs.value = [...logs.value.slice(-(MAX_LOGS - 1)),
        `[Agent] Step ${data.stepNumber}: ${data.toolName || 'unknown'}${durLabel}${errLabel}`]
    })

    const offAgentReasoning = ws.on('agent_reasoning', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      agentReasoning.value = data.text
      // Attach reasoning to latest step
      if (liveAgentSteps.value.length > 0) {
        const updated = [...liveAgentSteps.value]
        const last = updated[updated.length - 1]
        updated[updated.length - 1] = { ...last, reasoning: data.text }
        liveAgentSteps.value = updated
      }
    })

    const offAgentScreenshot = ws.on('agent_screenshot', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      // Use URL-based screenshots instead of base64 to save memory
      const url = authUrl(data.screenshotUrl || '')
      latestScreenshot.value = url
      // Store screenshot URL on the step for inline thumbnails
      if (liveAgentSteps.value.length > 0) {
        const last = liveAgentSteps.value[liveAgentSteps.value.length - 1]
        if (!last.screenshotUrl) {
          const updated = [...liveAgentSteps.value]
          updated[updated.length - 1] = { ...last, hasScreenshot: true, screenshotUrl: url }
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

    // Test execution WS listeners (inline browser tests during analysis)
    const offTestProgress = ws.on('test_progress', (data) => {
      if (!testRunId.value || data.testId !== testRunId.value) return
      if (data.flowName) {
        testFlowProgress.value = [...testFlowProgress.value,
          { flowName: data.flowName, status: data.status, duration: data.duration || '' }]

        // Debug log entry for test flow result
        const durLabel = data.duration ? ` (${data.duration})` : ''
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)),
          `[Test] Flow "${data.flowName}" ${data.status}${durLabel}`]
      }
    })

    const offTestStepScreenshot = ws.on('test_step_screenshot', (data) => {
      if (!testRunId.value || data.testId !== testRunId.value) return
      testStepScreenshots.value = [...testStepScreenshots.value.slice(-(MAX_LIVE_STEPS - 1)), {
        flowName: data.flowName, stepIndex: data.stepIndex, command: data.command,
        screenshotUrl: authUrl(data.screenshotUrl || ''), result: data.result, status: data.status,
        reasoning: data.reasoning || '',
      }]

      // Debug log entry for test step
      logs.value = [...logs.value.slice(-(MAX_LOGS - 1)),
        `[Test] ${data.flowName} step ${data.stepIndex}: ${data.command} → ${data.status}`]
    })

    const offCompleted = ws.on('analysis_completed', (data) => {
      if (analysisId.value && data.analysisId !== analysisId.value) return
      stopStatusPolling()

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
      devices.value = result.devices || []
      autoTestPlanId.value = data.testPlanId || null
      testRunId.value = data.testRunId || null

      // Debug log entry for completion
      logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Complete] Analysis finished successfully`]

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
      stopStatusPolling()

      // End the last step's timing
      const now = Date.now()
      if (currentStep.value && stepTimings.value[currentStep.value]) {
        stepTimings.value = { ...stepTimings.value, [currentStep.value]: { ...stepTimings.value[currentStep.value], end: now } }
      }

      error.value = data.error || 'Analysis failed'
      failedStep.value = currentStep.value || null
      status.value = 'error'

      // Debug log entries for error context
      logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Error] ${data.error || 'Analysis failed'}`]
      if (data.lastStep) {
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Error] Last step: ${data.lastStep}`]
      }
      if (data.exitCode != null && data.exitCode !== -1) {
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Error] Exit code: ${data.exitCode}`]
      }
      if (data.stderrLineCount) {
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Error] Stderr lines: ${data.stderrLineCount}`]
      }
      if (data.hasCheckpoint != null) {
        logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Error] Checkpoint available: ${data.hasCheckpoint}`]
      }
      if (data.stderrTail) {
        data.stderrTail.split('\n').forEach(line => {
          if (line.trim()) {
            logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), `[Stderr] ${line}`]
          }
        })
      }

      stopElapsedTimer()
      clearLocalStorage()

      // Keep liveAgentSteps for debug — do NOT clear them
      // Load persisted steps from API
      loadPersistedSteps(data.analysisId)
    })

    cleanups = [offProgress, offStepDetail, offAgentReasoning, offAgentScreenshot, offUserHint, offTestProgress, offTestStepScreenshot, offCompleted, offFailed]

    startStatusPolling()
  }

  async function start(gameUrl, projectId, useAgentMode = false, profileParams = {}, modules = {}) {
    resetState()
    status.value = 'scouting'
    currentStep.value = 'scouting'
    agentMode.value = useAgentMode

    startElapsedTimer()
    setupListeners()

    try {
      const response = await analyzeApi.start(gameUrl, projectId, useAgentMode, profileParams, modules)
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

  async function continueAnalysis(failedAnalysisId) {
    // Partially reset state — keep analysisId, clear error/logs
    error.value = null
    failedStep.value = null
    logs.value = ['Resuming from checkpoint...']
    status.value = 'scouting'
    currentStep.value = 'resuming'
    analysisId.value = failedAnalysisId
    stepTimings.value = {}
    liveAgentSteps.value = []
    latestScreenshot.value = null
    agentReasoning.value = ''
    testRunId.value = null
    testStepScreenshots.value = []
    testFlowProgress.value = []

    startElapsedTimer()
    setupListeners()

    try {
      const response = await analyzeApi.continue(failedAnalysisId)

      // Persist to localStorage for reconnect
      localStorage.setItem(LS_KEY, JSON.stringify({
        analysisId: failedAnalysisId,
        gameUrl: '',
        startedAt: startTime.value || Date.now(),
        agentMode: agentMode.value,
      }))

      return response
    } catch (err) {
      error.value = err.message || 'Failed to continue analysis'
      status.value = 'error'
      stopElapsedTimer()
      clearLocalStorage()
    }
  }

  /**
   * Start tracking a pre-existing batch analysis (created by the batch endpoint).
   * Unlike start(), this does NOT make an API POST — the batch endpoint already created
   * the analysis. We just set up WS listeners and status polling.
   */
  function startBatch(batchAnalysisId, gameUrl, useAgentMode = false) {
    resetState()
    status.value = 'scouting'
    currentStep.value = 'scouting'
    analysisId.value = batchAnalysisId
    agentMode.value = useAgentMode

    startElapsedTimer()
    setupListeners()

    // Persist to localStorage so we can recover on refresh
    localStorage.setItem(LS_KEY, JSON.stringify({
      analysisId: batchAnalysisId,
      gameUrl,
      startedAt: startTime.value || Date.now(),
      agentMode: useAgentMode,
    }))
  }

  /**
   * Try to recover a running or completed analysis.
   * If explicitAnalysisId is provided, fetches the record from the API
   * (used when navigating from the analyses list). Otherwise uses localStorage.
   * Returns { recovered: true, status: 'running'|'completed' } or false.
   */
  async function tryRecover(explicitAnalysisId = null) {
    let parsed

    if (explicitAnalysisId) {
      // Fetch analysis record to build recovery state
      try {
        const fullData = await analysesApi.get(explicitAnalysisId)
        parsed = {
          analysisId: explicitAnalysisId,
          gameUrl: fullData.gameUrl || '',
          startedAt: new Date(fullData.createdAt).getTime(),
          agentMode: fullData.agentMode || false,
        }
      } catch {
        return false
      }
    } else {
      const saved = localStorage.getItem(LS_KEY)
      if (!saved) return false

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
    }

    try {
      const statusData = await analysesApi.status(parsed.analysisId)

      if (statusData.status === 'running') {
        // Persist to localStorage so page refresh continues to recover
        localStorage.setItem(LS_KEY, JSON.stringify(parsed))
        // Reconnect to running analysis
        analysisId.value = parsed.analysisId
        status.value = STEP_TO_STATUS[statusData.step] || 'analyzing'
        currentStep.value = statusData.step || ''

        // Restore agent mode from persisted state or infer from step name
        const agentStepNames = ['agent_start', 'agent_step', 'agent_action', 'agent_adaptive', 'agent_timeout_extend', 'agent_done', 'agent_synthesize', 'synthesis_retry', 'agent_reasoning', 'agent_step_detail', 'agent_screenshot']
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

        // Load persisted agent steps that were saved during execution
        try {
          const stepsData = await analysesApi.steps(parsed.analysisId)
          const steps = stepsData.steps || []
          if (steps.length > 0) {
            liveAgentSteps.value = steps.map(s => ({
              stepNumber: s.stepNumber,
              toolName: s.toolName,
              input: s.input,
              result: s.result,
              error: s.error || '',
              durationMs: s.durationMs,
              type: 'tool',
              timestamp: new Date(s.createdAt).getTime() || Date.now(),
              reasoning: s.reasoning || '',
              hasScreenshot: !!s.screenshotPath,
              screenshotUrl: s.screenshotPath
                ? analysesApi.stepScreenshotUrl(parsed.analysisId, s.stepNumber)
                : '',
            }))
            agentStepCurrent.value = steps.length
            persistedAgentSteps.value = steps
            logs.value = [
              'Reconnected to running analysis...',
              `Restored ${steps.length} agent step(s) from server.`,
              ...steps.map(s => `Step ${s.stepNumber}: ${s.toolName} (${s.durationMs}ms)`)
            ]
          }
        } catch {
          // Steps may not be available yet — continue with empty state
        }

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
          devices.value = fullData.result.devices || []
        }
        autoTestPlanId.value = fullData.testPlanId || null
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
    stopStatusPolling()
    stopElapsedTimer()
    clearLocalStorage()
    resetState()
    if (hintCooldownTimer) {
      clearTimeout(hintCooldownTimer)
      hintCooldownTimer = null
    }
  }

  function stopListening() {
    stopStatusPolling()
    cleanups.forEach((fn) => fn())
    cleanups = []
  }

  onUnmounted(() => {
    stopListening()
    stopStatusPolling()
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
    startBatch,
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
    // Continue from checkpoint
    continueAnalysis,
    // Failed step tracking
    failedStep,
    // Persisted agent steps
    persistedAgentSteps,
    loadPersistedSteps,
    // Live step message
    latestStepMessage,
    // Auto-created test plan
    autoTestPlanId,
    // Inline browser test execution
    testRunId,
    testStepScreenshots,
    testFlowProgress,
    // Multi-device batch
    devices,
    // Multi-device progress tracking
    currentDeviceIndex,
    currentDeviceTotal,
    currentDeviceCategory,
  }
}
