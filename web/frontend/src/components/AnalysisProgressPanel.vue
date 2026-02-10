<template>
  <div class="space-y-4">
    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- A. Header Banner -->
      <div
        :class="[
          'px-5 py-4 border-b',
          mode === 'error'
            ? 'bg-gradient-to-r from-destructive/10 via-destructive/5 to-transparent'
            : 'bg-gradient-to-r from-primary/5 via-transparent to-transparent',
        ]"
      >
        <div class="flex items-center justify-between gap-4">
          <!-- Left: status icon + title + URL -->
          <div class="flex items-center gap-3 min-w-0">
            <div class="relative shrink-0">
              <template v-if="mode === 'error'">
                <div class="h-9 w-9 rounded-full bg-destructive/10 flex items-center justify-center">
                  <AlertCircle class="h-5 w-5 text-destructive" />
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
              <h3 class="text-sm font-semibold">
                {{ mode === 'error' ? 'Analysis Failed' : 'Analyzing' }}
              </h3>
              <p class="text-xs text-muted-foreground truncate" :title="gameUrl">
                {{ gameUrl }}
              </p>
            </div>
          </div>

          <!-- Right: segmented progress bar + elapsed time -->
          <div class="flex items-center gap-4 shrink-0">
            <!-- Segmented progress bar -->
            <div class="hidden sm:flex items-center gap-1">
              <div
                v-for="phase in phases"
                :key="phase.id"
                :class="[
                  'h-1.5 w-6 rounded-full transition-all',
                  phase.status === 'complete' ? phaseColorMap[phase.color]?.barActive || 'bg-green-500' : '',
                  phase.status === 'active' ? (phaseColorMap[phase.color]?.barActive || 'bg-primary') + ' animate-pulse' : '',
                  phase.status === 'pending' ? 'bg-muted-foreground/20' : '',
                ]"
              />
            </div>
            <div v-if="elapsedSeconds > 0" class="text-right">
              <span class="text-sm font-mono font-semibold block">{{ formatElapsed(elapsedSeconds) }}</span>
              <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Elapsed</span>
            </div>
          </div>
        </div>

        <!-- Error alert (error mode only) -->
        <Alert v-if="mode === 'error' && errorMessage" variant="destructive" class="mt-3">
          <AlertCircle class="h-4 w-4" />
          <AlertTitle>{{ errorMessage }}</AlertTitle>
          <AlertDescription v-if="failedPhaseLabel">
            Failed during: {{ failedPhaseLabel }}
          </AlertDescription>
        </Alert>
      </div>

      <!-- B. Phase Timeline -->
      <div class="px-5 py-4 relative">
        <!-- Vertical connector line -->
        <div
          v-if="phases.length > 1"
          class="absolute left-[31px] top-6 bottom-6 w-px bg-border"
        />

        <div class="space-y-1">
          <div v-for="(phase, idx) in phases" :key="phase.id" class="relative">
            <!-- Phase row -->
            <div class="flex items-start gap-3 pl-10 py-2">
              <!-- Node on the timeline line -->
              <div
                :class="[
                  'absolute left-[18px] top-3 w-[21px] h-[21px] rounded-full border-2 flex items-center justify-center z-10 transition-all',
                  phaseNodeClasses(phase),
                ]"
              >
                <!-- Ping ring for active -->
                <div
                  v-if="phase.status === 'active'"
                  class="absolute inset-0 rounded-full animate-ping opacity-40"
                  :class="phaseColorMap[phase.color]?.ping || 'border-2 border-primary'"
                />
                <!-- Icon -->
                <CheckCircle2 v-if="phase.status === 'complete'" class="h-3 w-3 text-green-500 relative z-10" />
                <Loader2 v-else-if="phase.status === 'active'" class="h-3 w-3 animate-spin relative z-10" :class="phaseColorMap[phase.color]?.text || 'text-primary'" />
                <component v-else :is="phaseIconMap[phase.icon]" class="h-3 w-3 text-muted-foreground/40 relative z-10" />
              </div>

              <!-- Content -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <component
                    :is="phaseIconMap[phase.icon]"
                    :class="[
                      'h-4 w-4 shrink-0',
                      phase.status === 'pending' ? 'text-muted-foreground/40' : phaseColorMap[phase.color]?.text || 'text-foreground',
                    ]"
                  />
                  <span
                    :class="[
                      'text-sm font-medium',
                      phase.status === 'pending' ? 'text-muted-foreground' : 'text-foreground',
                    ]"
                  >
                    {{ phase.label }}
                  </span>
                  <!-- Duration badge -->
                  <span
                    v-if="phase.status === 'complete' && phase.durationSeconds"
                    class="ml-auto text-[11px] font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded shrink-0"
                  >
                    {{ phase.durationSeconds }}s
                  </span>
                </div>

                <!-- Detail text -->
                <p
                  v-if="phase.detail && (phase.status === 'active' || phase.status === 'complete')"
                  class="text-xs text-muted-foreground mt-0.5 pl-6"
                >
                  {{ phase.detail }}
                </p>

                <!-- Sub-details as chips -->
                <div
                  v-if="phase.subDetails?.length && (phase.status === 'active' || phase.status === 'complete')"
                  class="flex flex-wrap gap-1.5 mt-2 pl-6"
                >
                  <span
                    v-for="(item, si) in visibleSubDetails(phase, idx)"
                    :key="si"
                    class="inline-flex items-center gap-1 text-[11px] border rounded-full px-2 py-0.5 text-muted-foreground bg-muted/50"
                  >
                    <span class="text-muted-foreground/70">{{ item.label }}:</span>
                    <span>{{ item.value }}</span>
                  </span>
                  <button
                    v-if="phase.subDetails.length > 4 && !expandedPhases[idx]"
                    class="text-[11px] text-primary hover:underline cursor-pointer"
                    @click="expandedPhases[idx] = true"
                  >
                    +{{ phase.subDetails.length - 4 }} more
                  </button>
                  <button
                    v-else-if="phase.subDetails.length > 4 && expandedPhases[idx]"
                    class="text-[11px] text-primary hover:underline cursor-pointer"
                    @click="expandedPhases[idx] = false"
                  >
                    show less
                  </button>
                </div>
              </div>
            </div>

            <!-- Agent exploration slot -->
            <div
              v-if="phase.isAgentSlot && showAgentPanel"
              class="ml-10 my-1"
            >
              <slot name="agent-exploration" />
            </div>
          </div>
        </div>
      </div>

      <!-- C. Collapsible Logs Section -->
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
            <Button
              variant="ghost"
              size="sm"
              class="h-7 text-xs gap-1"
              @click.stop="$emit('copy-log')"
            >
              <Copy class="h-3 w-3" />
              Copy
            </Button>
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
          <!-- Scroll to latest button -->
          <div v-if="logsOpen && !isAtBottom && logs.length > 5" class="flex justify-center mt-2">
            <Button variant="outline" size="sm" class="h-6 text-[11px] gap-1" @click="scrollToBottom">
              <ArrowDown class="h-3 w-3" />
              Scroll to latest
            </Button>
          </div>
        </div>
      </div>

      <!-- D. Footer -->
      <div class="border-t px-5 py-3 flex items-center justify-between">
        <span class="text-xs text-muted-foreground">
          <template v-if="mode === 'error'">
            Failed after {{ formatElapsed(elapsedSeconds) }}
          </template>
          <template v-else>
            {{ completedCount }}/{{ phases.length }} phases complete
          </template>
        </span>

        <div class="flex items-center gap-2">
          <template v-if="mode === 'error'">
            <Button v-if="canContinue" size="sm" @click="$emit('continue')">
              <PlayCircle class="h-4 w-4 mr-1" />
              Continue
            </Button>
            <Button size="sm" :variant="canContinue ? 'secondary' : 'default'" @click="$emit('retry')">
              <RefreshCw class="h-4 w-4 mr-1" />
              Retry
            </Button>
            <Button variant="outline" size="sm" @click="$emit('start-over')">
              Start Over
            </Button>
          </template>
          <template v-else>
            <Button variant="outline" size="sm" @click="$emit('cancel')">
              Cancel
            </Button>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import {
  Loader2, AlertCircle, CheckCircle2, Terminal, Copy, ChevronDown,
  ArrowDown, RefreshCw, PlayCircle,
  Radar, Bot, Brain, ListTree, ClipboardCheck,
} from 'lucide-vue-next'
import { PlayCircle as PlayCircleIcon } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'

