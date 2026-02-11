<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Tests</h2>
        <p class="text-muted-foreground">View test results and manage test plans</p>
      </div>
      <router-link :to="projectId ? `/projects/${projectId}/tests/new` : '/tests/new'">
        <Button>
          <Plus class="h-4 w-4 mr-2" />
          New Test Plan
        </Button>
      </router-link>
    </div>

    <!-- Main Tabs -->
    <Tabs v-model="activeTab" class="space-y-4">
      <TabsList>
        <TabsTrigger value="results">Test Results</TabsTrigger>
        <TabsTrigger value="plans">Test Plans</TabsTrigger>
      </TabsList>

      <!-- Test Results Tab -->
      <TabsContent value="results">
        <!-- Search -->
        <div class="mb-4">
          <Input v-model="search" placeholder="Search tests..." class="max-w-sm" />
        </div>

        <!-- Loading State -->
        <Card v-if="loading">
          <CardContent class="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead v-for="i in 5" :key="i"><Skeleton class="h-4 w-20" /></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <LoadingSkeleton variant="table-row" :count="5" />
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <!-- Error State -->
        <Alert v-else-if="error" variant="destructive" class="mb-6">
          <AlertCircle class="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{{ error }}</AlertDescription>
        </Alert>

        <!-- Tests Table -->
        <Card v-else>
          <CardContent class="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="cursor-pointer" @click="toggleSort('name')">
                    Name
                    <span v-if="sortField === 'name'" class="ml-1">{{ sortAsc ? '↑' : '↓' }}</span>
                  </TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead class="cursor-pointer" @click="toggleSort('duration')">
                    Duration
                    <span v-if="sortField === 'duration'" class="ml-1">{{ sortAsc ? '↑' : '↓' }}</span>
                  </TableHead>
                  <TableHead class="cursor-pointer" @click="toggleSort('successRate')">
                    Success Rate
                    <span v-if="sortField === 'successRate'" class="ml-1">{{ sortAsc ? '↑' : '↓' }}</span>
                  </TableHead>
                  <TableHead class="text-right cursor-pointer" @click="toggleSort('timestamp')">
                    Timestamp
                    <span v-if="sortField === 'timestamp'" class="ml-1">{{ sortAsc ? '↑' : '↓' }}</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="test in filteredTests"
                  :key="test.id"
                  class="cursor-pointer"
                  @click="openDetail(test)"
                >
                  <TableCell class="font-medium">{{ test.name }}</TableCell>
                  <TableCell><StatusBadge :status="test.status" /></TableCell>
                  <TableCell>{{ test.duration }}</TableCell>
                  <TableCell>{{ Math.round(test.successRate) }}%</TableCell>
                  <TableCell class="text-right text-muted-foreground">
                    {{ formatDate(test.timestamp) }}
                  </TableCell>
                </TableRow>
                <TableRow v-if="!filteredTests.length">
                  <TableCell colspan="5" class="text-center text-muted-foreground py-8">
                    {{ search ? 'No tests match your search' : 'No tests found' }}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Test Plans Tab -->
      <TabsContent value="plans">
        <!-- Run Error -->
        <Alert v-if="runError" variant="destructive" class="mb-4">
          <AlertCircle class="h-4 w-4" />
          <AlertTitle>Run Failed</AlertTitle>
          <AlertDescription>{{ runError }}</AlertDescription>
        </Alert>

        <!-- Plans Error -->
        <Alert v-if="plansError" variant="destructive" class="mb-4">
          <AlertCircle class="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{{ plansError }}</AlertDescription>
        </Alert>

        <!-- Loading -->
        <Card v-if="plansLoading">
          <CardContent class="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead v-for="i in 5" :key="i"><Skeleton class="h-4 w-20" /></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <LoadingSkeleton variant="table-row" :count="3" />
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <!-- Plans Table -->
        <Card v-else>
          <CardContent class="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Flows</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead class="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="plan in plans"
                  :key="plan.id"
                  :class="plan.status === 'running' ? 'cursor-pointer hover:bg-muted/50' : ''"
                  @click="plan.status === 'running' && plan.lastRunId ? navigateToRunning(plan) : null"
                >
                  <TableCell class="font-medium">
                    <div class="flex items-center gap-2">
                      <Loader2
                        v-if="plan.status === 'running'"
                        class="h-3.5 w-3.5 animate-spin text-primary shrink-0"
                      />
                      {{ plan.name }}
                    </div>
                  </TableCell>
                  <TableCell><StatusBadge :status="plan.status" /></TableCell>
                  <TableCell>{{ plan.flowCount }}</TableCell>
                  <TableCell class="text-muted-foreground">
                    {{ formatDate(plan.createdAt) }}
                  </TableCell>
                  <TableCell class="text-right">
                    <div class="flex items-center justify-end gap-1">
                      <Button
                        v-if="plan.status === 'running' && plan.lastRunId"
                        size="sm"
                        variant="outline"
                        @click.stop="navigateToRunning(plan)"
                      >
                        <Eye class="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button
                        size="sm"
                        :disabled="plan.status === 'running'"
                        @click.stop="runPlan(plan)"
                      >
                        <Play class="h-3 w-3 mr-1" />
                        Run
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        :disabled="plan.status === 'running'"
                        @click.stop="deletePlan(plan)"
                      >
                        <Trash2 class="h-3 w-3 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
                <TableRow v-if="!plans.length">
                  <TableCell colspan="5" class="text-center text-muted-foreground py-8">
                    No test plans yet. Create one to get started.
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>

    <!-- Detail Sheet (completed test results only) -->
    <Sheet :open="sheetOpen" @update:open="sheetOpen = $event">
      <SheetContent side="right">
        <SheetHeader>
          <SheetTitle>{{ selectedTest?.name }}</SheetTitle>
          <SheetDescription>Test execution details</SheetDescription>
        </SheetHeader>
        <div v-if="selectedTest" class="mt-6 space-y-4">
          <Alert v-if="detailData?._fetchError" variant="destructive" class="mb-2">
            <AlertCircle class="h-4 w-4" />
            <AlertDescription>Could not load full test details. Showing summary only.</AlertDescription>
          </Alert>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Status</span>
            <StatusBadge :status="selectedTest.status" />
          </div>
          <Separator />
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Duration</span>
            <span class="text-sm font-medium">{{ selectedTest.duration }}</span>
          </div>
          <Separator />
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Success Rate</span>
            <span class="text-sm font-medium">{{ Math.round(selectedTest.successRate) }}%</span>
          </div>
          <Separator />

          <!-- Flow Results -->
          <div v-if="detailData?.flows?.length">
            <h4 class="text-sm font-medium mb-3">Flow Results</h4>
            <div class="space-y-2">
              <div
                v-for="flow in detailData.flows"
                :key="flow.name"
                class="flex items-center justify-between rounded-md border p-3"
              >
                <div>
                  <p class="text-sm font-medium">{{ flow.name }}</p>
                  <p class="text-xs text-muted-foreground">{{ flow.duration }}</p>
                </div>
                <StatusBadge :status="flow.status" />
              </div>
            </div>
          </div>

          <!-- Error Output -->
          <div v-if="detailData?.errorOutput">
            <h4 class="text-sm font-medium mb-2">Error Output</h4>
            <pre class="text-xs bg-muted rounded-md p-3 overflow-auto max-h-48">{{ detailData.errorOutput }}</pre>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, Plus, Play, Trash2, Eye, Loader2 } from 'lucide-vue-next'
