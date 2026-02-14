<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <div class="border-b bg-card">
      <div class="max-w-5xl mx-auto px-6 py-4">
        <div class="flex items-center gap-3">
          <div class="h-8 w-8 rounded-md bg-primary/10 flex items-center justify-center">
            <Gamepad2 class="h-4 w-4 text-primary" />
          </div>
          <div class="flex-1 min-w-0">
            <h1 class="text-xl font-bold tracking-tight truncate">{{ gameName }}</h1>
            <div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground mt-0.5">
              <span class="truncate max-w-xs" :title="analysisData?.gameUrl">{{ truncateUrl(analysisData?.gameUrl || '', 50) }}</span>
              <span v-if="framework">&middot;</span>
              <span v-if="framework" class="capitalize">{{ framework }}</span>
              <span>&middot;</span>
              <span>{{ formatDate(analysisData?.createdAt) }}</span>
            </div>
          </div>
          <span class="text-xs text-muted-foreground">Shared Report</span>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="max-w-5xl mx-auto px-6 py-6">
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
        <AlertTitle>Failed to load report</AlertTitle>
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>

      <!-- Loaded -->
      <template v-else-if="analysisData">
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
            <TabsTrigger v-if="showGliTab" value="gli" :disabled="!gliCount">
              GLI Compliance
              <Badge v-if="gliCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ gliCount }}</Badge>
            </TabsTrigger>
            <TabsTrigger v-if="showFlowsTab" value="flows" :disabled="!flowCount">
              Test Flows
              <Badge v-if="flowCount" variant="secondary" class="ml-1.5 text-xs px-1.5 py-0">{{ flowCount }}</Badge>
            </TabsTrigger>
            <TabsTrigger v-if="isAgentMode" value="exploration">
              Exploration
            </TabsTrigger>
          </TabsList>

          <div class="mt-6">
            <TabsContent value="overview">
              <OverviewTab :analysis="analysis" :page-meta="pageMeta" :flows="flows" :devices="devices" />
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
            <TabsContent v-if="showGliTab" value="gli">
              <FindingsTab :findings="analysis?.gliCompliance" type="gli" />
            </TabsContent>
            <TabsContent v-if="showFlowsTab" value="flows">
              <TestFlowsTab :flows="flows" :game-url="analysisData.gameUrl" />
            </TabsContent>
            <TabsContent v-if="isAgentMode" value="exploration">
              <AgentStepNavigator
                :analysis-id="analysisData.id"
                :initial-steps="agentSteps"
                :screenshot-base-url="`/api/shared/${token}/screenshots/`"
              />
            </TabsContent>
          </div>
        </Tabs>
      </template>
    </div>

    <!-- Footer -->
    <div class="border-t mt-12">
      <div class="max-w-5xl mx-auto px-6 py-4 text-center text-xs text-muted-foreground">
        Powered by Wizards QA
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { sharedApi } from '@/lib/api'
import { truncateUrl } from '@/lib/utils'
import { formatDate } from '@/lib/dateUtils'
import { Gamepad2 } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import OverviewTab from '@/components/analysis/OverviewTab.vue'
import FunctionalQATab from '@/components/analysis/FunctionalQATab.vue'
import FindingsTab from '@/components/analysis/FindingsTab.vue'
import TestFlowsTab from '@/components/analysis/TestFlowsTab.vue'
import AgentStepNavigator from '@/components/AgentStepNavigator.vue'

const route = useRoute()
const token = computed(() => route.params.token)

const loading = ref(true)
const error = ref('')
const analysisData = ref(null)
const agentSteps = ref([])
const activeTab = ref('overview')

const analysis = computed(() => analysisData.value?.result?.analysis || null)
const pageMeta = computed(() => analysisData.value?.result?.pageMeta || null)
const flows = computed(() => analysisData.value?.result?.flows || [])
const devices = computed(() => analysisData.value?.result?.devices || [])
const isAgentMode = computed(() => analysisData.value?.result?.mode === 'agent')
const framework = computed(() => pageMeta.value?.framework || '')

const gameName = computed(() => {
  return analysis.value?.gameInfo?.name || pageMeta.value?.title || 'Analysis Report'
})

const enabledModules = computed(() => {
  try { return JSON.parse(analysisData.value?.modules || '{}') } catch { return {} }
})
const showUiuxTab = computed(() => enabledModules.value.uiux !== false)
const showWordingTab = computed(() => enabledModules.value.wording !== false)
const showGameDesignTab = computed(() => enabledModules.value.gameDesign !== false)
const showFlowsTab = computed(() => enabledModules.value.testFlows !== false)
const showGliTab = computed(() => enabledModules.value.gli === true)

const uiuxCount = computed(() => analysis.value?.uiuxAnalysis?.length || 0)
const wordingCount = computed(() => analysis.value?.wordingCheck?.length || 0)
const gameDesignCount = computed(() => analysis.value?.gameDesign?.length || 0)
const gliCount = computed(() => analysis.value?.gliCompliance?.length || 0)
const flowCount = computed(() => flows.value?.length || 0)

onMounted(async () => {
  try {
    const data = await sharedApi.getAnalysis(token.value)
    analysisData.value = data

    if (data.result?.mode === 'agent') {
      try {
        const stepsData = await sharedApi.getSteps(token.value)
        agentSteps.value = stepsData.steps || []
      } catch {
        // Steps may not be available
      }
    }
  } catch (err) {
    error.value = err.message || 'This shared report is no longer available'
  } finally {
    loading.value = false
  }
})
</script>