const props = defineProps({
  mode: { type: String, default: 'progress' },         // 'progress' | 'error'
  gameUrl: { type: String, default: '' },
  elapsedSeconds: { type: Number, default: 0 },
  formatElapsed: { type: Function, default: (s) => `${Math.floor(s / 60)}m ${s % 60}s` },
  agentMode: { type: Boolean, default: false },
  phases: { type: Array, default: () => [] },
  showAgentPanel: { type: Boolean, default: false },
  logs: { type: Array, default: () => [] },
  errorMessage: { type: String, default: '' },
  failedPhaseLabel: { type: String, default: '' },
  canContinue: { type: Boolean, default: false },
})

defineEmits(['cancel', 'retry', 'continue', 'start-over', 'copy-log'])

// --- Phase icon map ---
const phaseIconMap = {
  Radar,
  Bot,
  Brain,
  ListTree,
  PlayCircle: PlayCircleIcon,
  ClipboardCheck,
}

// --- Phase color map ---
const phaseColorMap = {
  blue: {
    border: 'border-blue-400 dark:border-blue-600',
    bg: 'bg-blue-50 dark:bg-blue-950',
    text: 'text-blue-500',
    barActive: 'bg-blue-500',
    ping: 'border-2 border-blue-400',
  },
  purple: {
    border: 'border-purple-400 dark:border-purple-600',
    bg: 'bg-purple-50 dark:bg-purple-950',
    text: 'text-purple-500',
    barActive: 'bg-purple-500',
    ping: 'border-2 border-purple-400',
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
  rose: {
    border: 'border-rose-400 dark:border-rose-600',
    bg: 'bg-rose-50 dark:bg-rose-950',
    text: 'text-rose-500',
    barActive: 'bg-rose-500',
    ping: 'border-2 border-rose-400',
  },
  sky: {
    border: 'border-sky-400 dark:border-sky-600',
    bg: 'bg-sky-50 dark:bg-sky-950',
    text: 'text-sky-500',
    barActive: 'bg-sky-500',
    ping: 'border-2 border-sky-400',
  },
}

function phaseNodeClasses(phase) {
  const colors = phaseColorMap[phase.color] || phaseColorMap.blue
  if (phase.status === 'complete') {
    return 'border-green-500 bg-green-50 dark:bg-green-950'
  }
  if (phase.status === 'active') {
    return colors.border + ' ' + colors.bg
  }
  return 'border-muted-foreground/30 bg-muted'
}

// --- Sub-detail expand/collapse ---
const expandedPhases = ref({})

function visibleSubDetails(phase, idx) {
  if (!phase.subDetails) return []
  if (expandedPhases.value[idx] || phase.subDetails.length <= 4) {
    return phase.subDetails
  }
  return phase.subDetails.slice(0, 4)
}

// --- Completed count ---
const completedCount = computed(() => {
  return props.phases.filter(p => p.status === 'complete').length
})

// --- Logs ---
const logsOpen = ref(false)
const logContainer = ref(null)
const isAtBottom = ref(true)

// Auto-open logs in error mode
onMounted(() => {
  if (props.mode === 'error') {
    logsOpen.value = true
  }
})

watch(() => props.mode, (val) => {
  if (val === 'error') {
    logsOpen.value = true
  }
})

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

// Auto-scroll on new log entries when near bottom
watch(() => props.logs.length, () => {
  if (!logsOpen.value) return
  nextTick(() => {
    const el = logContainer.value
    if (!el) return
    if (isAtBottom.value) {
      el.scrollTop = el.scrollHeight
    }
  })
})
</script>

<style scoped>
.phase-node-pop {
  animation: nodePop 0.3s ease-out;
}
@keyframes nodePop {
  0% { transform: scale(0.6); opacity: 0; }
  70% { transform: scale(1.15); }
  100% { transform: scale(1); opacity: 1; }
}
</style>
