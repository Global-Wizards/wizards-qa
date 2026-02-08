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
        <Button :disabled="!isValidUrl(gameUrl) || analyzing" @click="handleAnalyze">
          {{ analyzing ? 'Starting...' : 'Analyze Game' }}
        </Button>
      </CardContent>
    </Card>

    <!-- Recent Analyses (idle state) -->
    <Card v-if="status === 'idle' && recentAnalyses.length" class="mt-4">
      <CardHeader>
        <CardTitle class="text-lg">Recent Analyses</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="space-y-2">
          <div
            v-for="item in recentAnalyses"
            :key="item.id"
            class="flex items-center justify-between p-3 rounded-md border hover:bg-muted/50 transition-colors"
          >
            <div class="min-w-0 cursor-pointer flex-1" @click="viewAnalysis(item)">
              <p class="text-sm font-medium truncate">{{ item.gameName || item.gameUrl }}</p>
              <p class="text-xs text-muted-foreground">
                {{ item.framework }} &middot; {{ item.flowCount }} flow(s) &middot; {{ formatDate(item.createdAt) }}
              </p>
            </div>
            <div class="flex items-center gap-2 shrink-0 ml-2">
              <Badge variant="secondary">{{ item.status }}</Badge>
              <Button variant="ghost" size="sm" @click="reAnalyze(item)">
                <RefreshCw class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="sm" @click="deleteAnalysis(item)">
                <Trash2 class="h-3 w-3 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- State 2: Progress -->
    <div v-else-if="status === 'scouting' || status === 'analyzing' || status === 'generating'" class="space-y-4">
      <Card>
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle>Analyzing: {{ gameUrl }}</CardTitle>
            <span v-if="elapsedSeconds > 0" class="text-sm text-muted-foreground">
              {{ formatElapsed(elapsedSeconds) }}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <div class="space-y-1">
            <ProgressStep
              :status="granularStepStatus('scouting')"
              label="Scouting page"
              :detail="stepDuration('scouting') ? `Completed in ${stepDuration('scouting')}s` : 'Fetching page and extracting metadata...'"
              :sub-details="scoutingDetails"
            />
            <ProgressStep
              :status="granularStepStatus('analyzing')"
              label="Analyzing game mechanics"
              :detail="analyzingDetail"
              :sub-details="analysisDetails"
            />
            <ProgressStep
              :status="granularStepStatus('scenarios')"
              label="Generating test scenarios"
              :detail="scenariosDetail"
            />
            <ProgressStep
              :status="granularStepStatus('flows')"
              label="Generating Maestro test flows"
              :detail="flowsDetail"
            />
          </div>

          <Separator class="my-4" />

          <div ref="logContainer" class="max-h-40 overflow-y-auto rounded-md bg-muted p-3">
            <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">
              {{ line }}
            </p>
            <p v-if="!logs.length" class="text-xs text-muted-foreground">Waiting for output...</p>
          </div>

          <div class="flex justify-end mt-4">
            <Button variant="outline" @click="handleReset">Cancel</Button>
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
              <div v-if="analysis.edgeCases?.length">
                <span class="text-muted-foreground">Edge Cases ({{ analysis.edgeCases.length }}):</span>
                <ul class="ml-4 list-disc">
                  <li v-for="ec in analysis.edgeCases" :key="ec.name">{{ ec.name }}: {{ ec.description }}</li>
                </ul>
              </div>
            </div>
          </details>

          <!-- Generated Flows -->
          <details v-if="flowList.length" class="group">
            <summary class="cursor-pointer text-sm font-medium">Generated Flows ({{ flowList.length }})</summary>
            <div class="mt-2 flex flex-wrap gap-2">
              <Badge
                v-for="flow in flowList"
                :key="flow.name"
                variant="outline"
                class="cursor-pointer hover:bg-accent"
                @click="previewFlow(flow)"
              >
                {{ flow.name }}
              </Badge>
            </div>
          </details>

          <!-- Debug Info -->
          <details class="group">
            <summary class="cursor-pointer text-sm font-medium">Debug Info</summary>
            <div class="mt-2 space-y-3 text-sm">
              <!-- Screenshot -->
              <div v-if="pageMeta?.screenshotB64">
                <span class="text-muted-foreground font-medium">Screenshot:</span>
                <img
                  :src="'data:image/jpeg;base64,' + pageMeta.screenshotB64"
                  class="mt-1 rounded-md border max-w-md"
                  alt="Game screenshot"
                />
              </div>

              <!-- JS Globals -->
              <div v-if="pageMeta?.jsGlobals?.length">
                <span class="text-muted-foreground font-medium">JS Globals:</span>
                <div class="mt-1 flex flex-wrap gap-1">
                  <Badge v-for="g in pageMeta.jsGlobals" :key="g" variant="secondary" class="text-xs">{{ g }}</Badge>
                </div>
              </div>

              <!-- URL Hints -->
              <div v-if="gameUrl">
                <span class="text-muted-foreground font-medium">URL Hints:</span>
                <div class="mt-1 space-y-0.5">
                  <div v-for="(value, key) in parseUrlHints(gameUrl)" :key="key" class="text-xs font-mono">
                    <span class="text-muted-foreground">{{ key }}:</span> {{ value }}
                  </div>
                </div>
              </div>

              <!-- Step Timings -->
              <div v-if="formatStepTimingSummary()">
                <span class="text-muted-foreground font-medium">Step Timings:</span>
                <p class="text-xs font-mono mt-1">{{ formatStepTimingSummary() }}</p>
              </div>

              <!-- Body Snippet -->
              <div v-if="pageMeta?.bodySnippet">
                <span class="text-muted-foreground font-medium">Body Snippet:</span>
                <pre class="mt-1 max-h-32 overflow-auto rounded-md bg-muted p-2 text-xs">{{ pageMeta.bodySnippet.slice(0, 500) }}</pre>
              </div>

              <!-- Raw AI Response (shown when JSON parsing failed) -->
              <div v-if="analysis?.rawResponse">
                <span class="text-muted-foreground font-medium">Raw AI Response:</span>
                <pre class="mt-1 max-h-48 overflow-auto rounded-md bg-muted p-2 text-xs">{{ analysis.rawResponse }}</pre>
              </div>

              <!-- Script Sources (full list) -->
              <div v-if="pageMeta?.scriptSrcs?.length">
                <span class="text-muted-foreground font-medium">Script Sources ({{ pageMeta.scriptSrcs.length }}):</span>
                <ul class="mt-1 ml-4 list-disc text-xs text-muted-foreground">
                  <li v-for="src in pageMeta.scriptSrcs" :key="src">{{ src }}</li>
                </ul>
              </div>
            </div>
          </details>

          <Separator />

          <!-- Actions -->
          <div class="flex flex-wrap gap-2">
            <Button @click="navigateToNewPlan">Create Test Plan</Button>
            <Button variant="secondary" @click="runFlowsNow">Run Flows Now</Button>
            <Button variant="outline" @click="navigateToFlows">View Flows</Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="outline">
                  <Download class="h-4 w-4 mr-1" />
                  Export
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem @click="exportAnalysis('json')">Export as JSON</DropdownMenuItem>
                <DropdownMenuItem @click="exportAnalysis('markdown')">Export as Markdown</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
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

        <!-- Show progress steps so user sees where it failed -->
        <div class="mt-4 space-y-1" v-if="currentStep || Object.keys(stepTimings).length">
          <ProgressStep
            :status="granularStepStatus('scouting')"
            label="Scouting page"
            :detail="stepDuration('scouting') ? `${stepDuration('scouting')}s` : ''"
            :sub-details="scoutingDetails"
          />
          <ProgressStep
            :status="granularStepStatus('analyzing')"
            label="Analyzing game mechanics"
            :detail="stepDuration('analyzing') ? `${stepDuration('analyzing')}s` : ''"
            :sub-details="analysisDetails"
          />
          <ProgressStep
            :status="granularStepStatus('scenarios')"
            label="Generating test scenarios"
            :detail="stepDuration('scenarios') ? `${stepDuration('scenarios')}s` : ''"
          />
          <ProgressStep
            :status="granularStepStatus('flows')"
            label="Generating Maestro test flows"
            :detail="stepDuration('flows') ? `${stepDuration('flows')}s` : ''"
          />
        </div>

        <!-- Show collected logs -->
        <div v-if="logs.length" class="mt-4 max-h-40 overflow-y-auto rounded-md bg-muted p-3">
          <p v-for="(line, i) in logs" :key="i" class="text-xs font-mono text-muted-foreground">{{ line }}</p>
        </div>

        <!-- Elapsed time at failure -->
        <p v-if="elapsedSeconds > 0" class="mt-2 text-xs text-muted-foreground">
          Failed after {{ formatElapsed(elapsedSeconds) }}
        </p>

        <div class="mt-4 flex gap-2">
          <Button variant="outline" @click="handleReset">Try Again</Button>
        </div>
      </CardContent>
    </Card>

    <!-- Flow Preview Dialog -->
    <Dialog :open="flowDialogOpen" @update:open="flowDialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>{{ previewFlowData?.name }}</DialogTitle>
          <DialogDescription>Generated flow YAML</DialogDescription>
        </DialogHeader>
        <div class="mt-4 relative">
          <Button
            variant="outline"
            size="sm"
            class="absolute top-2 right-2 z-10"
            @click="copyFlowYaml"
          >
            {{ flowCopied ? 'Copied!' : 'Copy' }}
          </Button>
          <pre class="bg-muted rounded-md p-4 text-sm overflow-auto max-h-[60vh]"><code>{{ previewFlowYaml }}</code></pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAnalysis } from '@/composables/useAnalysis'
