<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Flow Templates</h2>
        <p class="text-muted-foreground">Reusable test flow configurations</p>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-4 mb-6">
      <Input v-model="search" placeholder="Search flows..." class="max-w-sm" />
      <Select v-model="categoryFilter">
        <SelectTrigger class="w-[180px]">
          <SelectValue placeholder="All categories" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All categories</SelectItem>
          <SelectItem v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Loading State -->
    <template v-if="loading">
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <LoadingSkeleton variant="card" :count="6" />
      </div>
    </template>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive" class="mb-6">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Flow Cards -->
    <template v-else>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card
          v-for="flow in filteredFlows"
          :key="flow.name"
          class="hover:shadow-md transition-shadow"
        >
          <CardHeader class="pb-3">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm">{{ flow.name }}</CardTitle>
              <Badge variant="secondary">{{ flow.category }}</Badge>
            </div>
          </CardHeader>
          <CardContent class="pt-0">
            <p class="text-xs text-muted-foreground mb-3 truncate">{{ flow.path }}</p>
            <Button variant="outline" size="sm" @click="viewFlow(flow)">View YAML</Button>
          </CardContent>
        </Card>
      </div>

      <div v-if="!filteredFlows.length" class="text-center py-12 text-muted-foreground">
        {{ search || categoryFilter !== 'all' ? 'No flows match your filters' : 'No flow templates found' }}
      </div>
    </template>

    <!-- YAML Viewer Dialog -->
    <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>{{ selectedFlow?.name }}</DialogTitle>
          <DialogDescription>{{ selectedFlow?.path }}</DialogDescription>
        </DialogHeader>
        <div class="mt-4 relative">
          <Button
            variant="outline"
            size="sm"
            class="absolute top-2 right-2 z-10"
            @click="copyContent"
          >
            <Copy class="h-3 w-3 mr-1" />
            {{ copied ? 'Copied!' : 'Copy' }}
          </Button>
          <pre class="bg-muted rounded-md p-4 text-sm overflow-auto max-h-[60vh]"><code>{{ flowContent }}</code></pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { AlertCircle, Copy } from 'lucide-vue-next'
import { flowsApi } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'

const loading = ref(true)
const error = ref(null)
const flows = ref([])
const search = ref('')
const categoryFilter = ref('all')
const dialogOpen = ref(false)
const selectedFlow = ref(null)
const flowContent = ref('')
const copied = ref(false)
let copyTimeoutId = null

const categories = computed(() => {
  const cats = new Set(flows.value.map((f) => f.category))
  return [...cats].sort()
})

const filteredFlows = computed(() => {
  let result = [...flows.value]

  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter((f) => f.name.toLowerCase().includes(q) || f.category.toLowerCase().includes(q))
  }

  if (categoryFilter.value !== 'all') {
    result = result.filter((f) => f.category === categoryFilter.value)
  }

  return result
})

async function viewFlow(flow) {
  selectedFlow.value = flow
  dialogOpen.value = true
  copied.value = false
  try {
    const data = await flowsApi.get(flow.name)
    flowContent.value = data.content || 'No content available'
  } catch {
    flowContent.value = 'Failed to load flow content'
  }
}

async function copyContent() {
  try {
    await navigator.clipboard.writeText(flowContent.value)
    copied.value = true
    if (copyTimeoutId != null) clearTimeout(copyTimeoutId)
    copyTimeoutId = setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // clipboard API not available
  }
}

onUnmounted(() => {
  if (copyTimeoutId != null) clearTimeout(copyTimeoutId)
})

onMounted(async () => {
  try {
    const data = await flowsApi.list()
    flows.value = data.flows || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>