import { testsApi, testPlansApi, testPlansDeleteApi, projectsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { getWebSocket } from '@/lib/websocket'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import StatusBadge from '@/components/StatusBadge.vue'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'

const route = useRoute()
const router = useRouter()
const projectId = computed(() => route.params.projectId || '')

const activeTab = ref('results')
const loading = ref(true)
const error = ref(null)
const tests = ref([])
const search = ref('')
const debouncedSearch = ref('')
let searchTimeout = null
watch(search, (val) => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => { debouncedSearch.value = val }, 300)
})
const sortField = ref('timestamp')
const sortAsc = ref(false)
const sheetOpen = ref(false)
const selectedTest = ref(null)
const detailData = ref(null)

// Plans state
const plansLoading = ref(true)
const plansError = ref(null)
const plans = ref([])

// Run error
const runError = ref(null)

const filteredTests = computed(() => {
  let result = [...tests.value]

  if (debouncedSearch.value) {
    const q = debouncedSearch.value.toLowerCase()
    result = result.filter((t) => t.name.toLowerCase().includes(q))
  }

  result.sort((a, b) => {
    const aVal = a[sortField.value]
    const bVal = b[sortField.value]
    const cmp = aVal < bVal ? -1 : aVal > bVal ? 1 : 0
    return sortAsc.value ? cmp : -cmp
  })

  return result
})

