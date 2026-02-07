<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Analyze Game</h2>
        <p class="text-muted-foreground">
          Enter a game URL and our AI will automatically detect the framework, analyze game mechanics, and generate test flows.
        </p>
      </div>
    </div>

    <!-- State 1: Input -->
    <Card v-if="status === 'idle'">
      <CardHeader>
        <CardTitle>Game URL</CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="space-y-2">
          <Input
            v-model="gameUrl"
            placeholder="https://your-game.example.com"
            @keyup.enter="handleAnalyze"
          />
          <p v-if="gameUrl && !isValidUrl(gameUrl)" class="text-xs text-destructive">
            Enter a valid URL starting with http:// or https://
          </p>
        </div>
        <Button :disabled="!isValidUrl(gameUrl)" @click="handleAnalyze">
          Analyze Game
        </Button>
      </CardContent>
    </Card>

    <!-- State 2: Progress -->
    <div v-else-if="status === 'scouting' || status === 'analyzing' || status === 'generating'" class="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Analyzing: {{ gameUrl }}</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="space-y-1">
            <ProgressStep
              :status="stepStatus('scouting')"
              label="Scouting page..."
              :detail="status !== 'scouting' ? 'Page metadata collected' : ''"
            />
            <ProgressStep
              :status="stepStatus('analyzing')"
              label="Analyzing game mechanics..."
              :detail="status === 'generating' || status === 'complete' ? 'Analysis complete' : ''"
            />
            <ProgressStep
              :status="stepStatus('generating')"
              label="Generating test flows..."
            />
          </div>

          <Separator class="my-4" />

          <div class="max-h-40 overflow-y-auto rounded-md bg-muted p-3">
            <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">
              {{ line }}
            </p>
            <p v-if="!logs.length" class="text-xs text-muted-foreground">Waiting for output...</p>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- State 3: Results -->
    <div v-else-if="status === 'complete'" class="space-y-4">
      <Card>
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle>Analysis Complete</CardTitle>
            <Badge variant="secondary">{{ flowCount }} flow(s) generated</Badge>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <!-- Summary -->
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <span class="text-sm text-muted-foreground">Game</span>
              <p class="font-medium">{{ gameName }}</p>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Framework</span>
              <p class="font-medium capitalize">{{ framework }}</p>
            </div>
            <div>
              <span class="text-sm text-muted-foreground">Canvas</span>
              <p class="font-medium">{{ pageMeta?.canvasFound ? 'Yes' : 'No' }}</p>
            </div>
          </div>

          <Separator />

          <!-- Page Metadata -->
          <details class="group">
            <summary class="cursor-pointer text-sm font-medium">Page Metadata</summary>
            <div class="mt-2 space-y-2 text-sm">
              <div v-if="pageMeta?.title">
                <span class="text-muted-foreground">Title:</span> {{ pageMeta.title }}
              </div>
              <div v-if="pageMeta?.description">
                <span class="text-muted-foreground">Description:</span> {{ pageMeta.description }}
              </div>
              <div v-if="pageMeta?.scriptSrcs?.length">
                <span class="text-muted-foreground">Scripts ({{ pageMeta.scriptSrcs.length }}):</span>
                <ul class="ml-4 list-disc text-xs text-muted-foreground">
                  <li v-for="src in pageMeta.scriptSrcs.slice(0, 10)" :key="src">{{ src }}</li>
                </ul>
              </div>
            </div>
          </details>

          <!-- Game Analysis -->
          <details v-if="analysis" class="group">
            <summary class="cursor-pointer text-sm font-medium">Game Analysis</summary>
            <div class="mt-2 space-y-2 text-sm">
              <div v-if="analysis.mechanics?.length">
                <span class="text-muted-foreground">Mechanics ({{ analysis.mechanics.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="m in analysis.mechanics" :key="m.name">{{ m.name }}: {{ m.description }}</li>
                </ul>
              </div>
              <div v-if="analysis.uiElements?.length">
                <span class="text-muted-foreground">UI Elements ({{ analysis.uiElements.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="el in analysis.uiElements" :key="el.name">{{ el.name }} ({{ el.type }})</li>
                </ul>
              </div>
              <div v-if="analysis.userFlows?.length">
                <span class="text-muted-foreground">User Flows ({{ analysis.userFlows.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="f in analysis.userFlows" :key="f.name">{{ f.name }}: {{ f.description }}</li>
                </ul>
              </div>
            </div>
          </details>

          <!-- Generated Flows -->
          <details v-if="flowList.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">Generated Flows ({{ flowList.length }})</summary>
            <div class="mt-2 flex flex-wrap gap-2">
              <Badge v-for="flow in flowList" :key="flow.name" variant="outline">
                {{ flow.name }}
              </Badge>
            </div>
          </details>

          <Separator />

          <!-- Actions -->
          <div class="flex flex-wrap gap-2">
            <Button @click="navigateToNewPlan">Create Test Plan</Button>
            <Button variant="outline" @click="navigateToFlows">View Flows</Button>
            <Button variant="outline" @click="handleReset">Analyze Another</Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Error State -->
    <Card v-else-if="status === 'error'">
      <CardContent class="pt-6">
        <Alert variant="destructive">
          <AlertTitle>Analysis Failed</AlertTitle>
          <AlertDescription>{{ analysisError }}</AlertDescription>
        </Alert>
        <div class="mt-4">
          <Button variant="outline" @click="handleReset">Try Again</Button>
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAnalysis } from '@/composables/useAnalysis'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import ProgressStep from '@/components/ProgressStep.vue'

const router = useRouter()
const gameUrl = ref('')

const {
  status,
  pageMeta,
  analysis,
  flows: flowList,
  error: analysisError,
  logs,
  start,
  reset,
} = useAnalysis()

const gameName = computed(() => {
  return analysis.value?.gameInfo?.name || pageMeta.value?.title || 'Unknown Game'
})

const framework = computed(() => {
  return pageMeta.value?.framework || 'unknown'
})

const flowCount = computed(() => {
  return flowList.value?.length || 0
})

function isValidUrl(str) {
  try {
    const url = new URL(str)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function stepStatus(step) {
  const order = ['scouting', 'analyzing', 'generating', 'complete']
  const currentIdx = order.indexOf(status.value)
  const stepIdx = order.indexOf(step)

  if (currentIdx > stepIdx) return 'complete'
  if (currentIdx === stepIdx) return 'active'
  return 'pending'
}

function handleAnalyze() {
  if (!isValidUrl(gameUrl.value)) return
  start(gameUrl.value)
}

function handleReset() {
  reset()
  gameUrl.value = ''
}

function navigateToNewPlan() {
  router.push('/tests/new')
}

function navigateToFlows() {
  router.push('/flows')
}
</script>
