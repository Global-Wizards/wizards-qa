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
          <span class="text-sm font-semibold block">Agent Exploring Game</span>
          <span v-if="explorationStatus === 'active' && currentActivity" class="text-xs text-muted-foreground truncate block">
            {{ currentActivity.label }}
          </span>
          <span v-else-if="explorationStatus === 'complete'" class="text-xs text-green-600 dark:text-green-400 block">
            Exploration complete
          </span>
        </div>
      </div>
      <div class="flex items-center gap-4 shrink-0">
        <div v-if="stepCurrent" class="text-right">
          <span class="text-sm font-mono font-semibold block">{{ stepCurrent }}/{{ stepTotal }}</span>
          <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Steps</span>
        </div>
        <div v-if="elapsedSeconds > 0" class="text-right">
          <span class="text-sm font-mono font-semibold block">{{ formatElapsed(elapsedSeconds) }}</span>
          <span class="text-[10px] uppercase text-muted-foreground tracking-wider">Elapsed</span>
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
    <div v-if="toolSteps.length >= 2" class="grid grid-cols-4 gap-px border-b bg-border">
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
    </div>

    <!-- D. Timeline -->
    <div ref="timelineRef" class="max-h-[500px] overflow-y-auto px-4 py-3 relative">
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
                  'flex-1 rounded-md border p-2.5 transition-all cursor-pointer hover:bg-accent/50',
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
                  <span class="text-[10px] text-muted-foreground ml-auto shrink-0 font-mono">{{ entry.durationMs }}ms</span>
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
                <p v-if="entry.reasoning" :class="['text-xs text-muted-foreground mt-1.5', expandedSteps[i] ? '' : 'line-clamp-2']">
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
                    <pre class="bg-muted rounded px-2 py-1 mt-0.5 overflow-x-auto text-[11px] whitespace-pre-wrap">{{ typeof entry.input === 'string' ? entry.input : JSON.stringify(entry.input, null, 2) }}</pre>
                  </div>
                  <div v-if="entry.result" class="mt-2 text-xs">
                    <span class="text-muted-foreground font-medium">Result:</span>
                    <p class="text-muted-foreground mt-0.5 whitespace-pre-wrap">{{ entry.result }}</p>
                  </div>
                  <div v-if="entry.error" class="mt-2 text-xs">
                    <span class="font-medium text-red-500">Error:</span>
                    <p class="text-red-500 mt-0.5 whitespace-pre-wrap">{{ entry.error }}</p>
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
    </div>

    <!-- E. Hint Input Bar -->
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
          ({{ toolSteps.length }} steps{{ elapsedSeconds > 0 ? ', ' + formatElapsed(elapsedSeconds) : '' }})
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import {
  Loader2, CheckCircle, MessageCircle, Send,
  MousePointerClick, Type, ArrowDown, Camera, Search, Terminal, Code,
  Globe, Clock, Plus, Timer, Circle,
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

// --- Computed ---
const toolSteps = computed(() => props.steps.filter(s => s.type === 'tool'))

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
  // Find last tool step index in the full steps array
  let lastToolIdx = -1
  for (let i = props.steps.length - 1; i >= 0; i--) {
    if (props.steps[i].type !== 'hint') { lastToolIdx = i; break }
  }
  return index === lastToolIdx
}

// --- Minimap scroll ---
const timelineRef = ref(null)

function scrollToStep(minimapIndex) {
  // minimapIndex is into toolSteps; find corresponding full steps index
  const toolName = toolSteps.value[minimapIndex]
  if (!toolName) return
  let count = 0
  for (let i = 0; i < props.steps.length; i++) {
    if (props.steps[i].type === 'tool') {
      if (count === minimapIndex) {
        const el = timelineRef.value
        if (!el) return
        const children = el.querySelectorAll('[class*="relative"]')
        // Find the step-entry divs (direct children of the TransitionGroup)
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
    const el = timelineRef.value
    if (!el) return
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
    if (isNearBottom) {
      el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
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