function toggleSort(field) {
  if (sortField.value === field) {
    sortAsc.value = !sortAsc.value
  } else {
    sortField.value = field
    sortAsc.value = true
  }
}

async function openDetail(test) {
  selectedTest.value = test
  sheetOpen.value = true
  detailData.value = null
  try {
    detailData.value = await testsApi.get(test.id)
  } catch {
    detailData.value = { ...test, _fetchError: true }
  }
}

function navigateToRunning(plan) {
  const base = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${base}/tests/run/${plan.lastRunId}`)
}

async function runPlan(plan) {
  runError.value = null
  try {
    const data = await testPlansApi.run(plan.id)
    plan.status = 'running'
    plan.lastRunId = data.testId
    const base = projectId.value ? `/projects/${projectId.value}` : ''
    router.push({
      path: `${base}/tests/run/${data.testId}`,
      query: { fresh: '1', planId: plan.id, planName: plan.name },
    })
  } catch (err) {
    runError.value = 'Failed to start test: ' + err.message
  }
}

async function deletePlan(plan) {
  if (!confirm(`Delete test plan "${plan.name}"? This cannot be undone.`)) return
  try {
    await testPlansDeleteApi.delete(plan.id)
    plans.value = plans.value.filter((p) => p.id !== plan.id)
  } catch (err) {
    runError.value = 'Failed to delete plan: ' + err.message
  }
}

async function loadPlans() {
  plansLoading.value = true
  plansError.value = null
  try {
    const data = projectId.value
      ? await projectsApi.testPlans(projectId.value)
      : await testPlansApi.list()
    plans.value = data.plans || []
  } catch (err) {
    plansError.value = err.message || 'Failed to load test plans'
  } finally {
    plansLoading.value = false
  }
}

// WebSocket for live updates
let wsCleanup = null

function setupWs() {
  if (wsCleanup) wsCleanup()
  const ws = getWebSocket()
  ws.connect()

  async function refreshTables() {
    try {
      const data = await testsApi.list()
      tests.value = data.tests || []
    } catch (err) {
      console.warn('Failed to refresh tests:', err.message)
    }
    await loadPlans()
  }

  const offStarted = ws.on('test_started', (data) => {
    const plan = plans.value.find(p => p.id === data.planId)
    if (plan) { plan.status = 'running'; plan.lastRunId = data.testId }
  })
  const offCompleted = ws.on('test_completed', refreshTables)
  const offFailed = ws.on('test_failed', refreshTables)

  wsCleanup = () => {
    offStarted()
    offCompleted()
    offFailed()
  }
}

onMounted(async () => {
  setupWs()

  // Support ?tab= query parameter to pre-select a tab
  const initialTab = route.query.tab
  if (initialTab === 'plans' || initialTab === 'results') {
    activeTab.value = initialTab
  }

  try {
    const data = projectId.value
      ? await projectsApi.tests(projectId.value)
      : await testsApi.list()
    tests.value = data.tests || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }

  await loadPlans()

  // Auto-switch to plans tab when results are empty but plans exist (only when no explicit tab param)
  if (!route.query.tab && tests.value.length === 0 && plans.value.length > 0) {
    activeTab.value = 'plans'
  }
})

onUnmounted(() => {
  if (wsCleanup) wsCleanup()
  clearTimeout(searchTimeout)
})
</script>