import { testsApi, analysesApi, projectsApi } from '@/lib/api'
import { useProject } from '@/composables/useProject'
import { RefreshCw, Trash2, Download } from 'lucide-vue-next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'
import ProgressStep from '@/components/ProgressStep.vue'

const router = useRouter()
const route = useRoute()
const { currentProject } = useProject()
const projectId = computed(() => route.params.projectId || '')
const gameUrl = ref('')
const analyzing = ref(false)
const recentAnalyses = ref([])
const logContainer = ref(null)
const currentAnalysisId = ref(null)

// Flow preview state
const flowDialogOpen = ref(false)
const previewFlowData = ref(null)
const previewFlowYaml = ref('')
const flowCopied = ref(false)

const {
  status,
  currentStep,
  analysisId,
  pageMeta,
  analysis,
  flows: flowList,
  error: analysisError,
  logs,
  elapsedSeconds,
  stepTimings,
  formatElapsed,
  start,
  reset,
  tryRecover,
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

// --- Rich progress detail computeds ---

const scoutingDetails = computed(() => {
  if (!pageMeta.value) return []
  const details = []
  if (pageMeta.value.title) details.push({ label: 'Title', value: pageMeta.value.title })
  details.push({ label: 'Framework', value: pageMeta.value.framework || 'unknown' })
  details.push({ label: 'Canvas', value: pageMeta.value.canvasFound ? 'Detected' : 'Not found' })
  details.push({ label: 'Scripts', value: `${pageMeta.value.scriptSrcs?.length || 0} found` })
  if (pageMeta.value.jsGlobals?.length) {
    details.push({ label: 'JS Globals', value: pageMeta.value.jsGlobals.join(', ') })
  }
  if (pageMeta.value.screenshotB64) {
    const sizeKB = Math.round(pageMeta.value.screenshotB64.length * 3 / 4 / 1024)
    details.push({ label: 'Screenshot', value: `Captured (${sizeKB} KB)` })
  }
  return details
})

const analyzingDetail = computed(() => {
  const dur = stepDuration('analyzing')
  if (analysis.value) {
    const mode = pageMeta.value?.screenshotB64 ? 'multimodal' : 'text-only'
    return `${mode} analysis${dur ? ` in ${dur}s` : ''}`
  }
  // Show the latest log message for the analyzing step
  const lastAnalyzingLog = logs.value.filter(l => l.includes('AI') || l.includes('multimodal') || l.includes('Sending')).pop()
  return lastAnalyzingLog || 'Waiting for AI response...'
})

const analysisDetails = computed(() => {
  if (!analysis.value) return []
  const details = []
  if (analysis.value.gameInfo?.name) {
    details.push({ label: 'Game', value: `${analysis.value.gameInfo.name}${analysis.value.gameInfo.genre ? ' (' + analysis.value.gameInfo.genre + ')' : ''}` })
  }
  if (analysis.value.gameInfo?.technology) {
    details.push({ label: 'Technology', value: analysis.value.gameInfo.technology })
  }
  if (analysis.value.mechanics?.length) {
    details.push({ label: 'Mechanics', value: `${analysis.value.mechanics.length} found` })
  }
  if (analysis.value.uiElements?.length) {
    details.push({ label: 'UI Elements', value: `${analysis.value.uiElements.length} found` })
  }
  if (analysis.value.userFlows?.length) {
    details.push({ label: 'User Flows', value: `${analysis.value.userFlows.length} identified` })
  }
  if (analysis.value.edgeCases?.length) {
    details.push({ label: 'Edge Cases', value: `${analysis.value.edgeCases.length} identified` })
  }
  return details
})

const scenariosDetail = computed(() => {
  const dur = stepDuration('scenarios')
  if (stepOrder(currentStep.value) > stepOrder('scenarios_done')) {
    return `Scenarios generated${dur ? ` in ${dur}s` : ''}`
  }
  if (currentStep.value === 'scenarios_done') {
    return `Scenarios generated${dur ? ` in ${dur}s` : ''}`
  }
  return dur ? `Working... (${dur}s)` : ''
})

const flowsDetail = computed(() => {
  const dur = stepDuration('flows')
  if (flowList.value.length) {
    return `${flowList.value.length} flow(s) generated${dur ? ` in ${dur}s` : ''}`
  }
  return dur ? `Working... (${dur}s)` : ''
})

function isValidUrl(str) {
  try {
    const url = new URL(str)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

// Ordered step names for granular progress
const STEP_ORDER = ['scouting', 'scouted', 'analyzing', 'analyzed', 'scenarios', 'scenarios_done', 'flows', 'flows_done', 'saving', 'complete']

function stepOrder(step) {
  const idx = STEP_ORDER.indexOf(step)
  return idx >= 0 ? idx : -1
}

/**
 * Determine the status of a progress step group based on the current granular step.
 * Each ProgressStep represents a group of granular steps:
 *   scouting  → scouting, scouted
 *   analyzing → analyzing, analyzed
 *   scenarios → scenarios, scenarios_done
 *   flows     → flows, flows_done, saving
 */
function granularStepStatus(groupStart) {
  const groupMap = {
    scouting: { start: 'scouting', end: 'scouted' },
    analyzing: { start: 'analyzing', end: 'analyzed' },
    scenarios: { start: 'scenarios', end: 'scenarios_done' },
    flows: { start: 'flows', end: 'complete' },
  }
  const group = groupMap[groupStart]
  if (!group) return 'pending'

  const current = stepOrder(currentStep.value)
  const groupStartIdx = stepOrder(group.start)
  const groupEndIdx = stepOrder(group.end)

  if (current < 0 || current < groupStartIdx) return 'pending'
  if (current > groupEndIdx) return 'complete'
  return 'active'
}

function stepDuration(stepName) {
  const timing = stepTimings.value[stepName]
  if (!timing || !timing.start) return null
  const end = timing.end || Date.now()
  return ((end - timing.start) / 1000).toFixed(1)
}

function formatStepTimingSummary() {
  const labels = { scouting: 'Scouting', analyzing: 'Analyzing', scenarios: 'Scenarios', flows: 'Flows' }
  return Object.entries(labels)
    .map(([key, label]) => {
      const d = stepDuration(key)
      return d ? `${label}: ${d}s` : null
    })
    .filter(Boolean)
    .join(' | ')
}

function parseUrlHints(urlStr) {
  try {
    const url = new URL(urlStr)
    const hints = {}
    hints.domain = url.hostname
    const interestingParams = ['game_type', 'mode', 'game_id', 'operator_id', 'gameType', 'gameid']
    for (const [key, value] of url.searchParams) {
      if (interestingParams.includes(key) || value) {
        hints[key] = value
      }
    }
    return hints
  } catch {
    return {}
  }
}

async function handleAnalyze() {
  if (!isValidUrl(gameUrl.value)) return
  analyzing.value = true
  try {
    await start(gameUrl.value, projectId.value)
  } catch {
    analyzing.value = false
  }
}

// Track the analysis ID for export
watch(analysisId, (val) => {
  if (val) currentAnalysisId.value = val
})

// Reset analyzing flag when status changes from idle
watch(status, (val) => {
  if (val !== 'idle') {
    analyzing.value = false
  }
})

function handleReset() {
  reset()
  analyzing.value = false
  gameUrl.value = ''
  currentAnalysisId.value = null
  loadRecentAnalyses()
}

function navigateToNewPlan() {
  const flowNames = flowList.value.map((f) => f.name).join(',')
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push({ path: `${basePath}/tests/new`, query: { flows: flowNames, gameUrl: gameUrl.value } })
}

function navigateToFlows() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/flows`)
}

async function runFlowsNow() {
  try {
    await testsApi.run({ gameUrl: gameUrl.value })
    const basePath = projectId.value ? `/projects/${projectId.value}` : ''
    router.push(`${basePath}/tests`)
  } catch (err) {
    console.error('Failed to run flows:', err)
  }
}

function reAnalyze(item) {
  gameUrl.value = item.gameUrl
  handleAnalyze()
}

async function deleteAnalysis(item) {
  try {
    await analysesApi.delete(item.id)
    recentAnalyses.value = recentAnalyses.value.filter((a) => a.id !== item.id)
  } catch (err) {
    console.error('Failed to delete analysis:', err)
  }
}

function exportAnalysis(format) {
  if (!currentAnalysisId.value) return
  analysesApi.export(currentAnalysisId.value, format).catch((err) => {
    console.error('Failed to export analysis:', err)
  })
}

function previewFlow(flow) {
  previewFlowData.value = flow
  flowCopied.value = false

  // Build a simple YAML representation from the flow object
  let yaml = ''
  if (flow.url) yaml += `url: ${flow.url}\n`
  if (flow.appId) yaml += `appId: ${flow.appId}\n`
  if (flow.tags?.length) {
    yaml += 'tags:\n'
    flow.tags.forEach((t) => { yaml += `  - ${t}\n` })
  }
  yaml += '---\n'
  if (flow.commands?.length) {
    flow.commands.forEach((cmd) => {
      for (const [key, val] of Object.entries(cmd)) {
        if (key === 'comment') {
          yaml += `# ${val}\n`
        } else if (typeof val === 'object' && val !== null) {
          yaml += `- ${key}:\n`
          for (const [sk, sv] of Object.entries(val)) {
            yaml += `    ${sk}: ${sv}\n`
          }
        } else {
          yaml += `- ${key}: ${val}\n`
        }
      }
    })
  }
  previewFlowYaml.value = yaml || 'No YAML content available'
  flowDialogOpen.value = true
}

