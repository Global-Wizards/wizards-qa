<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="space-y-4">
      <Skeleton class="h-8 w-48" />
      <Skeleton class="h-10 w-full" />
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <Skeleton v-for="i in 4" :key="i" class="h-24" />
      </div>
    </div>

    <!-- Error -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Failed to load analysis</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Content -->
    <template v-else-if="analysisData">
      <!-- Header -->
      <div class="flex items-center gap-3 mb-6">
        <Button variant="ghost" size="sm" @click="goBack">
          <ArrowLeft class="h-4 w-4 mr-1" />
          Back
        </Button>
        <div class="flex-1 min-w-0">
          <h2 class="text-2xl font-bold tracking-tight truncate">{{ gameName }}</h2>
          <div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground mt-0.5">
            <span class="truncate max-w-xs" :title="analysisData.gameUrl">{{ truncateUrl(analysisData.gameUrl, 50) }}</span>
            <span v-if="framework">&middot;</span>
            <span v-if="framework" class="capitalize">{{ framework }}</span>
            <span>&middot;</span>
            <span>{{ formatDate(analysisData.createdAt) }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <Button v-if="hasTestPlan" variant="default" size="sm" @click="runTests" :disabled="runningTests">
            <FlaskConical class="h-4 w-4 mr-1" />
            {{ runningTests ? 'Running...' : 'Run Tests' }}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="outline" size="sm">
                <Download class="h-4 w-4 mr-1" />
                Export
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem @click="exportAnalysis('json')">Export as JSON</DropdownMenuItem>
              <DropdownMenuItem @click="exportAnalysis('markdown')">Export as Markdown</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <!-- Tabs -->
      <Tabs v-model="activeTab">
        <TabsList class="flex flex-wrap h-auto gap-1">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="functional">Functional QA</TabsTrigger>
          <TabsTrigger v-if="showUiuxTab" value="uiux" :disabled="!uiuxCount">
            UI/UX Analysis
            <Badge v-if="uiuxCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ uiuxCount }}</Badge>
          </TabsTrigger>
          <TabsTrigger v-if="showWordingTab" value="wording" :disabled="!wordingCount">
            Wording
            <Badge v-if="wordingCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ wordingCount }}</Badge>
          </TabsTrigger>
          <TabsTrigger v-if="showGameDesignTab" value="gamedesign" :disabled="!gameDesignCount">
            Game Design
            <Badge v-if="gameDesignCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ gameDesignCount }}</Badge>
          </TabsTrigger>
          <TabsTrigger v-if="showFlowsTab" value="flows" :disabled="!flowCount">
            Test Flows
            <Badge v-if="flowCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ flowCount }}</Badge>
          </TabsTrigger>
          <TabsTrigger v-if="isAgentMode" value="exploration">
            Exploration
          </TabsTrigger>
          <TabsTrigger v-if="lastTestRunId" value="test-results">
            Test Results
          </TabsTrigger>
        </TabsList>

        <div class="mt-6">
          <TabsContent value="overview">
            <OverviewTab :analysis="analysis" :page-meta="pageMeta" :flows="flows" />
          </TabsContent>
          <TabsContent value="functional">
            <FunctionalQATab :analysis="analysis" />
          </TabsContent>
          <TabsContent v-if="showUiuxTab" value="uiux">
            <FindingsTab :findings="analysis?.uiuxAnalysis" type="uiux" />
          </TabsContent>
          <TabsContent v-if="showWordingTab" value="wording">
            <FindingsTab :findings="analysis?.wordingCheck" type="wording" />
          </TabsContent>
          <TabsContent v-if="showGameDesignTab" value="gamedesign">
            <FindingsTab :findings="analysis?.gameDesign" type="gamedesign" />
          </TabsContent>
          <TabsContent v-if="showFlowsTab" value="flows">
            <TestFlowsTab :flows="flows" :game-url="analysisData.gameUrl" />
          </TabsContent>
          <TabsContent v-if="isAgentMode" value="exploration">
            <AgentStepNavigator :analysis-id="route.params.id" :initial-steps="agentSteps" />
          </TabsContent>
          <TabsContent v-if="lastTestRunId" value="test-results">
            <div v-if="testResultLoading" class="flex items-center gap-2 text-muted-foreground py-8 justify-center">
              Loading test results...
            </div>
            <div v-else-if="testResult" class="space-y-4">
              <div class="flex items-center gap-3">
                <Badge :variant="testResult.status === 'passed' ? 'default' : 'destructive'">
                  {{ testResult.status }}
                </Badge>
                <span class="text-sm text-muted-foreground">
                  {{ testResult.successRate != null ? `${Math.round(testResult.successRate * 100)}% pass rate` : '' }}
                  {{ testResult.duration ? `· ${testResult.duration}` : '' }}
                </span>
              </div>
              <div v-if="testResult.flows?.length" class="space-y-2">
                <div
                  v-for="flow in testResult.flows"
                  :key="flow.name"
                  class="flex items-center justify-between p-3 rounded-md border"
                >
                  <span class="text-sm font-medium">{{ flow.name }}</span>
                  <div class="flex items-center gap-2">
                    <span v-if="flow.duration" class="text-xs text-muted-foreground">{{ flow.duration }}</span>
                    <Badge :variant="flow.status === 'passed' ? 'default' : 'destructive'" class="text-xs">
                      {{ flow.status }}
                    </Badge>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="text-sm text-muted-foreground py-8 text-center">
              Test result not found
            </div>
          </TabsContent>
        </div>
      </Tabs>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { analysesApi, testPlansApi, testsApi } from '@/lib/api'
import { truncateUrl } from '@/lib/utils'
import { formatDate } from '@/lib/dateUtils'
import { ArrowLeft, Download, FlaskConical } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'
import OverviewTab from '@/components/analysis/OverviewTab.vue'
import FunctionalQATab from '@/components/analysis/FunctionalQATab.vue'
import FindingsTab from '@/components/analysis/FindingsTab.vue'
import TestFlowsTab from '@/components/analysis/TestFlowsTab.vue'
import AgentStepNavigator from '@/components/AgentStepNavigator.vue'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref('')
const analysisData = ref(null)
const agentSteps = ref([])
const activeTab = ref('overview')

const runningTests = ref(false)
const testResult = ref(null)
const testResultLoading = ref(false)

const analysis = computed(() => analysisData.value?.result?.analysis || null)
const pageMeta = computed(() => analysisData.value?.result?.pageMeta || null)
const flows = computed(() => analysisData.value?.result?.flows || [])
const isAgentMode = computed(() => analysisData.value?.result?.mode === 'agent')
const framework = computed(() => pageMeta.value?.framework || '')
const lastTestRunId = computed(() => analysisData.value?.lastTestRunId || '')
const hasTestPlan = computed(() => !!analysisData.value?.testPlanId)

const gameName = computed(() => {
  return analysis.value?.gameInfo?.name || pageMeta.value?.title || 'Analysis'
})

const enabledModules = computed(() => {
  try { return JSON.parse(analysisData.value?.modules || '{}') } catch { return {} }
})
const showUiuxTab = computed(() => enabledModules.value.uiux !== false)
const showWordingTab = computed(() => enabledModules.value.wording !== false)
const showGameDesignTab = computed(() => enabledModules.value.gameDesign !== false)
const showFlowsTab = computed(() => enabledModules.value.testFlows !== false)

const uiuxCount = computed(() => analysis.value?.uiuxAnalysis?.length || 0)
const wordingCount = computed(() => analysis.value?.wordingCheck?.length || 0)
const gameDesignCount = computed(() => analysis.value?.gameDesign?.length || 0)
const flowCount = computed(() => flows.value?.length || 0)

function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    const basePath = route.params.projectId ? `/projects/${route.params.projectId}` : ''
    router.push(`${basePath}/analyses`)
  }
}

