<template>
  <div class="rounded-lg border bg-card my-2 overflow-hidden">
    <!-- A. Current Activity Banner -->
    <div class="flex items-center justify-between px-4 py-3 border-b bg-gradient-to-r from-primary/5 via-transparent to-transparent">
      <div class="flex items-center gap-3 min-w-0">
        <!-- Animated status icon -->
        <div class="relative shrink-0">
          <div v-if="explorationStatus === 'active'" class="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
            <div class="absolute inset-0 rounded-full bg-primary/20 animate-ping" style="animation-duration: 2s;" />
            <Loader2 class="h-4 w-4 text-primary animate-spin relative z-10" />
          </div>
          <div v-else class="h-8 w-8 rounded-full bg-green-500/10 flex items-center justify-center">
            <CheckCircle class="h-4 w-4 text-green-500" />
          </div>
        </div>
        <div class="min-w-0">
          <span class="text-sm font-semibold block">Agent Exploring Game<template v-if="deviceLabel"> <span class="text-xs font-normal text-muted-foreground">[{{ deviceLabel }}]</span></template></span>
          <span v-if="explorationStatus === 'active' && currentActivity" class="text-xs text-muted-foreground truncate block">
            {{ currentActivity.label }}
          </span>
          <span v-else-if="explorationStatus === 'complete'" class="text-xs text-green-600 dark:text-green-400 block">
            Exploration complete
          </span>
        </div>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <!-- Steps with progress ring -->
        <div v-if="stepCurrent" class="flex items-center gap-2">
          <div class="relative h-5 w-5 shrink-0">
            <svg class="h-5 w-5 -rotate-90" viewBox="0 0 20 20">
              <circle cx="10" cy="10" r="8" fill="none" stroke="currentColor" class="text-muted-foreground/15" stroke-width="2.5" />
              <circle cx="10" cy="10" r="8" fill="none" stroke="currentColor" class="text-primary" stroke-width="2.5"
                :stroke-dasharray="stepProgressCircumference"
                :stroke-dashoffset="stepProgressOffset"
                stroke-linecap="round"
              />
            </svg>
          </div>
          <div class="text-right">
            <span class="text-sm font-mono font-semibold block leading-tight">{{ stepCurrent }}/{{ stepTotal }}</span>
            <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Steps</span>
          </div>
        </div>
        <!-- Elapsed timer -->
        <div v-if="elapsedSeconds > 0" class="text-right">
          <span class="text-sm font-mono font-semibold text-primary block leading-tight">{{ formatElapsed(elapsedSeconds) }}</span>
          <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Elapsed</span>
        </div>
        <!-- Avg per step -->
        <div v-if="avgStepMs > 0" class="text-right">
          <span class="text-sm font-mono font-semibold block leading-tight">{{ formatMs(avgStepMs) }}</span>
          <span class="text-[10px] uppercase text-muted-foreground tracking-wider flex items-center gap-0.5 justify-end">
            <Zap class="h-2.5 w-2.5" />Avg/Step
          </span>
        </div>
        <!-- Credits used -->
        <div v-if="liveStepCredits > 0" class="text-right">
          <span class="text-sm font-mono font-semibold text-amber-500 block leading-tight">{{ liveStepCredits }}</span>
          <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Credits</span>
        </div>
      </div>
    </div>

    <!-- B. Exploration Mini-Map -->
    <div v-if="toolSteps.length > 1" class="px-4 py-2 border-b overflow-x-auto scrollbar-thin">
      <div class="flex items-center gap-1.5 min-w-max">
        <Tooltip v-for="(entry, i) in toolSteps" :key="'dot-' + i">
          <TooltipTrigger as-child>
            <button
              :class="[
                'rounded-full transition-all shrink-0 cursor-pointer',
                i === toolSteps.length - 1 && explorationStatus === 'active'
                  ? 'w-3 h-3 ring-2 ring-primary/40 ring-offset-1 ring-offset-background'
                  : 'w-2 h-2',
                dotColor(entry),
              ]"
              @click="scrollToStep(i)"
            />
          </TooltipTrigger>
          <TooltipContent side="bottom" class="text-xs">
            Step {{ entry.stepNumber }}: {{ getToolMeta(entry.toolName).label }}
            <span v-if="entry.durationMs" class="text-muted-foreground">({{ formatMs(entry.durationMs) }})</span>
          </TooltipContent>
        </Tooltip>
        <!-- Remaining placeholder dots -->
        <template v-if="explorationStatus === 'active' && stepTotal > toolSteps.length">
          <div
            v-for="j in Math.min(stepTotal - toolSteps.length, 20)"
            :key="'placeholder-' + j"
            class="w-1.5 h-1.5 rounded-full bg-muted-foreground/15 shrink-0"
          />
        </template>
      </div>
    </div>

    <!-- C. Stats Strip -->
    <div v-if="toolSteps.length >= 2" class="grid grid-cols-5 gap-px border-b bg-border">
      <div class="bg-card px-3 py-2 text-center">
        <span class="text-lg font-mono font-semibold block leading-tight">{{ statsData.totalSteps }}</span>
        <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Steps</span>
      </div>
      <div class="bg-card px-3 py-2 text-center">
        <span class="text-lg font-mono font-semibold text-amber-500 block leading-tight">{{ statsData.screenshots }}</span>
        <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Screenshots</span>
      </div>
      <div class="bg-card px-3 py-2 text-center">
        <span class="text-lg font-mono font-semibold text-emerald-500 block leading-tight">{{ statsData.interactions }}</span>
        <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Actions</span>
      </div>
      <div class="bg-card px-3 py-2 text-center">
        <span :class="['text-lg font-mono font-semibold block leading-tight', statsData.errors > 0 ? 'text-red-500' : 'text-muted-foreground/40']">{{ statsData.errors }}</span>
        <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Errors</span>
      </div>
      <div class="bg-card px-3 py-2 text-center">
        <span class="text-lg font-mono font-semibold text-primary block leading-tight">{{ formatMs(totalDurationMs) }}</span>
        <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Total Time</span>
      </div>
    </div>

    <!-- D. Timeline -->
    <div ref="timelineRef" class="max-h-[500px] overflow-y-auto px-4 py-3 relative scrollbar-thin" @scroll="handleTimelineScroll">
      <!-- Vertical connector line -->
      <div v-if="steps.length > 0" class="absolute left-[27px] top-4 bottom-4 w-px bg-border" />

      <TransitionGroup name="step-entry" tag="div" class="space-y-2 relative">
        <div v-for="(entry, i) in steps" :key="'step-' + i" class="relative">
          <!-- Hint entry -->
          <template v-if="entry.type === 'hint'">
            <div class="flex items-start gap-3 pl-10">
              <!-- Node on line -->
              <div class="absolute left-[18px] top-2 w-[21px] h-[21px] rounded-full border-2 border-blue-400 bg-blue-50 dark:bg-blue-950 flex items-center justify-center z-10">
                <MessageCircle class="h-2.5 w-2.5 text-blue-500" />
              </div>
              <!-- Hint card -->
              <div class="flex-1 rounded-md border border-blue-200 dark:border-blue-800 bg-blue-50/50 dark:bg-blue-950/30 p-2.5">
                <div class="flex items-center gap-2 mb-1">
                  <Badge variant="outline" class="text-[10px] px-1.5 py-0 border-blue-300 dark:border-blue-700 text-blue-600 dark:text-blue-400">Hint</Badge>
                </div>
                <p class="text-xs text-blue-700 dark:text-blue-300">{{ entry.message }}</p>
              </div>
            </div>
          </template>

          <!-- Tool step entry -->
          <template v-else>
            <div class="flex items-start gap-3 pl-10">
              <!-- Node on line -->
              <div
                :class="[
                  'absolute left-[18px] top-2 w-[21px] h-[21px] rounded-full border-2 flex items-center justify-center z-10',
                  entry.error ? 'border-red-400 bg-red-50 dark:bg-red-950' : nodeColorClasses(entry).border,
                  entry.error ? '' : nodeColorClasses(entry).bg,
                ]"
              >
                <div v-if="isLatestStep(i) && explorationStatus === 'active'" class="absolute inset-0 rounded-full border-2 border-primary animate-ping opacity-40" />
                <component :is="getToolMeta(entry.toolName).icon" :class="['h-2.5 w-2.5 relative z-10', entry.error ? 'text-red-500' : nodeColorClasses(entry).text]" />
              </div>

              <!-- Step card -->
              <div
                :class="[
                  'flex-1 rounded-md border p-2.5 transition-all cursor-pointer hover:bg-accent/50 overflow-hidden min-w-0',
                  isLatestStep(i) && explorationStatus === 'active' ? 'border-primary/30 glow-primary-sm' : '',
                  entry.error ? 'border-red-200 dark:border-red-800/50' : '',
                ]"
                @click="toggleExpanded(i)"
              >
                <!-- Header -->
                <div class="flex items-center gap-2">
                  <Badge variant="outline" class="shrink-0 text-[10px] px-1.5 py-0 font-mono">{{ entry.stepNumber }}</Badge>
                  <component :is="getToolMeta(entry.toolName).icon" :class="['h-3 w-3 shrink-0', entry.error ? 'text-red-500' : nodeColorClasses(entry).text]" />
                  <span class="text-xs font-medium truncate">{{ getToolMeta(entry.toolName).label }}</span>
                  <!-- AI thinking time -->
                  <span v-if="entry.thinkingMs" :class="['text-[10px] font-mono shrink-0 flex items-center gap-0.5', gapColor(entry.thinkingMs)]">
                    <Brain class="h-2.5 w-2.5" />
                    AI: {{ formatMs(entry.thinkingMs) }}
                  </span>
                  <span v-else-if="stepGapMs(i) != null" :class="['text-[10px] font-mono shrink-0 flex items-center gap-0.5', gapColor(stepGapMs(i))]">
                    <Clock class="h-2.5 w-2.5" />
                    +{{ formatMs(stepGapMs(i)) }}
                  </span>
                  <!-- Duration pill -->
                  <span v-if="entry.durationMs" :class="['text-[10px] font-mono px-1.5 py-0.5 rounded-full shrink-0', durationColor(entry.durationMs)]">
                    {{ formatMs(entry.durationMs) }}
                  </span>
                  <!-- Credits -->
                  <span v-if="entry.credits" class="text-[10px] font-mono px-1.5 py-0.5 rounded-full bg-amber-500/10 text-amber-600 dark:text-amber-400 shrink-0">
                    {{ entry.credits }} cr
                  </span>
                  <!-- Cumulative timestamp -->
                  <span v-if="cumulativeTime(i) != null" class="text-[10px] text-muted-foreground/50 font-mono shrink-0 ml-auto">
                    @{{ formatTimestamp(cumulativeTime(i)) }}
                  </span>
                </div>

                <!-- Screenshot thumbnail -->
                <img
                  v-if="entry.screenshotB64 || entry.screenshotUrl"
                  :src="entry.screenshotB64 ? 'data:image/jpeg;base64,' + entry.screenshotB64 : entry.screenshotUrl"
                  class="mt-2 max-w-[200px] h-auto rounded border cursor-pointer shrink-0 object-contain"
                  alt="Step screenshot"
                  @click.stop="$emit('expand-screenshot', entry)"
                />

                <!-- Reasoning (collapsed: 2-line clamp) -->
                <p v-if="entry.reasoning" :class="['text-xs text-muted-foreground mt-1.5 break-words', expandedSteps[i] ? '' : 'line-clamp-2']">
                  {{ entry.reasoning }}
                </p>

                <!-- Error indicator (collapsed) -->
                <p v-if="entry.error && !expandedSteps[i]" class="text-xs text-red-500 mt-1 truncate">
                  {{ entry.error }}
                </p>

                <!-- Expanded details -->
                <template v-if="expandedSteps[i]">
                  <div v-if="entry.input" class="mt-2 text-xs">
                    <span class="text-muted-foreground font-medium">Input:</span>
                    <pre class="bg-muted rounded px-2 py-1 mt-0.5 overflow-hidden text-[11px] whitespace-pre-wrap break-words">{{ typeof entry.input === 'string' ? entry.input : JSON.stringify(entry.input, null, 2) }}</pre>
                  </div>
                  <div v-if="entry.result" class="mt-2 text-xs">
                    <span class="text-muted-foreground font-medium">Result:</span>
                    <p class="text-muted-foreground mt-0.5 whitespace-pre-wrap break-words">{{ entry.result }}</p>
                  </div>
                  <div v-if="entry.error" class="mt-2 text-xs">
                    <span class="font-medium text-red-500">Error:</span>
                    <p class="text-red-500 mt-0.5 whitespace-pre-wrap break-words">{{ entry.error }}</p>
                  </div>
                </template>

                <!-- Collapsed result (non-expanded) -->
                <p v-if="!expandedSteps[i] && entry.result && !entry.error" class="text-[11px] text-muted-foreground truncate mt-0.5">
                  {{ entry.result }}
                </p>
              </div>
            </div>
          </template>
        </div>
      </TransitionGroup>

      <!-- "Thinking" skeleton when active -->
      <div v-if="explorationStatus === 'active'" class="relative pl-10 mt-2">
        <div class="absolute left-[18px] top-2 w-[21px] h-[21px] rounded-full border-2 border-dashed border-muted-foreground/30 flex items-center justify-center z-10">
          <Loader2 class="h-2.5 w-2.5 text-muted-foreground/50 animate-spin" />
        </div>
        <div class="flex-1 rounded-md border border-dashed border-muted-foreground/20 p-2.5">
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 class="h-3 w-3 animate-spin" />
            <span>Thinking about next step...</span>
          </div>
        </div>
      </div>

      <!-- Bottom fade gradient -->
      <div v-if="steps.length > 3" class="sticky bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-card to-transparent pointer-events-none" />

      <!-- Scroll-to-bottom button -->
      <button
        v-if="userScrolledAway && explorationStatus === 'active'"
        class="sticky bottom-2 left-1/2 -translate-x-1/2 z-20 bg-primary text-primary-foreground rounded-full p-1.5 shadow-lg hover:bg-primary/90 transition-opacity"
        @click="scrollToBottom"
      >
        <ChevronDown class="h-4 w-4" />
      </button>
    </div>

    <!-- E. Hint Input Bar / Completion Footer -->
    <div class="border-t px-4 py-3">
      <div v-if="explorationStatus === 'active'" class="flex gap-2">
        <div class="flex-1 relative">
          <MessageCircle class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            v-model="hintInput"
            placeholder="Send a hint to the agent..."
            :disabled="hintCooldown"
            class="pl-8 text-sm"
            @keyup.enter="handleSendHint"
          />
        </div>
        <Button
          size="sm"
          :disabled="!hintInput.trim() || hintCooldown"
          @click="handleSendHint"
        >
          <Send class="h-3.5 w-3.5 mr-1" />
          {{ hintSent ? 'Sent!' : hintCooldown ? 'Wait...' : 'Send' }}
        </Button>
      </div>
      <div v-else class="flex items-center gap-2 text-xs text-muted-foreground">
        <CheckCircle class="h-3.5 w-3.5 text-green-500" />
        <span>
          Exploration complete
          <template v-if="toolSteps.length > 0">
            — {{ toolSteps.length }} steps
            <template v-if="elapsedSeconds > 0"> in {{ formatElapsed(elapsedSeconds) }}</template>
            <template v-if="avgStepMs > 0"> (avg {{ formatMs(avgStepMs) }}/step</template>
            <template v-if="fastestStepMs > 0 && fastestStepMs < Infinity">, fastest {{ formatMs(fastestStepMs) }}</template>
            <template v-if="slowestStepMs > 0">, slowest {{ formatMs(slowestStepMs) }})</template>
          </template>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import {
  Loader2, CheckCircle, MessageCircle, Send, ChevronDown,
  MousePointerClick, Type, ArrowDown, Camera, Search, Terminal, Code,
  Globe, Clock, Plus, Timer, Circle, Zap, Brain,
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'

const props = defineProps({
  steps: { type: Array, default: () => [] },
  stepCurrent: { type: Number, default: 0 },
  stepTotal: { type: Number, default: 0 },
  explorationStatus: { type: String, default: 'pending' },
  elapsedSeconds: { type: Number, default: 0 },
  hintCooldown: { type: Boolean, default: false },
  formatElapsed: { type: Function, default: (s) => `${Math.floor(s / 60)}m ${s % 60}s` },
  liveStepCredits: { type: Number, default: 0 },
  deviceLabel: { type: String, default: '' },
})

const emit = defineEmits(['send-hint', 'expand-screenshot'])

// --- Tool classification map ---
const TOOL_META = {
  click:              { icon: MousePointerClick, label: 'Click',              category: 'interaction', color: 'emerald' },
  type_text:          { icon: Type,              label: 'Type Text',          category: 'interaction', color: 'emerald' },
  scroll:             { icon: ArrowDown,         label: 'Scroll',             category: 'interaction', color: 'emerald' },
  screenshot:         { icon: Camera,            label: 'Screenshot',         category: 'observation', color: 'amber' },
  get_page_info:      { icon: Search,            label: 'Page Info',          category: 'observation', color: 'amber' },
  console_logs:       { icon: Terminal,           label: 'Console Logs',      category: 'observation', color: 'amber' },
  evaluate_js:        { icon: Code,              label: 'Evaluate JS',        category: 'observation', color: 'amber' },
  navigate:           { icon: Globe,             label: 'Navigate',           category: 'navigation', color: 'blue' },
  wait:               { icon: Clock,             label: 'Wait',               category: 'navigation', color: 'blue' },
  request_more_steps: { icon: Plus,              label: 'More Steps',         category: 'meta',       color: 'purple' },
  request_more_time:  { icon: Timer,             label: 'More Time',          category: 'meta',       color: 'purple' },
}
const DEFAULT_META = { icon: Circle, label: 'Unknown', category: 'unknown', color: 'gray' }

function getToolMeta(toolName) {
  return TOOL_META[toolName] || DEFAULT_META
}

// --- Color classes ---
const COLOR_MAP = {
  emerald: {
    border: 'border-emerald-400 dark:border-emerald-600',
    bg: 'bg-emerald-50 dark:bg-emerald-950',
    text: 'text-emerald-500',
    dot: 'bg-emerald-400',
  },
  amber: {
    border: 'border-amber-400 dark:border-amber-600',
    bg: 'bg-amber-50 dark:bg-amber-950',
    text: 'text-amber-500',
    dot: 'bg-amber-400',
  },
  blue: {
    border: 'border-blue-400 dark:border-blue-600',
    bg: 'bg-blue-50 dark:bg-blue-950',
    text: 'text-blue-500',
    dot: 'bg-blue-400',
  },
  purple: {
    border: 'border-purple-400 dark:border-purple-600',
    bg: 'bg-purple-50 dark:bg-purple-950',
    text: 'text-purple-500',
    dot: 'bg-purple-400',
  },
  gray: {
    border: 'border-gray-400 dark:border-gray-600',
    bg: 'bg-gray-50 dark:bg-gray-900',
    text: 'text-gray-500',
    dot: 'bg-gray-400',
  },
}

function nodeColorClasses(entry) {
  const meta = getToolMeta(entry.toolName)
  return COLOR_MAP[meta.color] || COLOR_MAP.gray
}

function dotColor(entry) {
  if (entry.error) return 'bg-red-400'
  const meta = getToolMeta(entry.toolName)
  return (COLOR_MAP[meta.color] || COLOR_MAP.gray).dot
}

// --- Timing helpers ---
function formatMs(ms) {
  if (ms == null || ms === 0) return '0ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const m = Math.floor(ms / 60000)
  const s = Math.round((ms % 60000) / 1000)
  return `${m}m ${s}s`
}

function formatTimestamp(ms) {
  if (ms == null) return ''
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function durationColor(ms) {
  if (ms < 500) return 'bg-emerald-100 dark:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300'
  if (ms < 2000) return 'bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300'
  return 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300'
}

function gapColor(ms) {
  if (ms < 2000) return 'text-emerald-500'
  if (ms < 5000) return 'text-amber-500'
  return 'text-red-500'
}

// --- Computed ---
const toolSteps = computed(() => props.steps.filter(s => s.type === 'tool'))

const totalDurationMs = computed(() =>
  toolSteps.value.reduce((sum, s) => sum + (s.durationMs || 0), 0)
)

const avgStepMs = computed(() =>
  toolSteps.value.length ? Math.round(totalDurationMs.value / toolSteps.value.length) : 0
)

const fastestStepMs = computed(() =>
  toolSteps.value.length ? Math.min(...toolSteps.value.map(s => s.durationMs || Infinity)) : 0
)

const slowestStepMs = computed(() =>
  toolSteps.value.length ? Math.max(...toolSteps.value.map(s => s.durationMs || 0)) : 0
)

// Progress ring calculations
const stepProgressCircumference = computed(() => 2 * Math.PI * 8) // r=8
const stepProgressOffset = computed(() => {
  if (!props.stepTotal) return stepProgressCircumference.value
  const pct = props.stepCurrent / props.stepTotal
  return stepProgressCircumference.value * (1 - pct)
})

// Gap between consecutive steps (thinking time)
function stepGapMs(index) {
  const step = props.steps[index]
  if (!step || step.type === 'hint') return null
  const toolIndex = toolSteps.value.indexOf(step)
  if (toolIndex <= 0) return null
  const prev = toolSteps.value[toolIndex - 1]
  const curr = toolSteps.value[toolIndex]
  if (!prev.timestamp || !curr.timestamp) return null
  const gap = curr.timestamp - prev.timestamp - (prev.durationMs || 0)
  return gap > 0 ? gap : null
}

// Cumulative time offset from start
function cumulativeTime(index) {
  const step = props.steps[index]
  if (!step || step.type === 'hint' || !step.timestamp) return null
  const firstTool = toolSteps.value[0]
  if (!firstTool?.timestamp) return null
  const offset = step.timestamp - firstTool.timestamp
  return offset >= 0 ? offset : null
}

const statsData = computed(() => {
  const tools = toolSteps.value
  return {
    totalSteps: tools.length,
    screenshots: tools.filter(s => s.toolName === 'screenshot').length,
    interactions: tools.filter(s => {
      const meta = getToolMeta(s.toolName)
      return meta.category === 'interaction'
    }).length,
    errors: tools.filter(s => s.error).length,
  }
})

const currentActivity = computed(() => {
  if (toolSteps.value.length === 0) return { label: 'Starting exploration...' }
  const last = toolSteps.value[toolSteps.value.length - 1]
  const meta = getToolMeta(last.toolName)
  return { label: `${meta.label}...`, icon: meta.icon }
})

// --- Step expansion ---
const expandedSteps = ref({})

function toggleExpanded(index) {
  expandedSteps.value = { ...expandedSteps.value, [index]: !expandedSteps.value[index] }
}

function isLatestStep(index) {
  let lastToolIdx = -1
  for (let i = props.steps.length - 1; i >= 0; i--) {
    if (props.steps[i].type !== 'hint') { lastToolIdx = i; break }
  }
  return index === lastToolIdx
}

// --- Minimap scroll ---
const timelineRef = ref(null)
const userScrolledAway = ref(false)

function handleTimelineScroll() {
  const el = timelineRef.value
  if (!el) return
  userScrolledAway.value = el.scrollHeight - el.scrollTop - el.clientHeight > 120
}

function scrollToBottom() {
  const el = timelineRef.value
  if (!el) return
  el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
  userScrolledAway.value = false
}

function scrollToStep(minimapIndex) {
  const toolName = toolSteps.value[minimapIndex]
  if (!toolName) return
  let count = 0
  for (let i = 0; i < props.steps.length; i++) {
    if (props.steps[i].type === 'tool') {
      if (count === minimapIndex) {
        const el = timelineRef.value
        if (!el) return
        const stepEntries = el.querySelector('.space-y-2')?.children
        if (stepEntries && stepEntries[i]) {
          stepEntries[i].scrollIntoView({ behavior: 'smooth', block: 'center' })
        }
        return
      }
      count++
    }
  }
}

// --- Hint state ---
const hintInput = ref('')
const hintSent = ref(false)
let hintSentTimeout = null

async function handleSendHint() {
  if (!hintInput.value.trim()) return
  emit('send-hint', hintInput.value)
  hintInput.value = ''
  hintSent.value = true
  if (hintSentTimeout) clearTimeout(hintSentTimeout)
  hintSentTimeout = setTimeout(() => { hintSent.value = false }, 2000)
}

onUnmounted(() => {
  if (hintSentTimeout != null) clearTimeout(hintSentTimeout)
})

// --- Auto-scroll timeline ---
watch(() => props.steps.length, () => {
  nextTick(() => {
    if (!userScrolledAway.value) {
      scrollToBottom()
    }
  })
})
</script>

<style scoped>
.step-entry-enter-active {
  transition: all 0.3s ease-out;
}
.step-entry-enter-from {
  opacity: 0;
  transform: translateX(-12px);
}
.step-entry-leave-active {
  transition: all 0.2s ease-in;
}
.step-entry-leave-to {
  opacity: 0;
}
</style>
