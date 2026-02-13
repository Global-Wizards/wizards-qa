<template>
  <div v-if="steps.length" class="rounded-lg border bg-card">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b">
      <span class="text-sm font-medium">Test Steps ({{ steps.length }})</span>
      <div class="flex items-center gap-1">
        <Button variant="ghost" size="sm" :disabled="selectedIndex <= 0" @click="navigate(-1)">
          <ChevronLeft class="h-4 w-4" />
        </Button>
        <span class="text-xs text-muted-foreground min-w-[4rem] text-center">
          {{ selectedIndex + 1 }} / {{ steps.length }}
        </span>
        <Button variant="ghost" size="sm" :disabled="selectedIndex >= steps.length - 1" @click="navigate(1)">
          <ChevronRight class="h-4 w-4" />
        </Button>
        <Button
          v-if="live && !autoFollow"
          variant="ghost"
          size="sm"
          class="ml-2 text-xs"
          @click="resumeFollow"
        >
          Resume follow
        </Button>
      </div>
    </div>

    <div class="flex" style="min-height: 200px; max-height: 500px;">
      <!-- Left panel: Step list grouped by flow -->
      <div class="w-56 shrink-0 border-r overflow-y-auto" ref="sidebarRef">
        <template v-for="group in groupedSteps" :key="group.flowName">
          <div class="sticky top-0 z-10 bg-muted/80 backdrop-blur-sm px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground border-b">
            {{ group.flowName }}
          </div>
          <button
            v-for="step in group.steps"
            :key="step.globalIndex"
            :class="[
              'w-full text-left px-3 py-2 text-xs border-b hover:bg-muted/50 transition-colors flex items-center gap-2',
              step.globalIndex === selectedIndex ? 'bg-muted' : ''
            ]"
            @click="selectStep(step.globalIndex)"
          >
            <span class="font-mono text-muted-foreground w-5 shrink-0">#{{ step.stepIndex + 1 }}</span>
            <span class="truncate flex-1">{{ step.commandShort }}</span>
            <component :is="stepIcon(step)" class="h-3 w-3 shrink-0" :class="stepIconClass(step)" />
          </button>
        </template>
      </div>

      <!-- Right panel: Step detail -->
      <div class="flex-1 overflow-y-auto p-4 space-y-3" v-if="selectedStep">
        <!-- Screenshot -->
        <div v-if="selectedStep.screenshotUrl || selectedStep.screenshotB64">
          <img
            :src="selectedStep.screenshotUrl || ('data:image/webp;base64,' + selectedStep.screenshotB64)"
            class="w-full max-w-md rounded border cursor-pointer"
            alt="Step screenshot"
            @click="screenshotDialogOpen = true"
          />
        </div>

        <!-- Command + status -->
        <div class="flex items-center gap-2">
          <Badge variant="outline">{{ selectedStep.command }}</Badge>
          <Badge
            :variant="selectedStep.status === 'passed' ? 'default' : 'destructive'"
            class="text-[10px]"
          >
            {{ selectedStep.status }}
          </Badge>
        </div>

        <!-- Result -->
        <div v-if="selectedStep.result">
          <span class="text-xs text-muted-foreground font-medium">Result:</span>
          <pre class="mt-1 text-xs bg-muted rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap break-words">{{ selectedStep.result }}</pre>
        </div>

        <!-- AI Reasoning -->
        <div v-if="selectedStep.reasoning">
          <span class="text-xs text-muted-foreground font-medium">AI Reasoning:</span>
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
          <DialogTitle>Step #{{ (selectedStep?.stepIndex ?? 0) + 1 }}: {{ selectedStep?.command }}</DialogTitle>
          <DialogDescription>{{ selectedStep?.result }}</DialogDescription>
        </DialogHeader>
        <div class="mt-4">
          <img
            v-if="selectedStep?.screenshotUrl || selectedStep?.screenshotB64"
            :src="selectedStep.screenshotUrl || ('data:image/webp;base64,' + selectedStep.screenshotB64)"
            class="w-full rounded-md border"
            alt="Test step screenshot"
          />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ChevronLeft, ChevronRight, CheckCircle, XCircle, Circle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

const props = defineProps({
  steps: { type: Array, default: () => [] },
  live: { type: Boolean, default: false },
})

const selectedIndex = ref(0)
const autoFollow = ref(true)
const screenshotDialogOpen = ref(false)
const sidebarRef = ref(null)

const selectedStep = computed(() => props.steps[selectedIndex.value] || null)

const groupedSteps = computed(() => {
  const groups = []
  let currentGroup = null
  props.steps.forEach((step, i) => {
    if (!currentGroup || currentGroup.flowName !== step.flowName) {
      currentGroup = { flowName: step.flowName, steps: [] }
      groups.push(currentGroup)
    }
    currentGroup.steps.push({
      ...step,
      globalIndex: i,
      commandShort: (step.command || '').length > 30 ? step.command.substring(0, 30) + '...' : (step.command || ''),
    })
  })
  return groups
})

function stepIcon(step) {
  if (step.status === 'failed') return XCircle
  if (step.status === 'passed') return CheckCircle
  return Circle
}

function stepIconClass(step) {
  if (step.status === 'failed') return 'text-destructive'
  if (step.status === 'passed') return 'text-green-500'
  return 'text-muted-foreground'
}

function selectStep(index) {
  selectedIndex.value = index
  autoFollow.value = false
}

function navigate(delta) {
  const newIndex = selectedIndex.value + delta
  if (newIndex >= 0 && newIndex < props.steps.length) {
    selectedIndex.value = newIndex
    autoFollow.value = false
  }
}

function resumeFollow() {
  autoFollow.value = true
  if (props.steps.length > 0) {
    selectedIndex.value = props.steps.length - 1
  }
}

// Auto-follow latest step when live
watch(() => props.steps.length, (len) => {
  if (props.live && autoFollow.value && len > 0) {
    selectedIndex.value = len - 1
  }
})
</script>
