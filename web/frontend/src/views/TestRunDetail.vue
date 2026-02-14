<template>
  <div class="space-y-4">
    <!-- Back button -->
    <button
      class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      @click="goBack"
    >
      <ArrowLeft class="h-4 w-4" />
      Back to Tests
    </button>

    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- A. Header Banner -->
      <div
        :class="[
          'px-5 py-4 border-b',
          status === 'failed'
            ? 'bg-gradient-to-r from-destructive/10 via-destructive/5 to-transparent'
            : status === 'completed'
              ? 'bg-gradient-to-r from-emerald-500/10 via-transparent to-transparent'
              : 'bg-gradient-to-r from-primary/5 via-transparent to-transparent',
        ]"
      >
        <div class="flex items-center justify-between gap-4">
          <!-- Left: status icon + title -->
          <div class="flex items-center gap-3 min-w-0">
            <div class="relative shrink-0">
              <template v-if="status === 'failed'">
                <div class="h-9 w-9 rounded-full bg-destructive/10 flex items-center justify-center">
                  <XCircle class="h-5 w-5 text-destructive" />
                </div>
              </template>
              <template v-else-if="status === 'completed'">
                <div class="h-9 w-9 rounded-full bg-emerald-500/10 flex items-center justify-center">
                  <CheckCircle2 class="h-5 w-5 text-emerald-500" />
                </div>
              </template>
              <template v-else>
                <div class="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center">
                  <div class="absolute inset-0 rounded-full bg-primary/20 animate-ping" style="animation-duration: 2s;" />
                  <Loader2 class="h-5 w-5 text-primary animate-spin relative z-10" />
                </div>
              </template>
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold">Test Execution</h3>
              <p class="text-xs text-muted-foreground truncate">
                {{ planName || 'Running test plan...' }}
              </p>
            </div>
          </div>

          <!-- Right: segmented progress bar + elapsed time -->
          <div class="flex items-center gap-4 shrink-0">
            <div class="hidden sm:flex items-center gap-1">
              <div
                v-for="p in phases"
                :key="p.id"
                :class="[
                  'h-1.5 w-6 rounded-full transition-all',
                  p.status === 'complete' ? phaseColorMap[p.color]?.barActive || 'bg-green-500' : '',
                  p.status === 'active' ? (phaseColorMap[p.color]?.barActive || 'bg-primary') + ' animate-pulse' : '',
                  p.status === 'pending' ? 'bg-muted-foreground/20' : '',
                ]"
              />
            </div>
            <div v-if="elapsedSeconds > 0" class="text-right">
              <span class="text-sm font-mono font-semibold block">{{ formatElapsed(elapsedSeconds) }}</span>
              <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Elapsed</span>
            </div>
          </div>
        </div>
      </div>

      <!-- B. Phase Timeline -->
      <div class="px-5 py-4 relative">
        <div
          v-if="phases.length > 1"
          class="absolute left-[31px] top-6 bottom-6 w-px bg-border"
        />

        <div class="space-y-1">
          <div v-for="p in phases" :key="p.id" class="relative">
            <div class="flex items-start gap-3 pl-10 py-2">
              <!-- Node -->
              <div
                :class="[
                  'absolute left-[18px] top-3 w-[21px] h-[21px] rounded-full border-2 flex items-center justify-center z-10 transition-all',
                  phaseNodeClasses(p),
                ]"
              >
                <div
                  v-if="p.status === 'active'"
                  class="absolute inset-0 rounded-full animate-ping opacity-40"
                  :class="phaseColorMap[p.color]?.ping || 'border-2 border-primary'"
                />
                <CheckCircle2 v-if="p.status === 'complete'" class="h-3 w-3 text-green-500 relative z-10" />
                <Loader2 v-else-if="p.status === 'active'" class="h-3 w-3 animate-spin relative z-10" :class="phaseColorMap[p.color]?.text || 'text-primary'" />
                <component v-else :is="phaseIconMap[p.icon]" class="h-3 w-3 text-muted-foreground/40 relative z-10" />
              </div>

              <!-- Content -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <component
                    :is="phaseIconMap[p.icon]"
                    :class="[
                      'h-4 w-4 shrink-0',
                      p.status === 'pending' ? 'text-muted-foreground/40' : phaseColorMap[p.color]?.text || 'text-foreground',
                    ]"
                  />
                  <span
                    :class="[
                      'text-sm font-medium',
                      p.status === 'pending' ? 'text-muted-foreground' : 'text-foreground',
                    ]"
                  >
                    {{ p.label }}
                  </span>
                </div>
                <p
                  v-if="p.detail && (p.status === 'active' || p.status === 'complete')"
                  class="text-xs text-muted-foreground mt-0.5 pl-6"
                >
                  {{ p.detail }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Active flow command progress (browser mode) -->
      <div v-if="activeFlow && status === 'running'" class="border-t px-5 py-3 bg-muted/30">
        <div class="flex items-center gap-2 mb-2">
          <Loader2 class="h-3.5 w-3.5 animate-spin text-primary" />
          <span class="text-sm font-medium">{{ activeFlow.flowName }}</span>
          <span class="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
            {{ commandProgress.filter(c => c.flowName === activeFlow.flowName && c.status !== 'running').length }}/{{ activeFlow.commandCount }} commands
          </span>
        </div>
      </div>

      <!-- C. Flow Results -->
      <div v-if="progress.length || pendingFlowSlots.length" class="border-t px-5 py-4">
        <h4 class="text-sm font-medium mb-3">Flow Results</h4>
        <div class="space-y-2">
          <!-- Completed flows -->
          <div
            v-for="(flow, i) in progress"
            :key="flow.flowName"
            class="rounded-md border step-entry overflow-hidden"
            :style="{ animationDelay: `${i * 60}ms` }"
          >
            <div class="flex items-center justify-between p-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <div
                  :class="[
                    'w-2.5 h-2.5 rounded-full shrink-0',
                    flow.status === 'passed' ? 'bg-emerald-500' : 'bg-destructive',
                  ]"
                />
                <span class="text-sm font-medium truncate">{{ flow.flowName }}</span>
                <span
                  v-if="stepScreenshots.filter(s => s.flowName === flow.flowName).length"
                  class="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded"
                >
                  {{ stepScreenshots.filter(s => s.flowName === flow.flowName).length }} steps
                </span>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <span v-if="flow.duration" class="text-[11px] font-mono px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                  {{ flow.duration }}
                </span>
                <span
                  :class="[
                    'text-[11px] font-medium px-2 py-0.5 rounded-full',
                    flow.status === 'passed'
                      ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                      : 'bg-destructive/10 text-destructive',
                  ]"
                >
                  {{ flow.status }}
                </span>
              </div>
            </div>
          </div>

          <!-- Pending flow slots (known total minus completed) -->
          <div
            v-for="i in pendingFlowSlots"
            :key="'pending-' + i"
            class="flex items-center justify-between rounded-md border border-dashed p-3 opacity-40"
          >
            <div class="flex items-center gap-2.5">
              <div class="w-2.5 h-2.5 rounded-full bg-muted-foreground/30" />
              <span class="text-sm text-muted-foreground">Pending...</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Step Navigator (browser mode) -->
      <div v-if="stepScreenshots.length > 0" class="border-t px-5 py-4">
        <h4 class="text-sm font-medium mb-3">Step Navigator</h4>
        <TestStepNavigator :steps="stepScreenshots" :live="status === 'running'" />
      </div>

      <!-- D. Stats Strip -->
      <div
        v-if="stats.total > 0 && (status === 'completed' || status === 'failed' || phase === 'results' || phase === 'complete')"
        class="border-t grid grid-cols-4 divide-x"
      >
        <div class="px-4 py-3 text-center">
          <p class="text-lg font-semibold">{{ stats.total }}</p>
          <p class="text-[11px] text-muted-foreground uppercase tracking-wider">Total</p>
        </div>
        <div class="px-4 py-3 text-center">
          <p class="text-lg font-semibold text-emerald-500">{{ stats.passed }}</p>
          <p class="text-[11px] text-muted-foreground uppercase tracking-wider">Passed</p>
        </div>
        <div class="px-4 py-3 text-center">
          <p class="text-lg font-semibold text-destructive">{{ stats.failed }}</p>
          <p class="text-[11px] text-muted-foreground uppercase tracking-wider">Failed</p>
        </div>
        <div class="px-4 py-3 text-center">
          <p class="text-lg font-semibold">{{ stats.rate }}%</p>
          <p class="text-[11px] text-muted-foreground uppercase tracking-wider">Pass Rate</p>
        </div>
      </div>

      <!-- E. Collapsible Logs Section -->
      <div class="border-t">
        <button
          class="w-full flex items-center justify-between px-5 py-3 hover:bg-muted/50 transition-colors text-left"
          @click="logsOpen = !logsOpen"
        >
          <div class="flex items-center gap-2">
            <Terminal class="h-4 w-4 text-muted-foreground" />
            <span class="text-sm font-medium">Progress Log</span>
            <span class="text-[11px] font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
              {{ logs.length }} lines
            </span>
          </div>
          <div class="flex items-center gap-2">
            <button
              class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded hover:bg-muted transition-colors"
              @click.stop="copyLogs"
            >
              <Copy class="h-3 w-3" />
              Copy
            </button>
            <ChevronDown
              :class="['h-4 w-4 text-muted-foreground transition-transform', logsOpen ? 'rotate-180' : '']"
            />
          </div>
        </button>

        <div v-if="logsOpen" class="px-5 pb-4">
          <div
            ref="logContainer"
            class="max-h-48 overflow-y-auto rounded-md bg-muted p-3 scrollbar-thin relative"
            @scroll="handleLogScroll"
          >
            <div v-for="(line, i) in logs" :key="i" class="flex gap-2">
              <span class="text-[10px] font-mono text-muted-foreground/40 select-none w-6 text-right shrink-0">{{ i + 1 }}</span>
              <p
                :class="[
                  'text-xs font-mono flex-1',
                  lineColor(line),
                ]"
              >
                {{ line }}
              </p>
            </div>
            <p v-if="!logs.length" class="text-xs text-muted-foreground">Waiting for output...</p>
          </div>
          <div v-if="!isAtBottom && logs.length > 5" class="flex justify-center mt-2">
            <button
              class="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground border rounded px-2 py-1 hover:bg-muted transition-colors"
              @click="scrollToBottom"
            >
              <ArrowDown class="h-3 w-3" />
              Scroll to latest
            </button>
          </div>
        </div>
      </div>

      <!-- F. Footer -->
      <div class="border-t px-5 py-3 flex items-center justify-between">
        <span class="text-xs text-muted-foreground">
          {{ completedCount }}/{{ phases.length }} phases complete
        </span>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-1 text-xs border rounded px-3 py-1.5 hover:bg-muted transition-colors"
            @click="goBack"
          >
            Back to Tests
          </button>
          <button
            v-if="(status === 'completed' || status === 'failed') && planId"
            class="inline-flex items-center gap-1 text-xs bg-primary text-primary-foreground rounded px-3 py-1.5 hover:bg-primary/90 transition-colors"
            @click="rerun"
          >
            <RefreshCw class="h-3 w-3" />
            Re-run
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Loader2, CheckCircle2, XCircle, Terminal, Copy, ChevronDown,
  ArrowDown, ArrowLeft, RefreshCw,
  ListTree, ClipboardCheck,
} from 'lucide-vue-next'
import { PlayCircle as PlayCircleIcon } from 'lucide-vue-next'
import { testPlansApi } from '@/lib/api'
import { useTestExecution } from '@/composables/useTestExecution'
import TestStepNavigator from '@/components/TestStepNavigator.vue'

