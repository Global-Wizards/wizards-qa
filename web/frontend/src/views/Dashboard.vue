<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Activity, CheckCircle2, XCircle, TrendingUp, AlertCircle, Sparkles,
  GitBranch, FlaskConical, ChevronRight, HeartPulse, ShieldCheck,
  ArrowRight, Clock, FileText, BarChart3, RefreshCw, FolderKanban,
} from 'lucide-vue-next'
import { statsApi, projectsApi } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import StatCard from '@/components/StatCard.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'
import TestHistoryChart from '@/components/charts/TestHistoryChart.vue'
import PassFailDonut from '@/components/charts/PassFailDonut.vue'

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

let refreshInterval = null

async function loadStats() {
  try {
    stats.value = await statsApi.getStats()
  } catch (err) {
    if (loading.value) {
      error.value = err.message
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

function timeAgo(timestamp) {
  if (!timestamp) return '-'
  const now = Date.now()
  const diff = now - new Date(timestamp).getTime()
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return new Date(timestamp).toLocaleDateString()
}

function fullTimestamp(timestamp) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleString()
}

function openTest(test) {
  router.push('/tests')
}

onMounted(() => {
  loadStats()
  loadProjects()
  refreshInterval = setInterval(loadStats, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Dashboard</h2>
        <p class="text-muted-foreground">Overview of your testing activity</p>
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
      <Card class="border-dashed">
        <CardContent class="flex flex-col items-center justify-center py-20 text-center">
          <div class="rounded-full bg-violet-500/10 p-4 mb-6">
            <FolderKanban class="h-10 w-10 text-violet-500" />
          </div>
          <h3 class="text-2xl font-semibold mb-2">Welcome to Wizards QA</h3>
          <p class="text-muted-foreground max-w-md mb-8">
            Create your first project to get started. Projects help you organize game analyses, tests, and reports in one place.
          </p>
          <div class="flex flex-col sm:flex-row gap-3">
            <router-link to="/projects/new">
              <Button>
                <FolderKanban class="h-4 w-4 mr-2" />
                Create First Project
              </Button>
            </router-link>
            <Button variant="outline" @click="onboardingDismissed = true">
              Explore without a project
            </Button>
          </div>
        </CardContent>
      </Card>
    </template>

    <template v-else>
      <!-- Empty State -->
      <template v-if="isEmpty">
        <Card class="border-dashed">
          <CardContent class="flex flex-col items-center justify-center py-16 text-center">
            <div class="rounded-full bg-primary/10 p-4 mb-6">
              <Sparkles class="h-10 w-10 text-primary" />
            </div>
            <h3 class="text-2xl font-semibold mb-2">Welcome to Wizards QA</h3>
            <p class="text-muted-foreground max-w-md mb-8">
              Get started by analyzing a game or creating a test plan. Your testing dashboard will come to life as you run tests.
            </p>
            <div class="flex flex-col sm:flex-row gap-3">
              <router-link to="/analyze">
                <Button>
                  <Sparkles class="h-4 w-4 mr-2" />
                  Analyze Your First Game
                </Button>
              </router-link>
              <router-link to="/tests/new">
                <Button variant="outline">
                  <FileText class="h-4 w-4 mr-2" />
                  Create Test Plan
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
          <div class="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 stagger-grid">
            <router-link to="/analyze" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/20">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-primary/10 p-2.5">
                    <Sparkles class="h-5 w-5 text-primary" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">Analyze Game</p>
                    <p class="text-sm text-muted-foreground">Scan a game URL with AI</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/projects/new" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/20">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-violet-500/10 p-2.5">
                    <FolderKanban class="h-5 w-5 text-violet-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">New Project</p>
                    <p class="text-sm text-muted-foreground">Organize your testing work</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/tests/new" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/20">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-blue-500/10 p-2.5">
                    <FileText class="h-5 w-5 text-blue-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">Create Test Plan</p>
                    <p class="text-sm text-muted-foreground">Design a new testing strategy</p>
                  </div>
                  <ChevronRight class="h-4 w-4 text-muted-foreground transition-colors group-hover:text-primary" />
                </CardContent>
              </Card>
            </router-link>
            <router-link to="/reports" class="block group">
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/20">
                <CardContent class="flex items-center gap-4 pt-6">
                  <div class="rounded-lg bg-orange-500/10 p-2.5">
                    <BarChart3 class="h-5 w-5 text-orange-500" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium">View Reports</p>
                    <p class="text-sm text-muted-foreground">Review test results and trends</p>
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
              <Card class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary/20">
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead class="hidden sm:table-cell">Duration</TableHead>
                  <TableHead class="hidden md:table-cell">Success Rate</TableHead>
                  <TableHead class="text-right">Timestamp</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="test in visibleTests"
                  :key="test.name"
                  class="cursor-pointer hover:bg-muted/50"
                  @click="openTest(test)"
                >
                  <TableCell class="font-medium">{{ test.name }}</TableCell>
                  <TableCell><StatusBadge :status="test.status" /></TableCell>
                  <TableCell class="hidden sm:table-cell">{{ test.duration || '-' }}</TableCell>
                  <TableCell class="hidden md:table-cell">
                    <span v-if="test.successRate" :class="test.successRate >= 70 ? 'text-emerald-500' : 'text-red-500'">
                      {{ test.successRate }}%
                    </span>
                    <span v-else>-</span>
                  </TableCell>
                  <TableCell class="text-right">
                    <Tooltip>
                      <TooltipTrigger as-child>
                        <span class="text-muted-foreground cursor-default">
                          {{ timeAgo(test.timestamp) }}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>{{ fullTimestamp(test.timestamp) }}</p>
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
                </TableRow>
                <TableRow v-if="!visibleTests.length">
                  <TableCell colspan="5" class="text-center py-12">
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
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
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
</style>
