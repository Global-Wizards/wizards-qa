import { ref, computed, onUnmounted } from 'vue'
import { getWebSocket } from '@/lib/websocket'
import { testsApi } from '@/lib/api'

const MAX_LOGS = 500

export function useTestExecution() {
  const status = ref('idle') // idle, starting, running, completed, failed
  const phase = ref('starting') // starting, preparing, executing, results, complete
  const logs = ref([])
  const progress = ref([]) // { flowName, status, duration }
  const result = ref(null)
  const totalFlows = ref(0)
  const planName = ref('')
  const planId = ref('')
  const testId = ref('')
  const elapsedSeconds = ref(0)
  const startedAt = ref(null)

  let cleanups = []
  let timerInterval = null

  function formatElapsed(s) {
    const m = Math.floor(s / 60)
    const sec = s % 60
    return `${m}m ${sec.toString().padStart(2, '0')}s`
  }

  function startTimer(from) {
    stopTimer()
    startedAt.value = from || new Date()
    elapsedSeconds.value = Math.floor((Date.now() - startedAt.value.getTime()) / 1000)
    timerInterval = setInterval(() => {
      elapsedSeconds.value = Math.floor((Date.now() - startedAt.value.getTime()) / 1000)
    }, 1000)
  }

  function stopTimer() {
    if (timerInterval) {
      clearInterval(timerInterval)
      timerInterval = null
    }
  }

  // Detect phase from log lines
  function detectPhase(line) {
    if (/executing flows|running flows|starting execution/i.test(line)) {
      phase.value = 'executing'
    } else if (/--- results ---|summary|test results/i.test(line)) {
      phase.value = 'results'
    } else if (/preparing|loading|compiling|validating/i.test(line)) {
      phase.value = 'preparing'
    }
  }

  const phases = computed(() => {
    const p = phase.value
    const completedFlows = progress.value.length
    const total = totalFlows.value

    return [
      {
        id: 'preparing',
        label: 'Preparing test flows',
        icon: 'ListTree',
        color: 'blue',
        status: p === 'starting' || p === 'preparing' ? 'active' : 'complete',
        detail:
          p === 'starting'
            ? 'Initializing...'
            : total > 0
              ? `${total} flows loaded`
              : 'Loading flows...',
      },
      {
        id: 'executing',
        label: 'Executing flows',
        icon: 'PlayCircle',
        color: 'amber',
        status:
          p === 'starting' || p === 'preparing'
            ? 'pending'
            : p === 'executing'
              ? 'active'
              : 'complete',
        detail:
          p === 'executing'
            ? `${completedFlows}/${total || '?'} completed`
            : p === 'results' || p === 'complete'
              ? `${completedFlows} flows executed`
              : '',
      },
      {
        id: 'results',
        label: 'Results',
        icon: 'ClipboardCheck',
        color: 'emerald',
        status:
          p === 'results'
            ? 'active'
            : p === 'complete'
              ? 'complete'
              : 'pending',
        detail:
          p === 'complete' || p === 'results'
            ? status.value === 'failed'
              ? 'Test run failed'
              : 'Test run complete'
            : '',
      },
    ]
  })

  const stats = computed(() => {
    const total = progress.value.length
    const passed = progress.value.filter((f) => f.status === 'passed').length
    const failed = progress.value.filter((f) => f.status === 'failed').length
    const rate = total > 0 ? Math.round((passed / total) * 100) : 0
    return { total, passed, failed, rate }
  })

  function setupListeners(tid) {
    stopListening()
    const ws = getWebSocket()
    ws.connect()

    const offStarted = ws.on('test_started', (data) => {
      if (data.testId === tid) {
        status.value = 'running'
        phase.value = 'preparing'
        if (data.totalFlows) {
          totalFlows.value = data.totalFlows
        }
        if (data.name) {
          planName.value = data.name
        }
      }
    })

    const offProgress = ws.on('test_progress', (data) => {
      if (data.testId === tid) {
        if (data.line) {
          logs.value = [...logs.value.slice(-(MAX_LOGS - 1)), data.line]
          detectPhase(data.line)
        }
        if (data.flowName) {
          // Auto-switch to executing phase when first flow result arrives
          if (phase.value === 'starting' || phase.value === 'preparing') {
            phase.value = 'executing'
          }
          const existing = progress.value.find((p) => p.flowName === data.flowName)
          if (existing) {
            existing.status = data.status
            if (data.duration) existing.duration = data.duration
          } else {
            progress.value = [
              ...progress.value,
              { flowName: data.flowName, status: data.status, duration: data.duration || '' },
            ]
          }
        }
      }
    })

    const offCompleted = ws.on('test_completed', (data) => {
      if (data.testId === tid) {
        status.value = 'completed'
        phase.value = 'complete'
        result.value = data
        stopTimer()
      }
    })

    const offFailed = ws.on('test_failed', (data) => {
      if (data.testId === tid) {
        status.value = 'failed'
        phase.value = 'complete'
        result.value = data
        stopTimer()
      }
    })

    cleanups = [offStarted, offProgress, offCompleted, offFailed]
  }

  function startExecution(tid, pId, pName) {
    testId.value = tid
    planId.value = pId || ''
    planName.value = pName || ''
    status.value = 'running'
    phase.value = 'starting'
    logs.value = []
    progress.value = []
    result.value = null
    totalFlows.value = 0

    startTimer()
    setupListeners(tid)
  }

  async function reconnect(tid) {
    testId.value = tid
    status.value = 'running'
    phase.value = 'starting'
    logs.value = []
    progress.value = []
    result.value = null

    try {
      const data = await testsApi.live(tid)

      planName.value = data.planName || ''
      planId.value = data.planId || ''

      if (data.status === 'completed' || data.status === 'passed') {
        // Completed test
        status.value = 'completed'
        phase.value = 'complete'
        if (data.flows) {
          progress.value = data.flows.map((f) => ({
            flowName: f.name,
            status: f.status,
            duration: f.duration || '',
          }))
        }
        totalFlows.value = data.totalFlows || data.flows?.length || 0
        if (data.duration) {
          result.value = data
        }
        return
      }

      if (data.status === 'failed') {
        status.value = 'failed'
        phase.value = 'complete'
        if (data.flows) {
          progress.value = data.flows.map((f) => ({
            flowName: f.name,
            status: f.status,
            duration: f.duration || '',
          }))
        }
        totalFlows.value = data.totalFlows || data.flows?.length || 0
        result.value = data
        return
      }

      // Still running — restore state
      status.value = 'running'
      totalFlows.value = data.totalFlows || 0

      if (data.logs?.length) {
        logs.value = data.logs
      }

      if (data.flows?.length) {
        progress.value = data.flows.map((f) => ({
          flowName: f.name,
          status: f.status,
          duration: f.duration || '',
        }))
        phase.value = 'executing'
      } else {
        phase.value = 'preparing'
      }

      // Restore timer from startedAt
      if (data.startedAt) {
        startTimer(new Date(data.startedAt))
      } else {
        startTimer()
      }

      // Setup WS listeners for ongoing events
      setupListeners(tid)
    } catch {
      // Test not found — likely already completed and cleaned up
      status.value = 'failed'
      phase.value = 'complete'
    }
  }

  function stopListening() {
    cleanups.forEach((fn) => fn())
    cleanups = []
  }

  onUnmounted(() => {
    stopListening()
    stopTimer()
  })

  return {
    status,
    phase,
    logs,
    progress,
    result,
    totalFlows,
    planName,
    planId,
    testId,
    elapsedSeconds,
    phases,
    stats,
    formatElapsed,
    startExecution,
    reconnect,
    stopListening,
  }
}