const route = useRoute()
const router = useRouter()

const {
  status,
  phase,
  logs,
  progress,
  result,
  totalFlows,
  planName,
  planId,
  elapsedSeconds,
  phases,
  stats,
  formatElapsed,
  startExecution,
  reconnect,
  stepScreenshots,
  commandProgress,
  activeFlow,
  mode: execMode,
} = useTestExecution()

// Phase maps (matching AnalysisProgressPanel)
const phaseIconMap = { ListTree, PlayCircle: PlayCircleIcon, ClipboardCheck }

const phaseColorMap = {
  blue: {
    border: 'border-blue-400 dark:border-blue-600',
    bg: 'bg-blue-50 dark:bg-blue-950',
    text: 'text-blue-500',
    barActive: 'bg-blue-500',
    ping: 'border-2 border-blue-400',
  },
  amber: {
    border: 'border-amber-400 dark:border-amber-600',
    bg: 'bg-amber-50 dark:bg-amber-950',
    text: 'text-amber-500',
    barActive: 'bg-amber-500',
    ping: 'border-2 border-amber-400',
  },
  emerald: {
    border: 'border-emerald-400 dark:border-emerald-600',
    bg: 'bg-emerald-50 dark:bg-emerald-950',
    text: 'text-emerald-500',
    barActive: 'bg-emerald-500',
    ping: 'border-2 border-emerald-400',
  },
}