function exportAnalysis(format) {
  analysesApi.export(route.params.id, format).catch((err) => {
    console.error('Failed to export analysis:', err)
  })
}

async function runTests() {
  const planId = analysisData.value?.testPlanId
  if (!planId) return
  runningTests.value = true
  try {
    const data = await testPlansApi.run(planId, { mode: 'browser' })
    const basePath = route.params.projectId ? `/projects/${route.params.projectId}` : ''
    router.push({ path: `${basePath}/tests/run/${data.testId}`, query: { fresh: '1' } })
  } catch (err) {
    console.error('Failed to run tests:', err)
  } finally {
    runningTests.value = false
  }
}

async function loadTestResult(testRunId) {
  if (!testRunId) return
  testResultLoading.value = true
  try {
    testResult.value = await testsApi.get(testRunId)
  } catch {
    testResult.value = null
  } finally {
    testResultLoading.value = false
  }
}

onMounted(async () => {
  try {
    const data = await analysesApi.get(route.params.id)

    // Running analysis → redirect to live progress view
    if (data.status === 'running') {
      const basePath = route.params.projectId
        ? `/projects/${route.params.projectId}` : ''
      router.replace({
        path: `${basePath}/analyze`,
        query: { analysisId: data.id }
      })
      return
    }

    analysisData.value = data

    if (data.lastTestRunId) {
      loadTestResult(data.lastTestRunId)
    }

    if (data.result?.mode === 'agent') {
      try {
        const stepsData = await analysesApi.steps(route.params.id)
        agentSteps.value = stepsData.steps || []
      } catch {
        // Steps may not be available
      }
    }
  } catch (err) {
    error.value = err.message || 'Failed to load analysis'
  } finally {
    loading.value = false
  }
})
</script>
