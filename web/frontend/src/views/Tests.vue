<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Tests</h2>
        <p class="text-muted-foreground">View test results and manage test plans from analyses</p>
      </div>
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

        <!-- Bulk Action Bar -->
        <div
          v-if="selectedRowCount > 0"
          class="mb-4 flex items-center gap-3 rounded-md border bg-muted/50 px-4 py-2"
        >
          <span class="text-sm font-medium">{{ selectedRowCount }} selected</span>
          <Button size="sm" variant="destructive" @click="deleteSelected">
            <Trash2 class="h-3 w-3 mr-1" />
            Delete Selected
          </Button>
          <Button size="sm" variant="ghost" @click="rowSelection = {}">Clear</Button>
        </div>

        <!-- Loading State -->
        <Card v-if="loading">
          <CardContent class="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead v-for="i in 6" :key="i"><Skeleton class="h-4 w-20" /></TableHead>
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

        <!-- Tests DataTable -->
        <Card v-else>
          <CardContent class="pt-6">
            <DataTable
              ref="dataTableRef"
              :columns="columns"
              :data="tests"
              :sorting="sorting"
              :global-filter="debouncedSearch"
              :row-selection="rowSelection"
              :on-row-click="onRowClick"
              :empty-text="search ? 'No tests match your search' : 'No tests found'"
              @update:sorting="sorting = $event"
              @update:row-selection="rowSelection = $event"
            />
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
                  <TableHead>Analysis</TableHead>
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
                  class="cursor-pointer hover:bg-muted/50"
                  @click="openPlanEditor(plan)"
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
                  <TableCell>
                    <router-link
                      v-if="plan.analysisId"
                      :to="projectId ? `/projects/${projectId}/analyses/${plan.analysisId}` : `/analyses/${plan.analysisId}`"
                      class="text-primary hover:underline text-sm"
                      @click.stop
                    >
                      View Analysis
                    </router-link>
                    <span v-else class="text-muted-foreground text-sm">Manual</span>
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
                        v-if="plan.status !== 'running'"
                        size="sm"
                        variant="outline"
                        @click.stop="openPlanEditor(plan)"
                      >
                        <Pencil class="h-3 w-3 mr-1" />
                        Edit
                      </Button>
                      <Button size="sm" :disabled="plan.status === 'running'" @click.stop="runPlan(plan, plan.mode || 'browser')">
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
                  <TableCell colspan="6" class="text-center text-muted-foreground py-8">
                    No test plans yet. Run an analysis to generate test plans.
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>

<script setup>
import { ref, computed, h, onMounted, onUnmounted } from 'vue'
import { refDebounced } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, Play, Trash2, Eye, Loader2, Pencil } from 'lucide-vue-next'
import { createColumnHelper } from '@tanstack/vue-table'
import { testsApi, testPlansApi, testPlansDeleteApi, projectsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { getWebSocket } from '@/lib/websocket'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { DataTable } from '@/components/ui/data-table'
import { DataTableColumnHeader } from '@/components/ui/data-table'
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
const debouncedSearch = refDebounced(search, 300)

// TanStack Table state
const sorting = ref([{ id: 'timestamp', desc: true }])
const rowSelection = ref({})
const dataTableRef = ref(null)

const selectedRowCount = computed(() => Object.keys(rowSelection.value).length)

// Column definitions
const columnHelper = createColumnHelper()

const columns = [
  columnHelper.display({
    id: 'select',
    header: ({ table }) => h('input', {
      type: 'checkbox',
      checked: table.getIsAllPageRowsSelected(),
      indeterminate: table.getIsSomePageRowsSelected(),
      onChange: (e) => table.toggleAllPageRowsSelected(!!e.target.checked),
      class: 'rounded border-muted-foreground',
    }),
    cell: ({ row }) => h('div', { onClick: (e) => e.stopPropagation() }, [
      h('input', {
        type: 'checkbox',
        checked: row.getIsSelected(),
        onChange: (e) => row.toggleSelected(!!e.target.checked),
        class: 'rounded border-muted-foreground',
      }),
    ]),
    enableSorting: false,
  }),
  columnHelper.accessor('name', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Name' }),
    cell: (info) => h('span', { class: 'font-medium' }, info.getValue()),
  }),
  columnHelper.accessor('status', {
    header: 'Status',
    cell: (info) => h(StatusBadge, { status: info.getValue() }),
    enableSorting: false,
  }),
  columnHelper.accessor('duration', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Duration' }),
  }),
  columnHelper.accessor('successRate', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Success Rate' }),
    cell: (info) => `${Math.round(info.getValue())}%`,
  }),
  columnHelper.accessor('timestamp', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Timestamp' }),
    cell: (info) => h('span', { class: 'text-muted-foreground' }, formatDate(info.getValue())),
  }),
  columnHelper.display({
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Actions'),
    cell: ({ row }) => h('div', { class: 'text-right', onClick: (e) => e.stopPropagation() }, [
      h(Button, {
        variant: 'ghost',
        size: 'sm',
        onClick: () => deleteTest(row.original),
      }, () => h(Trash2, { class: 'h-3 w-3 text-destructive' })),
    ]),
    enableSorting: false,
  }),
]

function onRowClick(row) {
  openDetail(row.original)
}

// Plans state
const plansLoading = ref(true)
const plansError = ref(null)
const plans = ref([])

// Run error
const runError = ref(null)

function openDetail(test) {
  const base = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${base}/tests/run/${test.id}`)
}

function navigateToRunning(plan) {
  const base = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${base}/tests/run/${plan.lastRunId}`)
}

function openPlanEditor(plan) {
  if (plan.status === 'running' && plan.lastRunId) {
    navigateToRunning(plan)
    return
  }
  const base = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${base}/tests/plans/${plan.id}`)
}

async function deleteTest(test) {
  if (!confirm(`Delete test result "${test.name}"? This cannot be undone.`)) return
  try {
    await testsApi.delete(test.id)
    tests.value = tests.value.filter((t) => t.id !== test.id)
    // Clear selection for deleted row
    const newSelection = { ...rowSelection.value }
    const idx = tests.value.findIndex(t => t.id === test.id)
    if (idx >= 0) delete newSelection[idx]
    rowSelection.value = newSelection
  } catch (err) {
    error.value = 'Failed to delete test result: ' + err.message
  }
}

async function deleteSelected() {
  const table = dataTableRef.value?.table
  if (!table) return
  const selectedRows = table.getSelectedRowModel().rows
  const ids = selectedRows.map(r => r.original.id)
  if (!confirm(`Delete ${ids.length} test result(s)? This cannot be undone.`)) return
  try {
    await testsApi.deleteBatch(ids)
    tests.value = tests.value.filter((t) => !ids.includes(t.id))
    rowSelection.value = {}
  } catch (err) {
    error.value = 'Failed to delete test results: ' + err.message
  }
}

async function runPlan(plan, mode = 'browser') {
  runError.value = null
  try {
    const opts = { mode }
    if (mode === 'browser') {
      opts.viewport = 'desktop-std'
    }
    const data = await testPlansApi.run(plan.id, opts)
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
})
</script>