function phaseNodeClasses(p) {
  const colors = phaseColorMap[p.color] || phaseColorMap.blue
  if (p.status === 'complete') return 'border-green-500 bg-green-50 dark:bg-green-950'
  if (p.status === 'active') return colors.border + ' ' + colors.bg
  return 'border-muted-foreground/30 bg-muted'
}

const completedCount = computed(() => phases.value.filter((p) => p.status === 'complete').length)

// Pending flow slots: show empty slots for known total minus completed
const pendingFlowSlots = computed(() => {
  if (!totalFlows.value || status.value === 'completed' || status.value === 'failed') return 0
  return Math.max(0, totalFlows.value - progress.value.length)
})

// Logs
const logsOpen = ref(false)
const logContainer = ref(null)
const isAtBottom = ref(true)

function lineColor(line) {
  if (/error|fail|fatal/i.test(line)) return 'text-red-500'
  if (/warn/i.test(line)) return 'text-amber-500'
  return 'text-muted-foreground'
}

function handleLogScroll() {
  const el = logContainer.value
  if (!el) return
  isAtBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

function scrollToBottom() {
  const el = logContainer.value
  if (el) {
    el.scrollTop = el.scrollHeight
    isAtBottom.value = true
  }
}

function copyLogs() {
  const text = logs.value.join('\n')
  navigator.clipboard.writeText(text).catch(() => {})
}

// Auto-scroll logs
watch(() => logs.value.length, () => {
  if (!logsOpen.value) return
  nextTick(() => {
    const el = logContainer.value
    if (!el) return
    if (isAtBottom.value) el.scrollTop = el.scrollHeight
  })
})

// Navigation
const projectId = computed(() => route.params.projectId || '')

function goBack() {
  if (projectId.value) {
    router.push(`/projects/${projectId.value}/tests?tab=plans`)
  } else {
    router.push('/tests?tab=plans')
  }
}

async function rerun() {
  if (!planId.value) return
  try {
    const data = await testPlansApi.run(planId.value, { mode: execMode.value || 'browser' })
    if (projectId.value) {
      router.replace(`/projects/${projectId.value}/tests/run/${data.testId}`)
    } else {
      router.replace(`/tests/run/${data.testId}`)
    }
    startExecution(data.testId, planId.value, planName.value)
  } catch (err) {
    alert('Failed to re-run: ' + (err.message || 'Unknown error'))
  }
}

// Mount: start or reconnect
onMounted(() => {
  const tid = route.params.testId
  if (!tid) return

  // Check if this is a fresh run (navigated from Tests.vue with query param)
  const fresh = route.query.fresh
  const pId = route.query.planId || ''
  const pName = route.query.planName || ''

  if (fresh) {
    startExecution(tid, pId, pName)
  } else {
    reconnect(tid)
  }
})
</script>

<style scoped>
.step-entry {
  animation: stepEntry 0.3s ease-out both;
}
@keyframes stepEntry {
  from {
    opacity: 0;
    transform: translateX(-12px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
</style>
