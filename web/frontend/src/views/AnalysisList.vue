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

    <!-- DataTable -->
    <div v-else>
      <DataTable
        :columns="analysisColumns"
        :data="filteredAnalyses"
        :sorting="analysisSorting"
        :on-row-click="onAnalysisRowClick"
        empty-text="No analyses found"
        @update:sorting="analysisSorting = $event"
      >
        <template #empty>
          <div class="flex flex-col items-center gap-2">
            <Sparkles class="h-10 w-10 text-muted-foreground/30" />
            <p class="text-sm font-medium">No analyses found</p>
            <p class="text-xs text-muted-foreground">{{ analyses.length === 0 ? 'Start your first analysis to see results here.' : 'Try adjusting your filters.' }}</p>
          </div>
        </template>
      </DataTable>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn, useClipboard } from '@vueuse/core'
import { createColumnHelper } from '@tanstack/vue-table'
import { analysesApi, projectsApi } from '@/lib/api'
import { timeAgo, fullTimestamp } from '@/lib/dateUtils'
import { Plus, RefreshCw, Trash2, Loader2, Sparkles, Eye, Type, Gamepad2, PlayCircle, Copy, Check } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { DataTable, DataTableColumnHeader } from '@/components/ui/data-table'

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

// Clipboard
const { copy, copied } = useClipboard({ copiedDuring: 2000 })

// DataTable columns
const analysisSorting = ref([])
const columnHelper = createColumnHelper()

const analysisColumns = [
  columnHelper.accessor('gameName', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Game' }),
    cell: (info) => {
      const item = info.row.original
      return h('div', { class: 'min-w-0' }, [
        h('p', { class: 'text-sm font-medium truncate' }, item.gameName || 'Untitled'),
        item.gameUrl
          ? h(Tooltip, null, {
              default: () => h(TooltipTrigger, { asChild: true }, () =>
                h('p', { class: 'text-xs text-muted-foreground truncate cursor-default' }, item.gameUrl)
              ),
              content: () => h(TooltipContent, { class: 'max-w-[400px]', side: 'bottom', onClick: (e) => e.stopPropagation() }, () =>
                h('div', { class: 'flex flex-col gap-1' }, [
                  h('p', { class: 'text-xs break-all select-all' }, item.gameUrl),
                  h(Button, {
                    variant: 'ghost',
                    size: 'sm',
                    class: 'h-6 text-xs w-full justify-start',
                    onClick: (e) => { e.stopPropagation(); copy(item.gameUrl) },
                  }, () => [
                    h(copied.value ? Check : Copy, { class: 'h-3 w-3 mr-1' }),
                    copied.value ? 'Copied!' : 'Copy URL',
                  ]),
                ])
              ),
            })
          : null,
      ])
    },
    meta: { class: 'max-w-[250px]' },
  }),
  columnHelper.accessor('status', {
    header: 'Status',
    cell: (info) => {
      const status = info.getValue()
      return h(Badge, { variant: statusVariant(status), class: 'text-xs' }, () => [
        status === 'running' ? h(Loader2, { class: 'h-3 w-3 mr-1 animate-spin' }) : null,
        status,
      ])
    },
    enableSorting: false,
  }),
  columnHelper.accessor('framework', {
    header: 'Framework',
    cell: (info) => h('span', { class: 'text-sm text-muted-foreground capitalize' }, info.getValue() || '-'),
    meta: { class: 'hidden sm:table-cell' },
  }),
  columnHelper.accessor('flowCount', {
    header: 'Flows',
    cell: (info) => h('span', { class: 'text-sm text-muted-foreground' }, info.getValue() ? `${info.getValue()}` : '-'),
    meta: { class: 'hidden md:table-cell' },
  }),
  columnHelper.accessor('createdAt', {
    header: ({ column }) => h(DataTableColumnHeader, { column, title: 'Created' }),
    cell: (info) => h(Tooltip, null, {
      default: () => h(TooltipTrigger, { asChild: true }, () =>
        h('span', { class: 'text-xs text-muted-foreground cursor-default whitespace-nowrap' }, timeAgo(info.getValue()))
      ),
      content: () => h(TooltipContent, null, () => h('p', null, fullTimestamp(info.getValue()))),
    }),
    meta: { class: 'hidden lg:table-cell' },
  }),
  columnHelper.display({
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Actions'),
    cell: ({ row }) => h('div', { class: 'flex items-center gap-1', onClick: (e) => e.stopPropagation() }, [
      h(Button, { variant: 'ghost', size: 'sm', onClick: () => reAnalyze(row.original) },
        () => h(RefreshCw, { class: 'h-3 w-3' })),
      h(Button, { variant: 'ghost', size: 'sm', onClick: () => deleteAnalysis(row.original) },
        () => h(Trash2, { class: 'h-3 w-3 text-destructive' })),
    ]),
    enableSorting: false,
  }),
]

function onAnalysisRowClick(row) {
  viewAnalysis(row.original)
}

// Auto-refresh when any analysis is running
const hasRunning = computed(() => analyses.value.some(a => a.status === 'running'))

async function refreshAnalyses() {
  if (!hasRunning.value) return
  try {
    const data = projectId.value
      ? await projectsApi.analyses(projectId.value)
      : await analysesApi.list()
    analyses.value = data.analyses || []
  } catch {
    // silent
  }
}

useIntervalFn(refreshAnalyses, 5000)

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
