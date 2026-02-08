<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Activity, CheckCircle2, XCircle, TrendingUp, Sparkles,
  FlaskConical, ArrowRight, RefreshCw,
} from 'lucide-vue-next'
import { projectsApi } from '@/lib/api'
import { useProject } from '@/composables/useProject'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import StatCard from '@/components/StatCard.vue'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'

const route = useRoute()
const router = useRouter()
const { currentProject } = useProject()
const loading = ref(true)
const stats = ref({
  totalTests: 0,
  passedTests: 0,
  failedTests: 0,
  avgSuccessRate: 0,
  totalAnalyses: 0,
  totalPlans: 0,
  recentTests: [],
})

const projectId = computed(() => route.params.projectId)

const successRateColor = computed(() => {
  const rate = stats.value.avgSuccessRate
  if (rate >= 70) return 'text-emerald-500'
  if (rate > 0) return 'text-red-500'
  return 'text-muted-foreground'
})

async function loadStats() {
  try {
    stats.value = await projectsApi.stats(projectId.value)
  } catch (err) {
    console.error('Failed to load project stats:', err)
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <div
          v-if="currentProject"
          class="h-12 w-12 rounded-lg flex items-center justify-center text-white text-lg font-bold shrink-0"
          :style="{ backgroundColor: currentProject.color || '#6366f1' }"
        >
          {{ currentProject.name?.charAt(0)?.toUpperCase() }}
        </div>
        <div>
          <h2 class="text-3xl font-bold tracking-tight">{{ currentProject?.name || 'Project' }}</h2>
          <p class="text-muted-foreground">{{ currentProject?.gameUrl || 'Project dashboard' }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="icon" @click="loadStats" :disabled="loading">
          <RefreshCw class="h-4 w-4" :class="loading && 'animate-spin'" />
        </Button>
        <router-link :to="`/projects/${projectId}/analyze`">
          <Button>
            <Sparkles class="h-4 w-4 mr-2" />
            Analyze
          </Button>
        </router-link>
      </div>
    </div>

    <!-- Description & Tags -->
    <div v-if="currentProject?.description || currentProject?.tags?.length">
      <p v-if="currentProject.description" class="text-sm text-muted-foreground mb-2">{{ currentProject.description }}</p>
      <div v-if="currentProject.tags?.length" class="flex flex-wrap gap-1">
        <Badge v-for="tag in currentProject.tags" :key="tag" variant="secondary">{{ tag }}</Badge>
      </div>
    </div>

    <!-- Loading -->
    <template v-if="loading">
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <LoadingSkeleton variant="card" :count="4" />
      </div>
    </template>

    <template v-else>
      <!-- Stat Cards -->
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <StatCard title="Tests" :value="stats.totalTests" :icon="Activity" description="Test executions" />
        <StatCard title="Passed" :value="stats.passedTests" :icon="CheckCircle2" icon-color="text-emerald-500" description="Successful" />
        <StatCard title="Failed" :value="stats.failedTests" :icon="XCircle" icon-color="text-red-500" description="Failed" />
        <StatCard title="Success Rate" :value="stats.avgSuccessRate" suffix="%" :icon="TrendingUp" :icon-color="successRateColor" description="Average" />
      </div>

      <!-- Quick Stats -->
      <div class="grid gap-4 grid-cols-2 lg:grid-cols-3">
        <Card class="cursor-pointer hover:shadow-md transition-shadow" @click="router.push(`/projects/${projectId}/analyze`)">
          <CardContent class="flex items-center gap-4 pt-6">
            <div class="rounded-lg bg-primary/10 p-2.5">
              <Sparkles class="h-5 w-5 text-primary" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ stats.totalAnalyses || 0 }}</p>
              <p class="text-sm text-muted-foreground">Analyses</p>
            </div>
          </CardContent>
        </Card>
        <Card class="cursor-pointer hover:shadow-md transition-shadow" @click="router.push(`/projects/${projectId}/tests`)">
          <CardContent class="flex items-center gap-4 pt-6">
            <div class="rounded-lg bg-blue-500/10 p-2.5">
              <FlaskConical class="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ stats.totalPlans || 0 }}</p>
              <p class="text-sm text-muted-foreground">Test Plans</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="flex items-center gap-4 pt-6">
            <div class="rounded-lg bg-emerald-500/10 p-2.5">
              <TrendingUp class="h-5 w-5 text-emerald-500" />
            </div>
            <div>
              <p class="text-2xl font-bold" :class="successRateColor">
                {{ stats.avgSuccessRate }}%
              </p>
              <p class="text-sm text-muted-foreground">Success Rate</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <!-- Recent Tests -->
      <Card v-if="stats.recentTests?.length">
        <CardHeader class="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle class="text-base">Recent Tests</CardTitle>
            <CardDescription>Latest test executions in this project</CardDescription>
          </div>
          <router-link :to="`/projects/${projectId}/tests`">
            <Button variant="ghost" size="sm">
              View all
              <ArrowRight class="h-4 w-4 ml-1" />
            </Button>
          </router-link>
        </CardHeader>
        <CardContent>
          <div class="space-y-2">
            <div
              v-for="test in stats.recentTests.slice(0, 5)"
              :key="test.id"
              class="flex items-center justify-between p-2 rounded-md hover:bg-muted/50"
            >
              <div>
                <p class="text-sm font-medium">{{ test.name }}</p>
                <p class="text-xs text-muted-foreground">{{ test.duration }}</p>
              </div>
              <Badge :variant="test.status === 'passed' ? 'default' : 'destructive'">
                {{ test.status }}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
