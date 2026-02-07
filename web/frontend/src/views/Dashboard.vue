<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Dashboard</h2>
        <p class="text-muted-foreground">Overview of your testing activity</p>
      </div>
      <router-link to="/analyze">
        <Button>
          <Sparkles class="h-4 w-4 mr-2" />
          Analyze Game
        </Button>
      </router-link>
    </div>

    <!-- Loading State -->
    <template v-if="loading">
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <LoadingSkeleton variant="card" :count="4" />
      </div>
      <div class="grid gap-4 md:grid-cols-7 mb-6">
        <LoadingSkeleton variant="chart" :count="1" class="col-span-4" />
        <LoadingSkeleton variant="chart" :count="1" class="col-span-3" />
      </div>
    </template>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive" class="mb-6">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <template v-else>
      <!-- Stat Cards -->
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <StatCard title="Total Tests" :value="stats.totalTests" :icon="Activity" />
        <StatCard title="Passed" :value="stats.passedTests" :icon="CheckCircle2" trend="12%" :trend-up="true" />
        <StatCard title="Failed" :value="stats.failedTests" :icon="XCircle" />
        <StatCard
          title="Success Rate"
          :value="stats.avgSuccessRate"
          suffix="%"
          :icon="TrendingUp"
        />
      </div>

      <!-- Secondary stats row -->
      <div class="grid gap-4 md:grid-cols-3 mb-6">
        <Card class="cursor-pointer hover:shadow-md transition-shadow" @click="$router.push('/analyze')">
          <CardContent class="pt-6">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-muted-foreground">Analyses</p>
                <p class="text-2xl font-bold">{{ stats.totalAnalyses || 0 }}</p>
              </div>
              <Sparkles class="h-8 w-8 text-muted-foreground/40" />
            </div>
          </CardContent>
        </Card>
        <Card class="cursor-pointer hover:shadow-md transition-shadow" @click="$router.push('/flows')">
          <CardContent class="pt-6">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-muted-foreground">Flows</p>
                <p class="text-2xl font-bold">{{ stats.totalFlows || 0 }}</p>
              </div>
              <GitBranch class="h-8 w-8 text-muted-foreground/40" />
            </div>
          </CardContent>
        </Card>
        <Card class="cursor-pointer hover:shadow-md transition-shadow" @click="$router.push('/tests')">
          <CardContent class="pt-6">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-muted-foreground">Test Plans</p>
                <p class="text-2xl font-bold">{{ stats.totalPlans || 0 }}</p>
              </div>
              <FlaskConical class="h-8 w-8 text-muted-foreground/40" />
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Charts Row -->
      <div class="grid gap-4 md:grid-cols-7 mb-6">
        <Card class="col-span-4">
          <CardHeader>
            <CardTitle class="text-base">Test History</CardTitle>
          </CardHeader>
          <CardContent>
            <TestHistoryChart :data="stats.history || []" />
          </CardContent>
        </Card>

        <Card class="col-span-3">
          <CardHeader>
            <CardTitle class="text-base">Pass / Fail Ratio</CardTitle>
          </CardHeader>
          <CardContent>
            <PassFailDonut :passed="stats.passedTests" :failed="stats.failedTests" />
          </CardContent>
        </Card>
      </div>

      <!-- Recent Tests -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Recent Tests</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Success Rate</TableHead>
                <TableHead class="text-right">Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="test in stats.recentTests" :key="test.name">
                <TableCell class="font-medium">{{ test.name }}</TableCell>
                <TableCell><StatusBadge :status="test.status" /></TableCell>
                <TableCell>{{ test.duration || '-' }}</TableCell>
                <TableCell>{{ test.successRate ? test.successRate + '%' : '-' }}</TableCell>
                <TableCell class="text-right text-muted-foreground">
                  {{ new Date(test.timestamp).toLocaleString() }}
                </TableCell>
              </TableRow>
              <TableRow v-if="!stats.recentTests?.length">
                <TableCell colspan="5" class="text-center text-muted-foreground py-8">
                  No recent tests
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Activity, CheckCircle2, XCircle, TrendingUp, AlertCircle, Sparkles, GitBranch, FlaskConical } from 'lucide-vue-next'
import { statsApi } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import StatCard from '@/components/StatCard.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'
import TestHistoryChart from '@/components/charts/TestHistoryChart.vue'
import PassFailDonut from '@/components/charts/PassFailDonut.vue'

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

onMounted(() => {
  loadStats()
  // Auto-refresh every 30 seconds
  refreshInterval = setInterval(loadStats, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>
