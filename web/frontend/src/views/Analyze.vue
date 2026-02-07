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
          <CardTitle>Analyzing: {{ gameUrl }}</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="space-y-1">
            <ProgressStep
              :status="stepStatus('scouting')"
              label="Scouting page..."
              :detail="pageMeta ? `${pageMeta.framework} | Canvas: ${pageMeta.canvasFound} | ${pageMeta.title || 'No title'}` : ''"
            />
            <ProgressStep
              :status="stepStatus('analyzing')"
              label="Analyzing game mechanics..."
              :detail="analysis ? `${analysis.mechanics?.length || 0} mechanics, ${analysis.uiElements?.length || 0} UI elements` : ''"
            />
            <ProgressStep
              :status="stepStatus('generating')"
              label="Generating test flows..."
              :detail="flowList.length ? `${flowList.length} flow(s) generated` : ''"
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
        <div class="mt-4">
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
import { useRouter } from 'vue-router'
import { useAnalysis } from '@/composables/useAnalysis'
import { testsApi, analysesApi } from '@/lib/api'
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
  analysisId,
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

async function handleAnalyze() {
  if (!isValidUrl(gameUrl.value)) return
  analyzing.value = true
  try {
    await start(gameUrl.value)
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
  router.push({ path: '/tests/new', query: { flows: flowNames, gameUrl: gameUrl.value } })
}

function navigateToFlows() {
  router.push('/flows')
}

async function runFlowsNow() {
  try {
    await testsApi.run({ gameUrl: gameUrl.value })
    router.push('/tests')
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
  window.open(analysesApi.exportUrl(currentAnalysisId.value, format), '_blank')
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
    const data = await analysesApi.list()
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

onMounted(() => {
  loadRecentAnalyses()
})
</script>
