<script setup>
import { watch, ref, nextTick } from 'vue'
import { useTestExecution } from '@/composables/useTestExecution'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import StatusBadge from '@/components/StatusBadge.vue'

const props = defineProps({
  testId: { type: String, required: true },
})

const { status, logs, progress, result, startExecution } = useTestExecution()
const logContainer = ref(null)

startExecution(props.testId)

watch(logs, async () => {
  await nextTick()
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
})
</script>

<template>
  <div class="space-y-4">
    <!-- Status Header -->
    <div class="flex items-center justify-between">
      <h4 class="text-sm font-medium">Execution Status</h4>
      <Badge
        :class="{
          'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/20': status === 'running',
          'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border-emerald-500/20': status === 'completed',
          'bg-red-500/15 text-red-700 dark:text-red-400 border-red-500/20': status === 'failed',
          'bg-gray-500/15 text-gray-700 dark:text-gray-400 border-gray-500/20': status === 'idle',
        }"
        variant="outline"
      >
        {{ status === 'running' ? 'Running...' : status.charAt(0).toUpperCase() + status.slice(1) }}
      </Badge>
    </div>

    <!-- Progress Bar -->
    <div v-if="status === 'running'" class="w-full bg-muted rounded-full h-2 overflow-hidden">
      <div class="bg-primary h-2 rounded-full animate-pulse" style="width: 100%"></div>
    </div>

    <!-- Flow Progress -->
    <div v-if="progress.length" class="space-y-2">
      <h4 class="text-sm font-medium">Flow Results</h4>
      <div
        v-for="flow in progress"
        :key="flow.flowName"
        class="flex items-center justify-between rounded-md border p-3"
      >
        <span class="text-sm font-medium">{{ flow.flowName }}</span>
        <StatusBadge :status="flow.status" />
      </div>
    </div>

    <!-- Log Output -->
    <div v-if="logs.length">
      <h4 class="text-sm font-medium mb-2">Output</h4>
      <div
        ref="logContainer"
        class="bg-muted rounded-md p-3 font-mono text-xs overflow-auto max-h-64 space-y-0.5"
      >
        <div v-for="(line, i) in logs" :key="i" class="whitespace-pre-wrap">{{ line }}</div>
      </div>
    </div>

    <Separator v-if="result" />

    <!-- Result Summary -->
    <Card v-if="result">
      <CardHeader class="pb-2">
        <CardTitle class="text-sm">Summary</CardTitle>
      </CardHeader>
      <CardContent class="space-y-2 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">Status</span>
          <StatusBadge :status="result.status" />
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Duration</span>
          <span class="font-medium">{{ result.duration }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Success Rate</span>
          <span class="font-medium">{{ result.successRate?.toFixed(1) }}%</span>
        </div>
        <div v-if="result.flowCount" class="flex justify-between">
          <span class="text-muted-foreground">Flows</span>
          <span class="font-medium">{{ result.flowCount }}</span>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
