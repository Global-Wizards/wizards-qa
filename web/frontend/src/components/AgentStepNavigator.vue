<template>
  <div v-if="steps.length" class="rounded-lg border bg-card">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b">
      <span class="text-sm font-medium">Agent Steps ({{ steps.length }})</span>
      <div class="flex items-center gap-1">
        <Button variant="ghost" size="sm" :disabled="selectedIndex <= 0" @click="selectedIndex--">
          <ChevronLeft class="h-4 w-4" />
        </Button>
        <span class="text-xs text-muted-foreground min-w-[4rem] text-center">
          {{ selectedIndex + 1 }} / {{ steps.length }}
        </span>
        <Button variant="ghost" size="sm" :disabled="selectedIndex >= steps.length - 1" @click="selectedIndex++">
          <ChevronRight class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <div class="flex" style="min-height: 200px; max-height: 500px;">
      <!-- Left panel: Step list -->
      <div class="w-48 shrink-0 border-r overflow-y-auto">
        <button
          v-for="(step, i) in steps"
          :key="step.id || i"
          :class="[
            'w-full text-left px-3 py-2 text-xs border-b hover:bg-muted/50 transition-colors flex items-center gap-2',
            i === selectedIndex ? 'bg-muted' : ''
          ]"
          @click="selectedIndex = i"
        >
          <span class="font-mono text-muted-foreground w-5 shrink-0">{{ step.stepNumber }}</span>
          <span class="truncate flex-1">{{ step.toolName }}</span>
          <span v-if="step.credits" class="text-[10px] font-mono text-amber-500 shrink-0">{{ step.credits }}</span>
          <component :is="stepIcon(step)" class="h-3 w-3 shrink-0" :class="stepIconClass(step)" />
        </button>
      </div>

      <!-- Right panel: Step detail -->
      <div class="flex-1 overflow-y-auto p-4 space-y-3" v-if="selectedStep">
        <!-- Screenshot -->
        <div v-if="selectedStep.screenshotPath">
          <img
            :src="screenshotUrl(selectedStep)"
            class="w-full max-w-md rounded border cursor-pointer"
            alt="Step screenshot"
            @click="screenshotDialogOpen = true"
          />
        </div>

        <!-- Tool name + duration -->
        <div class="flex items-center gap-2">
          <Badge variant="outline">{{ selectedStep.toolName }}</Badge>
          <span class="text-xs text-muted-foreground">{{ selectedStep.durationMs }}ms</span>
        </div>
        <!-- Credits -->
        <span v-if="selectedStep.credits" class="text-xs font-mono text-amber-500">{{ selectedStep.credits }} credits</span>

        <!-- Input -->
        <div v-if="selectedStep.input">
          <span class="text-xs text-muted-foreground font-medium">Input:</span>
          <pre class="mt-1 text-xs bg-muted rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap break-words">{{ selectedStep.input }}</pre>
        </div>

        <!-- Result -->
        <div v-if="selectedStep.result">
          <span class="text-xs text-muted-foreground font-medium">Result:</span>
          <pre class="mt-1 text-xs bg-muted rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap break-words">{{ selectedStep.result }}</pre>
        </div>

        <!-- Error -->
        <div v-if="selectedStep.error">
          <span class="text-xs text-destructive font-medium">Error:</span>
          <pre class="mt-1 text-xs bg-destructive/10 text-destructive rounded p-2 max-h-24 overflow-auto whitespace-pre-wrap break-words">{{ selectedStep.error }}</pre>
        </div>

        <!-- Reasoning -->
        <div v-if="selectedStep.reasoning">
          <span class="text-xs text-muted-foreground font-medium">Reasoning:</span>
          <p class="mt-1 text-xs text-muted-foreground leading-relaxed max-h-32 overflow-auto">
            {{ selectedStep.reasoning }}
          </p>
        </div>
      </div>
    </div>

    <!-- Screenshot fullscreen dialog -->
    <Dialog :open="screenshotDialogOpen" @update:open="screenshotDialogOpen = $event">
      <DialogContent class="max-w-4xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>Step {{ selectedStep?.stepNumber }}: {{ selectedStep?.toolName }}</DialogTitle>
          <DialogDescription>{{ selectedStep?.result }}</DialogDescription>
        </DialogHeader>
        <div class="mt-4">
          <img
            v-if="selectedStep?.screenshotPath"
            :src="screenshotUrl(selectedStep)"
            class="w-full rounded-md border"
            alt="Agent step screenshot"
          />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { analysesApi } from '@/lib/api'
import { ChevronLeft, ChevronRight, CheckCircle, XCircle, Circle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

const props = defineProps({
  analysisId: { type: String, required: true },
  initialSteps: { type: Array, default: null },
  screenshotBaseUrl: { type: String, default: '' },
})

const steps = ref([])
const selectedIndex = ref(0)
const screenshotDialogOpen = ref(false)

const selectedStep = computed(() => steps.value[selectedIndex.value] || null)

function screenshotUrl(step) {
  if (!step?.screenshotPath) return ''
  if (props.screenshotBaseUrl) {
    return props.screenshotBaseUrl + encodeURIComponent(step.screenshotPath)
  }
  if (!props.analysisId) return ''
  return analysesApi.screenshotUrl(props.analysisId, step.screenshotPath)
}

function stepIcon(step) {
  if (step.error) return XCircle
  if (step.result) return CheckCircle
  return Circle
}

function stepIconClass(step) {
  if (step.error) return 'text-destructive'
  if (step.result) return 'text-green-500'
  return 'text-muted-foreground'
}

async function loadSteps() {
  if (props.initialSteps) {
    steps.value = props.initialSteps
    return
  }
  if (!props.analysisId) return
  try {
    const data = await analysesApi.steps(props.analysisId)
    steps.value = data.steps || []
  } catch {
    // ignore
  }
}

watch(() => props.initialSteps, (val) => {
  if (val) {
    steps.value = val
    if (selectedIndex.value >= val.length) {
      selectedIndex.value = Math.max(0, val.length - 1)
    }
  }
})

watch(() => props.analysisId, () => {
  if (!props.initialSteps) {
    loadSteps()
  }
})

onMounted(loadSteps)
</script>
