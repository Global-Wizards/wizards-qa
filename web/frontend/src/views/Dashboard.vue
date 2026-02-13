<script setup>
import { ref, computed, onMounted, h } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { createColumnHelper } from '@tanstack/vue-table'
import {
  Activity, CheckCircle2, XCircle, TrendingUp, AlertCircle, Sparkles,
  GitBranch, FlaskConical, ChevronRight, HeartPulse, ShieldCheck,
  ArrowRight, Clock, FileText, BarChart3, RefreshCw, FolderKanban,
  Gamepad2, Swords, Trophy,
} from 'lucide-vue-next'
import { statsApi, projectsApi } from '@/lib/api'
import { timeAgo, fullTimestamp } from '@/lib/dateUtils'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { DataTable, DataTableColumnHeader } from '@/components/ui/data-table'
import StatCard from '@/components/StatCard.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'
import TestHistoryChart from '@/components/charts/TestHistoryChart.vue'
import PassFailDonut from '@/components/charts/PassFailDonut.vue'
import { ShimmerButton } from '@/components/ui/shimmer-button'

const router = useRouter()
const loading = ref(true)
const error = ref(null)
const stats = ref({
  totalTests: 0,
  passedTests: 0,
  failedTests: 0,
  avgSuccessRate: 0,
  totalAnalyses: 0,
  totalFlows: 0,
  totalPlans: 0,
  recentTests: [],
  history: [],
})

const projects = ref([])
const onboardingDismissed = ref(false)

async function loadStats() {
  try {
    stats.value = await statsApi.getStats()
  } catch (err) {
    if (loading.value) {
      error.value = err.message
    } else if (err?.response?.status === 401 || err?.response?.status === 403) {
      error.value = 'Session expired. Please refresh the page.'
    }
  } finally {
    loading.value = false
  }
}

const allProjects = ref([])

async function loadProjects() {
  try {
    const data = await projectsApi.list()
    allProjects.value = data.projects || []
    projects.value = allProjects.value.slice(0, 3)
  } catch {
    // non-critical
  }
}

const hasNoProjects = computed(() => allProjects.value.length === 0 && !loading.value && !onboardingDismissed.value)

const isEmpty = computed(() =>
  stats.value.totalTests === 0 &&
  stats.value.totalAnalyses === 0 &&
  stats.value.totalFlows === 0 &&
  stats.value.totalPlans === 0
)

const healthStatus = computed(() => {
  if (stats.value.totalTests === 0) {
    return { label: 'No Data', color: 'text-muted-foreground', bg: 'bg-muted', icon: HeartPulse, description: 'Run tests to see health' }
  }
  const rate = stats.value.avgSuccessRate
  if (rate >= 90) {
    return { label: 'Excellent', color: 'text-emerald-500', bg: 'bg-emerald-500/10', icon: ShieldCheck, description: 'Tests are passing consistently' }
  }
  if (rate >= 70) {
    return { label: 'Good', color: 'text-yellow-500', bg: 'bg-yellow-500/10', icon: HeartPulse, description: 'Some tests need attention' }
  }
  return { label: 'Needs Attention', color: 'text-red-500', bg: 'bg-red-500/10', icon: AlertCircle, description: 'Many tests are failing' }
})

const visibleTests = computed(() => (stats.value.recentTests || []).slice(0, 5))
const hasMoreTests = computed(() => (stats.value.recentTests || []).length > 5)

const successRateColor = computed(() => {
  const rate = stats.value.avgSuccessRate
  if (rate >= 70) return 'text-emerald-500'
  if (rate > 0) return 'text-red-500'
  return 'text-muted-foreground'
})

// Recent Tests DataTable
const recentTestsSorting = ref([{ id: 'timestamp', desc: true }])
const columnHelper = createColumnHelper()

const recentTestColumns = [
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
    cell: (info) => info.getValue() || '-',
    meta: { class: 'hidden sm:table-cell' },
  }),
  columnHelper.accessor('successRate', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Success Rate' }),
    cell: (info) => {
      const val = info.getValue()
      if (!val) return '-'
      return h('span', { class: val >= 70 ? 'text-emerald-500' : 'text-red-500' }, `${val}%`)
    },
    meta: { class: 'hidden md:table-cell' },
  }),
  columnHelper.accessor('timestamp', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Timestamp' }),
    cell: (info) => h(Tooltip, null, {
      default: () => h(TooltipTrigger, { asChild: true }, () =>
        h('span', { class: 'text-muted-foreground cursor-default' }, timeAgo(info.getValue()))
      ),
      content: () => h(TooltipContent, null, () => h('p', null, fullTimestamp(info.getValue()))),
    }),
    meta: { class: 'text-right' },
  }),
]