async function copyFlowYaml() {
  try {
    await navigator.clipboard.writeText(previewFlowYaml.value)
    flowCopied.value = true
    setTimeout(() => { flowCopied.value = false }, 2000)
  } catch {
    // clipboard API not available
  }
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateStr
  }
}

async function viewAnalysis(item) {
  try {
    const data = await analysesApi.get(item.id)
    if (data.result) {
      pageMeta.value = data.result.pageMeta || null
      analysis.value = data.result.analysis || null
      flowList.value = data.result.flows || []
      gameUrl.value = data.gameUrl || ''
      currentAnalysisId.value = data.id
      status.value = 'complete'
    }
  } catch (err) {
    console.error('Failed to load analysis:', err)
  }
}

async function loadRecentAnalyses() {
  try {
    const data = projectId.value
      ? await projectsApi.analyses(projectId.value)
      : await analysesApi.list()
    const all = data.analyses || []
    recentAnalyses.value = all.slice(-5).reverse()
  } catch {
    // Silently ignore — analyses endpoint may not have data yet
  }
}

// Auto-scroll log area when new logs arrive
watch(logs, () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
})

onMounted(async () => {
  // Pre-fill game URL from project context
  if (currentProject.value?.gameUrl && !gameUrl.value) {
    gameUrl.value = currentProject.value.gameUrl
  }

  // Try to recover a running or completed analysis from localStorage
  const recovery = await tryRecover()
  if (recovery) {
    if (recovery.gameUrl) {
      gameUrl.value = recovery.gameUrl
    }
    if (recovery.status === 'running') {
      logs.value = [...logs.value, 'Reconnected to running analysis...']
    }
    // If recovered as completed, the status/data refs are already set
    return
  }

  loadRecentAnalyses()
})
</script>
