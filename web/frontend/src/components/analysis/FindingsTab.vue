<template>
  <div class="space-y-4">
    <!-- Summary bar -->
    <div v-if="findings?.length" class="flex flex-wrap items-center gap-2">
      <Badge v-if="counts.critical" variant="destructive">Critical: {{ counts.critical }}</Badge>
      <Badge v-if="counts.major" variant="default">Major: {{ counts.major }}</Badge>
      <Badge v-if="counts.minor" variant="secondary">Minor: {{ counts.minor }}</Badge>
      <Badge v-if="counts.suggestion" variant="secondary">Suggestion: {{ counts.suggestion }}</Badge>
      <Badge v-if="counts.positive" variant="outline">Positive: {{ counts.positive }}</Badge>
    </div>

    <!-- Filter controls -->
    <div v-if="findings?.length" class="flex flex-wrap items-center gap-3">
      <div class="flex items-center gap-1">
        <Button
          v-for="sev in severityOptions"
          :key="sev.value"
          size="sm"
          :variant="activeSeverity === sev.value ? 'default' : 'outline'"
          class="h-7 text-xs"
          @click="activeSeverity = sev.value"
        >
          {{ sev.label }}
        </Button>
      </div>
      <Select v-if="categories.length > 1" :model-value="activeCategory" @update:model-value="activeCategory = $event">
        <SelectTrigger class="w-[180px] h-8 text-xs">
          <SelectValue placeholder="All categories" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All categories</SelectItem>
          <SelectItem v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Finding cards -->
    <div v-if="filteredFindings.length" class="space-y-3">
      <div v-for="(finding, i) in filteredFindings" :key="i" class="rounded-md border p-4 space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <Badge :variant="severityVariant(finding.severity)">{{ finding.severity }}</Badge>
          <Badge v-if="finding.category" variant="outline">{{ finding.category }}</Badge>
          <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
        </div>

        <!-- Wording: show the text -->
        <div v-if="type === 'wording' && finding.text" class="font-mono text-xs bg-muted px-3 py-2 rounded">
          "{{ finding.text }}"
        </div>

        <p class="text-sm">{{ finding.description }}</p>

        <!-- Game design: show impact -->
        <p v-if="type === 'gamedesign' && finding.impact" class="text-sm">
          <span class="text-muted-foreground font-medium">Impact:</span> {{ finding.impact }}
        </p>

        <p v-if="finding.suggestion" class="text-sm text-muted-foreground">
          Suggestion: {{ finding.suggestion }}
        </p>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="text-center py-12 text-muted-foreground">
      <p v-if="findings?.length">No findings match the current filters.</p>
      <p v-else>No findings available.</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { severityVariant } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'

const props = defineProps({
  findings: { type: Array, default: () => [] },
  type: { type: String, default: 'uiux' }, // 'uiux' | 'wording' | 'gamedesign'
})

const activeSeverity = ref('all')
const activeCategory = ref('all')

const severityOptions = computed(() => {
  const opts = [{ value: 'all', label: 'All' }]
  if (counts.value.critical) opts.push({ value: 'critical', label: 'Critical' })
  if (counts.value.major) opts.push({ value: 'major', label: 'Major' })
  if (counts.value.minor) opts.push({ value: 'minor', label: 'Minor' })
  if (props.type === 'gamedesign') {
    if (counts.value.positive) opts.push({ value: 'positive', label: 'Positive' })
  } else {
    if (counts.value.suggestion) opts.push({ value: 'suggestion', label: 'Suggestion' })
  }
  return opts
})

const counts = computed(() => {
  const c = { critical: 0, major: 0, minor: 0, suggestion: 0, positive: 0 }
  for (const f of props.findings || []) {
    if (f.severity in c) c[f.severity]++
  }
  return c
})

const categories = computed(() => {
  const set = new Set()
  for (const f of props.findings || []) {
    if (f.category) set.add(f.category)
  }
  return [...set].sort()
})

const filteredFindings = computed(() => {
  let result = props.findings || []
  if (activeSeverity.value !== 'all') {
    result = result.filter(f => f.severity === activeSeverity.value)
  }
  if (activeCategory.value !== 'all') {
    result = result.filter(f => f.category === activeCategory.value)
  }
  return result
})
</script>
