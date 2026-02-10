<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Analyses</h2>
        <p class="text-muted-foreground">Browse and manage past game analyses.</p>
      </div>
      <Button @click="navigateToNewAnalysis">
        <Plus class="h-4 w-4 mr-1" />
        New Analysis
      </Button>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-3 mb-4">
      <Input
        v-model="searchQuery"
        placeholder="Search by game name or URL..."
        class="max-w-xs"
      />
      <Select :model-value="statusFilter" @update:model-value="statusFilter = $event">
        <SelectTrigger class="w-[140px]">
          <SelectValue placeholder="Status" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All</SelectItem>
          <SelectItem value="running">Running</SelectItem>
          <SelectItem value="completed">Completed</SelectItem>
          <SelectItem value="failed">Failed</SelectItem>
        </SelectContent>
      </Select>
      <div class="flex items-center gap-1.5">
        <button
          v-for="mod in moduleFilters"
          :key="mod.key"
          :class="[
            'inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium border transition-colors cursor-pointer',
            mod.active
              ? 'bg-primary/10 text-primary border-primary/30'
              : 'bg-muted text-muted-foreground border-border hover:bg-accent/50'
          ]"
          @click="mod.active = !mod.active"
        >
          <component :is="mod.icon" class="h-3 w-3" />
          {{ mod.label }}
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-3">
      <Skeleton v-for="i in 5" :key="i" class="h-20 w-full" />
    </div>

    <!-- Empty -->
    <div v-else-if="filteredAnalyses.length === 0" class="text-center py-12 text-muted-foreground">
      <Sparkles class="h-10 w-10 mx-auto mb-3 opacity-30" />
      <p class="text-lg font-medium">No analyses found</p>
      <p class="text-sm mt-1">{{ analyses.length === 0 ? 'Start your first analysis to see results here.' : 'Try adjusting your filters.' }}</p>
    </div>

    <!-- List -->
    <div v-else class="space-y-2">
      <div
        v-for="item in filteredAnalyses"
        :key="item.id"
        class="flex items-center gap-4 p-4 rounded-lg border hover:bg-muted/50 transition-colors cursor-pointer"
        @click="viewAnalysis(item)"
      >
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1">
            <p class="text-sm font-medium truncate">{{ item.gameName || 'Untitled' }}</p>
            <Badge :variant="statusVariant(item.status)" class="shrink-0 text-xs">
              <Loader2 v-if="item.status === 'running'" class="h-3 w-3 mr-1 animate-spin" />
              {{ item.status }}
            </Badge>
          </div>
          <p class="text-xs text-muted-foreground truncate" :title="item.gameUrl">{{ item.gameUrl }}</p>
          <div class="flex items-center gap-2 mt-1.5">
            <span v-if="item.framework" class="text-xs text-muted-foreground capitalize">{{ item.framework }}</span>
            <span v-if="item.framework && item.flowCount" class="text-xs text-muted-foreground">&middot;</span>
            <span v-if="item.flowCount" class="text-xs text-muted-foreground">{{ item.flowCount }} flow(s)</span>
            <!-- Module badges -->
            <template v-if="parsedModules(item)">
              <span class="text-xs text-muted-foreground">&middot;</span>
              <span v-if="parsedModules(item).uiux !== false" class="inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0 rounded-full bg-blue-500/10 text-blue-600 dark:text-blue-400">
                <Eye class="h-2.5 w-2.5" />UI/UX
              </span>
              <span v-if="parsedModules(item).wording !== false" class="inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0 rounded-full bg-amber-500/10 text-amber-600 dark:text-amber-400">
                <Type class="h-2.5 w-2.5" />Wording
              </span>
              <span v-if="parsedModules(item).gameDesign !== false" class="inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0 rounded-full bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <Gamepad2 class="h-2.5 w-2.5" />Design
              </span>
              <span v-if="parsedModules(item).testFlows !== false" class="inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0 rounded-full bg-green-500/10 text-green-600 dark:text-green-400">
                <PlayCircle class="h-2.5 w-2.5" />Flows
              </span>
            </template>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <span class="text-xs text-muted-foreground whitespace-nowrap" :title="fullTimestamp(item.createdAt)">{{ timeAgo(item.createdAt) }}</span>
          <Button variant="ghost" size="sm" @click.stop="reAnalyze(item)">
            <RefreshCw class="h-3 w-3" />
          </Button>
          <Button variant="ghost" size="sm" @click.stop="deleteAnalysis(item)">
            <Trash2 class="h-3 w-3 text-destructive" />
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { analysesApi, projectsApi } from '@/lib/api'
import { timeAgo, fullTimestamp } from '@/lib/dateUtils'
import { Plus, RefreshCw, Trash2, Loader2, Sparkles, Eye, Type, Gamepad2, PlayCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'

const route = useRoute()
const router = useRouter()
const projectId = computed(() => route.params.projectId || '')

const loading = ref(true)
const analyses = ref([])
const searchQuery = ref('')
const statusFilter = ref('all')

const moduleFilters = reactive([
  { key: 'uiux', label: 'UI/UX', icon: Eye, active: false },
  { key: 'wording', label: 'Wording', icon: Type, active: false },
  { key: 'gameDesign', label: 'Design', icon: Gamepad2, active: false },
  { key: 'testFlows', label: 'Flows', icon: PlayCircle, active: false },
])

function parsedModules(item) {
  if (!item.modules) return null
  try { return JSON.parse(item.modules) } catch { return null }
}

function statusVariant(status) {
  switch (status) {
    case 'completed': return 'default'
    case 'running': return 'secondary'
    case 'failed': return 'destructive'
    default: return 'outline'
  }
}

const filteredAnalyses = computed(() => {
  let list = analyses.value

  if (statusFilter.value !== 'all') {
    list = list.filter((a) => a.status === statusFilter.value)
  }

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(
      (a) =>
        (a.gameName || '').toLowerCase().includes(q) ||
        (a.gameUrl || '').toLowerCase().includes(q)
    )
  }

  const activeModuleFilters = moduleFilters.filter((m) => m.active)
  if (activeModuleFilters.length > 0) {
    list = list.filter((a) => {
      const mods = parsedModules(a)
      if (!mods) return false
      return activeModuleFilters.every((f) => mods[f.key] !== false)
    })
  }

  return list
})

function navigateToNewAnalysis() {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${basePath}/analyze`)
}

function viewAnalysis(item) {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  if (item.status === 'running') {
    router.push({ path: `${basePath}/analyze`, query: { analysisId: item.id } })
  } else {
    router.push(`${basePath}/analyses/${item.id}`)
  }
}

function reAnalyze(item) {
  const basePath = projectId.value ? `/projects/${projectId.value}` : ''
  router.push({ path: `${basePath}/analyze`, query: { gameUrl: item.gameUrl } })
}

async function deleteAnalysis(item) {
  if (!confirm(`Delete analysis "${item.gameName || item.id}"? This cannot be undone.`)) return
  try {
    await analysesApi.delete(item.id)
    analyses.value = analyses.value.filter((a) => a.id !== item.id)
  } catch (err) {
    console.error('Failed to delete analysis:', err)
  }
}

onMounted(async () => {
  try {
    const data = projectId.value
      ? await projectsApi.analyses(projectId.value)
      : await analysesApi.list()
    analyses.value = data.analyses || []
  } catch (err) {
    console.error('Failed to load analyses:', err)
  } finally {
    loading.value = false
  }
})
</script>