function onRecentTestClick(row) {
  router.push(`/tests/run/${row.original.id}`)
}

useIntervalFn(loadStats, 30000)

onMounted(() => {
  loadStats()
  loadProjects()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Command Center</h2>
        <p class="text-muted-foreground">Your QA campaign at a glance</p>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="icon" @click="loadStats" :disabled="loading">
          <RefreshCw class="h-4 w-4" :class="loading && 'animate-spin'" />
        </Button>
        <router-link to="/analyze">
          <Button>
            <Sparkles class="h-4 w-4 mr-2" />
            Analyze Game
          </Button>
        </router-link>
      </div>
    </div>

    <!-- Loading State -->
    <template v-if="loading">
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <LoadingSkeleton variant="card" :count="4" />
      </div>
      <div class="grid gap-4 md:grid-cols-7">
        <LoadingSkeleton variant="chart" :count="1" class="col-span-full md:col-span-4" />
        <LoadingSkeleton variant="chart" :count="1" class="col-span-full md:col-span-3" />
      </div>
    </template>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Project Onboarding -->
    <template v-else-if="hasNoProjects">
      <div class="onboarding-card relative overflow-hidden rounded-xl border border-border/50 bg-card">
        <div class="onboarding-bg absolute inset-0 pointer-events-none" />
        <div class="relative flex flex-col items-center justify-center py-20 px-6 text-center">
          <div class="flex items-center gap-3 mb-6">
            <div class="onboarding-icon rounded-lg bg-primary/15 p-3">
              <Gamepad2 class="h-8 w-8 text-primary" />
            </div>
            <div class="onboarding-icon rounded-lg bg-violet-500/15 p-3" style="animation-delay: 0.1s">
              <Swords class="h-8 w-8 text-violet-400" />
            </div>
            <div class="onboarding-icon rounded-lg bg-amber-500/15 p-3" style="animation-delay: 0.2s">
              <Trophy class="h-8 w-8 text-amber-400" />
            </div>
          </div>
          <h3 class="text-2xl font-bold mb-2 tracking-tight">Ready to Begin Your Quest?</h3>
          <p class="text-muted-foreground max-w-lg mb-8 leading-relaxed">
            Create your first project to start testing. Each project is a campaign — organize your game analyses, test plans, and QA reports all in one place.
          </p>
          <div class="flex flex-col sm:flex-row gap-3">
            <router-link to="/projects/new">
              <ShimmerButton border-radius="8px">
                <Swords class="h-4 w-4 mr-2" />
                Start New Campaign
              </ShimmerButton>
            </router-link>
            <Button variant="outline" @click="onboardingDismissed = true">
              Explore the Arena
            </Button>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <!-- Empty State -->
      <template v-if="isEmpty">
        <Card class="border-dashed border-border/50">
          <CardContent class="flex flex-col items-center justify-center py-16 text-center">
            <div class="rounded-lg bg-primary/15 p-4 mb-6">
              <Sparkles class="h-10 w-10 text-primary" />
            </div>
            <h3 class="text-2xl font-bold mb-2 tracking-tight">Your Arena Awaits</h3>
            <p class="text-muted-foreground max-w-md mb-8">
              Analyze a game or create a test plan to fill your dashboard with live data. Every great QA campaign starts with a single test.
            </p>
            <div class="flex flex-col sm:flex-row gap-3">
              <router-link to="/analyze">
                <ShimmerButton border-radius="8px">
                  <Sparkles class="h-4 w-4 mr-2" />
                  Scout Your First Game
                </ShimmerButton>
              </router-link>
              <router-link to="/tests/new">
                <Button variant="outline">
                  <FileText class="h-4 w-4 mr-2" />
                  Draft a Test Plan
                </Button>
              </router-link>
            </div>
          </CardContent>
        </Card>
      </template>

      <template v-else>
        <!-- Primary Stat Cards -->
        <div class="grid gap-4 grid-cols-2 lg:grid-cols-4 stagger-grid">
          <StatCard
            title="Total Tests"
            :value="stats.totalTests"
            :icon="Activity"
            description="All test executions"
          />
          <StatCard
            title="Passed"
            :value="stats.passedTests"
            :icon="CheckCircle2"
            icon-color="text-emerald-500"
            description="Successful tests"
          />
          <StatCard
            title="Failed"
            :value="stats.failedTests"
            :icon="XCircle"
            icon-color="text-red-500"
            description="Failed tests"
          />
          <StatCard
            title="Success Rate"
            :value="stats.avgSuccessRate"
            suffix="%"
            :icon="TrendingUp"
            :icon-color="successRateColor"
            description="Average pass rate"
          />
        </div>

        <!-- Secondary Stats + Health -->
        <div class="grid gap-4 grid-cols-2 lg:grid-cols-4 stagger-grid">
          <StatCard
            title="Analyses"
            :value="stats.totalAnalyses || 0"
            :icon="Sparkles"
            to="/analyze"
            description="Game analyses"
          />
          <StatCard
            title="Flows"
            :value="stats.totalFlows || 0"
            :icon="GitBranch"
            to="/flows"
            description="Test flows"
          />
          <StatCard
            title="Test Plans"
            :value="stats.totalPlans || 0"
            :icon="FlaskConical"
            to="/tests"
            description="Created plans"
          />
          <!-- Testing Health Card -->
          <Card class="relative overflow-hidden">
            <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle class="text-sm font-medium">Testing Health</CardTitle>
              <component :is="healthStatus.icon" class="h-4 w-4" :class="healthStatus.color" />
            </CardHeader>
            <CardContent>
              <div class="text-2xl font-bold" :class="healthStatus.color">
                {{ healthStatus.label }}
              </div>
              <p class="text-xs text-muted-foreground mt-1">
                {{ healthStatus.description }}
              </p>
            </CardContent>
            <div class="absolute bottom-0 left-0 right-0 h-1" :class="healthStatus.bg" />
          </Card>
        </div>

        <!-- Quick Actions -->
        <div>
          <h3 class="text-lg font-semibold mb-3">Quick Actions</h3>
          <!-- Game-themed action cards with refined hover -->
          <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 stagger-grid">
            <router-link to="/analyze" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/30 dark:hover:shadow-primary/5">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-primary/10 p-2.5">
                    <Sparkles class="h-5 w-5 text-primary" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">Scout Game</p>
                    <p class="text-sm text-muted-foreground">AI-powered recon on any URL</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/projects/new" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/30 dark:hover:shadow-primary/5">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-violet-500/10 p-2.5">
                    <FolderKanban class="h-5 w-5 text-violet-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">New Campaign</p>
                    <p class="text-sm text-muted-foreground">Launch a new QA project</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/tests/new" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/30 dark:hover:shadow-primary/5">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-blue-500/10 p-2.5">
                    <FileText class="h-5 w-5 text-blue-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">Draft Battle Plan</p>
                    <p class="text-sm text-muted-foreground">Design your testing strategy</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/reports" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/30 dark:hover:shadow-primary/5">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-orange-500/10 p-2.5">
                    <BarChart3 class="h-5 w-5 text-orange-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">War Room</p>
                    <p class="text-sm text-muted-foreground">Review results and intel</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
          </div>
        </div>

        <!-- Your Projects -->
        <div v-if="projects.length">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-lg font-semibold">Your Projects</h3>
            <router-link to="/projects">
              <Button variant="ghost" size="sm">
                View all
                <ArrowRight class="h-4 w-4 ml-1" />
              </Button>
            </router-link>
          </div>
          <div class="grid gap-4 grid-cols-1 sm:grid-cols-3">
            <router-link
              v-for="project in projects"
              :key="project.id"
              :to="`/projects/${project.id}`"
              class="block group"
            >
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/30 dark:hover:shadow-primary/5">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div
                    class="h-10 w-10 rounded-lg flex items-center justify-center text-white text-sm font-bold shrink-0"
                    :style="{ backgroundColor: project.color || '#6366f1' }"
                  >
                    {{ project.name.charAt(0).toUpperCase() }}
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium truncate">{{ project.name }}</p>
                    <p class="text-sm text-muted-foreground">
                      {{ project.analysisCount }} analyses &middot; {{ project.testCount }} tests
                    </p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary shrink-0" />
                </CardContent>
              </Card>
            </router-link>
          </div>
        </div>

        <!-- Charts Row -->
        <div class="grid gap-4 md:grid-cols-7">
          <Card class="col-span-full md:col-span-4">
            <CardHeader>
              <CardTitle class="text-base">Test History</CardTitle>
              <CardDescription>Passed vs. failed tests over the last 14 days</CardDescription>
            </CardHeader>
            <CardContent>
              <template v-if="stats.history && stats.history.length > 0">
                <TestHistoryChart :data="stats.history" />
              </template>
              <div v-else class="flex flex-col items-center justify-center py-12 text-center">
                <BarChart3 class="h-10 w-10 text-muted-foreground/30 mb-3" />
                <p class="text-sm text-muted-foreground">No test history yet</p>
                <p class="text-xs text-muted-foreground/70">Run some tests to see trends here</p>
              </div>
            </CardContent>
          </Card>

          <Card class="col-span-full md:col-span-3">
            <CardHeader>
              <CardTitle class="text-base">Pass / Fail Ratio</CardTitle>
              <CardDescription>Overall test outcome distribution</CardDescription>
            </CardHeader>
            <CardContent>
              <template v-if="stats.passedTests > 0 || stats.failedTests > 0">
                <PassFailDonut :passed="stats.passedTests" :failed="stats.failedTests" />
              </template>
              <div v-else class="flex flex-col items-center justify-center py-12 text-center">
                <Activity class="h-10 w-10 text-muted-foreground/30 mb-3" />
                <p class="text-sm text-muted-foreground">No test results</p>
                <p class="text-xs text-muted-foreground/70">Results will appear after running tests</p>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Recent Tests Table -->
        <Card>
          <CardHeader class="flex flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle class="text-base">Recent Tests</CardTitle>
              <CardDescription>Latest test executions</CardDescription>
            </div>
            <router-link v-if="hasMoreTests" to="/tests">
              <Button variant="ghost" size="sm">
                View all
                <ArrowRight class="h-4 w-4 ml-1" />
              </Button>
            </router-link>
          </CardHeader>
          <CardContent>
            <DataTable
              :columns="recentTestColumns"
              :data="visibleTests"
              :sorting="recentTestsSorting"
              :on-row-click="onRecentTestClick"
              empty-text="No recent tests"
              @update:sorting="recentTestsSorting = $event"
            >
              <template #empty>
                <div class="flex flex-col items-center gap-3">
                  <Clock class="h-10 w-10 text-muted-foreground/30" />
                  <div>
                    <p class="text-sm text-muted-foreground">No recent tests</p>
                    <p class="text-xs text-muted-foreground/70 mb-3">Your test results will appear here</p>
                  </div>
                  <router-link to="/tests/new">
                    <Button variant="outline" size="sm">
                      <FileText class="h-4 w-4 mr-2" />
                      Create Test Plan
                    </Button>
                  </router-link>
                </div>
              </template>
            </DataTable>
          </CardContent>
        </Card>
      </template>
    </template>
  </div>
</template>

<style scoped>
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stagger-grid > * {
  animation: fadeInUp 0.4s ease-out both;
}

.stagger-grid > *:nth-child(1) { animation-delay: 0ms; }
.stagger-grid > *:nth-child(2) { animation-delay: 50ms; }
.stagger-grid > *:nth-child(3) { animation-delay: 100ms; }
.stagger-grid > *:nth-child(4) { animation-delay: 150ms; }

.onboarding-bg {
  background: radial-gradient(ellipse at 50% 0%, hsl(var(--primary) / 0.08) 0%, transparent 70%);
}

.dark .onboarding-bg {
  background: radial-gradient(ellipse at 50% 0%, hsl(var(--primary) / 0.12) 0%, transparent 70%);
}

.dark .onboarding-card {
  box-shadow: 0 0 40px -12px hsl(var(--primary) / 0.15);
}

@keyframes iconFloat {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.9);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.onboarding-icon {
  animation: iconFloat 0.5s ease-out both;
}
</style>
