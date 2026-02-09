<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Reports</h2>
        <p class="text-muted-foreground">Test reports and documentation</p>
      </div>
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

    <template v-else>
      <!-- Filter Tabs -->
      <Tabs v-model="activeTab" class="mb-6">
        <TabsList>
          <TabsTrigger value="all">All</TabsTrigger>
          <TabsTrigger value="markdown">Markdown</TabsTrigger>
          <TabsTrigger value="json">JSON</TabsTrigger>
          <TabsTrigger value="junit">JUnit</TabsTrigger>
        </TabsList>
      </Tabs>

      <!-- Report Cards -->
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card
          v-for="report in filteredReports"
          :key="report.id"
          class="hover:shadow-md transition-shadow"
        >
          <CardHeader class="pb-3">
            <div class="flex items-center gap-3">
              <div class="rounded-md bg-primary/10 p-2">
                <FileText v-if="report.format === 'markdown'" class="h-4 w-4 text-primary" />
                <FileJson v-else-if="report.format === 'json'" class="h-4 w-4 text-primary" />
                <FileCode v-else class="h-4 w-4 text-primary" />
              </div>
              <div class="flex-1 min-w-0">
                <CardTitle class="text-sm truncate">{{ report.name }}</CardTitle>
                <p class="text-xs text-muted-foreground mt-1">
                  {{ report.format }} &middot; {{ report.size }}
                </p>
              </div>
            </div>
          </CardHeader>
          <CardContent class="pt-0">
            <div class="flex items-center justify-between">
              <span class="text-xs text-muted-foreground">
                {{ formatDate(report.timestamp) }}
              </span>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" @click="viewReport(report)">View</Button>
                <Button variant="ghost" size="sm" @click="downloadReport(report)">
                  <Download class="h-3 w-3" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div v-if="!filteredReports.length" class="text-center py-12 text-muted-foreground">
        No reports found for this filter
      </div>
    </template>

    <!-- Report Viewer Dialog -->
    <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>{{ selectedReport?.name }}</DialogTitle>
          <DialogDescription>{{ selectedReport?.format }} report</DialogDescription>
        </DialogHeader>
        <div class="mt-4">
          <pre class="bg-muted rounded-md p-4 text-sm overflow-auto max-h-[60vh]"><code>{{ reportContent }}</code></pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { FileText, FileJson, FileCode, Download, AlertCircle } from 'lucide-vue-next'
import { reportsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { downloadBlob } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'

const loading = ref(true)
const error = ref(null)
const reports = ref([])
const activeTab = ref('all')
const dialogOpen = ref(false)
const selectedReport = ref(null)
const reportContent = ref('')

const filteredReports = computed(() => {
  if (activeTab.value === 'all') return reports.value
  return reports.value.filter((r) => r.format === activeTab.value)
})

async function viewReport(report) {
  selectedReport.value = report
  dialogOpen.value = true
  try {
    const data = await reportsApi.get(report.id)
    reportContent.value = data.content || 'No content available'
  } catch {
    reportContent.value = 'Failed to load report content'
  }
}

async function downloadReport(report) {
  try {
    const data = await reportsApi.get(report.id)
    downloadBlob(data.content || '', report.id)
  } catch (err) {
    console.error('Failed to download report:', err)
  }
}

onMounted(async () => {
  try {
    const data = await reportsApi.list()
    reports.value = data.reports || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>
