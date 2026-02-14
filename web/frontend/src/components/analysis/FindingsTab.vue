<template>
  <div class="space-y-4">
    <!-- Filter bar -->
    <div v-if="normalizedFindings.length" class="flex flex-wrap items-center gap-2">
      <!-- Tier pills -->
      <button
        :class="[
          'inline-flex items-center rounded-full px-3 py-1 text-xs font-medium transition-colors',
          activeTier === 'all'
            ? 'bg-primary text-primary-foreground'
            : 'bg-muted text-muted-foreground hover:bg-muted/80'
        ]"
        @click="activeTier = 'all'"
      >
        All
      </button>
      <button
        :class="[
          'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors',
          activeTier === 'positive'
            ? 'bg-green-600 text-white'
            : 'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900/40 dark:text-green-300 dark:hover:bg-green-900/60'
        ]"
        @click="activeTier = activeTier === 'positive' ? 'all' : 'positive'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
        {{ tierCounts.positive }} Positive
      </button>
      <button
        :class="[
          'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors',
          activeTier === 'suggestion'
            ? 'bg-amber-600 text-white'
            : 'bg-amber-100 text-amber-800 hover:bg-amber-200 dark:bg-amber-900/40 dark:text-amber-300 dark:hover:bg-amber-900/60'
        ]"
        @click="activeTier = activeTier === 'suggestion' ? 'all' : 'suggestion'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
        {{ tierCounts.suggestion }} Suggestions
      </button>
      <button
        :class="[
          'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors',
          activeTier === 'bug'
            ? 'bg-red-600 text-white'
            : 'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900/40 dark:text-red-300 dark:hover:bg-red-900/60'
        ]"
        @click="activeTier = activeTier === 'bug' ? 'all' : 'bug'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" /></svg>
        {{ tierCounts.bug }} Bugs
      </button>

      <!-- Separator -->
      <div class="h-5 w-px bg-border" />

      <!-- Category dropdown -->
      <Select v-if="categories.length > 1" :model-value="activeCategory" @update:model-value="activeCategory = $event">
        <SelectTrigger class="w-[180px] h-8 text-xs">
          <SelectValue placeholder="All categories" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All categories</SelectItem>
          <SelectItem v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</SelectItem>
        </SelectContent>
      </Select>

      <!-- Text search -->
      <Input
        v-model="searchQuery"
        placeholder="Search findings..."
        class="h-8 w-[200px] text-xs"
      />

      <!-- Active filter count -->
      <span v-if="activeFilterCount > 0" class="text-xs text-muted-foreground">
        {{ activeFilterCount }} filter{{ activeFilterCount > 1 ? 's' : '' }} active
      </span>
    </div>

    <!-- Grouped findings sections -->
    <div v-if="filteredFindings.length" class="space-y-6">
      <!-- Section 1: What's Working Well (green) -->
      <div v-if="groupedFindings.positive.length" class="rounded-lg border-l-4 border-green-500 bg-green-500/[0.06] p-4 space-y-3">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold text-green-700 dark:text-green-300">What's Working Well</h3>
          <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-700 dark:text-green-300">
            {{ groupedFindings.positive.length }}
          </span>
        </div>
        <div class="space-y-3">
          <div v-for="(finding, i) in groupedFindings.positive" :key="'p-' + i" class="rounded-md border border-green-500/20 bg-card p-4 space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-700 dark:text-green-300">positive</span>
              <Badge v-if="finding.category" variant="outline">{{ finding.category }}</Badge>
              <span v-if="type === 'gli' && finding.status" :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', gliStatusClass(finding.status)]">{{ finding.status?.replace(/_/g, ' ') }}</span>
              <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
            </div>
            <div v-if="type === 'wording' && finding.text" class="font-mono text-xs bg-muted px-3 py-2 rounded">"{{ finding.text }}"</div>
            <p class="text-sm">{{ finding.description }}</p>
            <p v-if="type === 'gamedesign' && finding.impact" class="text-sm">
              <span class="text-muted-foreground font-medium">Impact:</span> {{ finding.impact }}
            </p>
            <div v-if="type === 'gli' && finding.jurisdictions?.length" class="flex flex-wrap gap-1">
              <span v-for="j in finding.jurisdictions" :key="j" class="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-700 dark:text-blue-300">{{ j }}</span>
            </div>
            <p v-if="type === 'gli' && finding.gliReference" class="text-xs text-muted-foreground">Ref: {{ finding.gliReference }}</p>
            <p v-if="finding.suggestion" class="text-sm text-muted-foreground italic">Suggestion: {{ finding.suggestion }}</p>
          </div>
        </div>
      </div>

      <!-- Section 2: Suggestions (amber) -->
      <div v-if="groupedFindings.suggestion.length" class="rounded-lg border-l-4 border-amber-500 bg-amber-500/[0.06] p-4 space-y-3">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold text-amber-700 dark:text-amber-300">Suggestions</h3>
          <span class="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300">
            {{ groupedFindings.suggestion.length }}
          </span>
        </div>
        <div class="space-y-3">
          <div v-for="(finding, i) in groupedFindings.suggestion" :key="'s-' + i" class="rounded-md border border-amber-500/20 bg-card p-4 space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300">{{ finding.severity }}</span>
              <Badge v-if="finding.category" variant="outline">{{ finding.category }}</Badge>
              <span v-if="type === 'gli' && finding.status" :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', gliStatusClass(finding.status)]">{{ finding.status?.replace(/_/g, ' ') }}</span>
              <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
            </div>
            <div v-if="type === 'wording' && finding.text" class="font-mono text-xs bg-muted px-3 py-2 rounded">"{{ finding.text }}"</div>
            <p class="text-sm">{{ finding.description }}</p>
            <p v-if="type === 'gamedesign' && finding.impact" class="text-sm">
              <span class="text-muted-foreground font-medium">Impact:</span> {{ finding.impact }}
            </p>
            <div v-if="type === 'gli' && finding.jurisdictions?.length" class="flex flex-wrap gap-1">
              <span v-for="j in finding.jurisdictions" :key="j" class="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-700 dark:text-blue-300">{{ j }}</span>
            </div>
            <p v-if="type === 'gli' && finding.gliReference" class="text-xs text-muted-foreground">Ref: {{ finding.gliReference }}</p>
            <p v-if="finding.suggestion" class="text-sm text-muted-foreground italic">Suggestion: {{ finding.suggestion }}</p>
          </div>
        </div>
      </div>

      <!-- Section 3: Bugs & Issues (red) -->
      <div v-if="groupedFindings.bug.length" class="rounded-lg border-l-4 border-red-500 bg-red-500/[0.06] p-4 space-y-3">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold text-red-700 dark:text-red-300">Bugs & Issues</h3>
          <span class="inline-flex items-center rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-700 dark:text-red-300">
            {{ groupedFindings.bug.length }}
          </span>
        </div>
        <div class="space-y-3">
          <div
            v-for="(finding, i) in groupedFindings.bug"
            :key="'b-' + i"
            :class="[
              'rounded-md border p-4 space-y-2',
              finding.severity === 'critical'
                ? 'border-red-500/30 bg-red-500/10'
                : 'border-red-500/20 bg-card'
            ]"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span v-if="finding.severity === 'critical'" class="inline-flex items-center rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-700 dark:text-red-300">critical</span>
              <span v-else class="inline-flex items-center rounded-full bg-orange-500/10 px-2 py-0.5 text-xs font-medium text-orange-700 dark:text-orange-300">major</span>
              <Badge v-if="finding.category" variant="outline">{{ finding.category }}</Badge>
              <span v-if="type === 'gli' && finding.status" :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', gliStatusClass(finding.status)]">{{ finding.status?.replace(/_/g, ' ') }}</span>
              <span v-if="finding.location" class="text-xs text-muted-foreground">{{ finding.location }}</span>
            </div>
            <div v-if="type === 'wording' && finding.text" class="font-mono text-xs bg-muted px-3 py-2 rounded">"{{ finding.text }}"</div>
            <p class="text-sm">{{ finding.description }}</p>
            <p v-if="type === 'gamedesign' && finding.impact" class="text-sm">
              <span class="text-muted-foreground font-medium">Impact:</span> {{ finding.impact }}
            </p>
            <div v-if="type === 'gli' && finding.jurisdictions?.length" class="flex flex-wrap gap-1">
              <span v-for="j in finding.jurisdictions" :key="j" class="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-700 dark:text-blue-300">{{ j }}</span>
            </div>
            <p v-if="type === 'gli' && finding.gliReference" class="text-xs text-muted-foreground">Ref: {{ finding.gliReference }}</p>
            <p v-if="finding.suggestion" class="text-sm text-muted-foreground italic">Suggestion: {{ finding.suggestion }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="text-center py-12 text-muted-foreground">
      <p v-if="normalizedFindings.length">No findings match the current filters.</p>
      <p v-else>No findings available.</p>
    </div>

    <!-- Role Checklists -->
    <div v-if="normalizedFindings.length" class="mt-6 rounded-lg border">
      <button
        class="flex w-full items-center justify-between px-4 py-3 text-sm font-semibold select-none hover:bg-muted/50 transition-colors"
        @click="checklistOpen = !checklistOpen"
      >
        Action Checklist by Role
        <ChevronDown class="h-4 w-4 text-muted-foreground transition-transform" :class="checklistOpen && 'rotate-180'" />
      </button>
      <div v-if="checklistOpen" class="px-4 pb-4 grid gap-4 sm:grid-cols-2">
        <!-- Developer -->
        <div v-if="developerItems.length" class="rounded-md border p-3 space-y-2">
          <div class="flex items-center gap-2 text-sm font-medium">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 011 1v3a1 1 0 11-2 0V8a1 1 0 011-1zm4 0a1 1 0 011 1v3a1 1 0 11-2 0V8a1 1 0 011-1zm4 0a1 1 0 011 1v3a1 1 0 11-2 0V8a1 1 0 011-1z" clip-rule="evenodd" /></svg>
            Developer
          </div>
          <label v-for="(item, i) in developerItems" :key="'dev-' + i" class="flex items-start gap-2 text-xs">
            <input type="checkbox" v-model="checklistState['dev-' + i]" class="mt-0.5 rounded" />
            <span>{{ item }}</span>
          </label>
        </div>

        <!-- QA Engineer -->
        <div v-if="qaItems.length" class="rounded-md border p-3 space-y-2">
          <div class="flex items-center gap-2 text-sm font-medium">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7 2a1 1 0 00-.707 1.707L7 4.414v3.758a1 1 0 01-.293.707l-4 4C1.077 14.509 2.156 17 4.414 17h11.172c2.258 0 3.337-2.49 1.707-4.121l-4-4A1 1 0 0113 8.172V4.414l.707-.707A1 1 0 0013 2H7zm2 6.172V4h2v4.172a3 3 0 00.879 2.12l1.027 1.028a4 4 0 00-2.171.102l-.47.156a4 4 0 01-2.53 0l-.563-.187 1.116-1.116A3 3 0 009 8.172z" clip-rule="evenodd" /></svg>
            QA Engineer
          </div>
          <label v-for="(item, i) in qaItems" :key="'qa-' + i" class="flex items-start gap-2 text-xs">
            <input type="checkbox" v-model="checklistState['qa-' + i]" class="mt-0.5 rounded" />
            <span>{{ item }}</span>
          </label>
        </div>

        <!-- Designer -->
        <div v-if="designerItems.length" class="rounded-md border p-3 space-y-2">
          <div class="flex items-center gap-2 text-sm font-medium">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4 2a2 2 0 00-2 2v11a3 3 0 106 0V4a2 2 0 00-2-2H4zm1 14a1 1 0 100-2 1 1 0 000 2zm5-1.757l4.9-4.9a2 2 0 000-2.828L13.485 5.1a2 2 0 00-2.828 0L10 5.757v8.486zM16 18H9.071l6-6H16a2 2 0 012 2v2a2 2 0 01-2 2z" clip-rule="evenodd" /></svg>
            Designer
          </div>
          <label v-for="(item, i) in designerItems" :key="'des-' + i" class="flex items-start gap-2 text-xs">
            <input type="checkbox" v-model="checklistState['des-' + i]" class="mt-0.5 rounded" />
            <span>{{ item }}</span>
          </label>
        </div>

        <!-- Product Manager -->
        <div v-if="pmItems.length" class="rounded-md border p-3 space-y-2">
          <div class="flex items-center gap-2 text-sm font-medium">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor"><path d="M9 2a1 1 0 000 2h2a1 1 0 100-2H9z" /><path fill-rule="evenodd" d="M4 5a2 2 0 012-2 3 3 0 003 3h2a3 3 0 003-3 2 2 0 012 2v11a2 2 0 01-2 2H6a2 2 0 01-2-2V5zm3 4a1 1 0 000 2h.01a1 1 0 100-2H7zm3 0a1 1 0 000 2h3a1 1 0 100-2h-3zm-3 4a1 1 0 100 2h.01a1 1 0 100-2H7zm3 0a1 1 0 100 2h3a1 1 0 100-2h-3z" clip-rule="evenodd" /></svg>
            Product Manager
          </div>
          <label v-for="(item, i) in pmItems" :key="'pm-' + i" class="flex items-start gap-2 text-xs">
            <input type="checkbox" v-model="checklistState['pm-' + i]" class="mt-0.5 rounded" />
            <span>{{ item }}</span>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive, watch } from 'vue'
import { findingTier } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { ChevronDown } from 'lucide-vue-next'

const props = defineProps({
  findings: { type: Array, default: () => [] },
  type: { type: String, default: 'uiux' }, // 'uiux' | 'wording' | 'gamedesign' | 'gli'
})

// Normalize GLI findings so complianceCategory maps to category for filters
const normalizedFindings = computed(() => {
  if (props.type !== 'gli') return props.findings || []
  return (props.findings || []).map(f => ({
    ...f,
    category: f.complianceCategory || f.category,
  }))
})

function gliStatusClass(status) {
  switch (status) {
    case 'compliant': return 'bg-green-500/10 text-green-700 dark:text-green-300'
    case 'non_compliant': return 'bg-red-500/10 text-red-700 dark:text-red-300'
    case 'needs_review': return 'bg-amber-500/10 text-amber-700 dark:text-amber-300'
    default: return 'bg-muted text-muted-foreground'
  }
}

const activeTier = ref('all')
const activeCategory = ref('all')
const searchQuery = ref('')
const checklistState = reactive({})
const checklistOpen = ref(false)

function pl(n, s, p) { return n === 1 ? s : (p || s + 's') }

// Debounced search text
let searchTimeout = null
const debouncedSearch = ref('')
watch(searchQuery, (val) => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => { debouncedSearch.value = val }, 200)
})

const counts = computed(() => {
  const c = { critical: 0, major: 0, minor: 0, suggestion: 0, positive: 0, informational: 0 }
  for (const f of normalizedFindings.value) {
    if (f.severity in c) c[f.severity]++
  }
  return c
})

const tierCounts = computed(() => {
  const t = { positive: 0, suggestion: 0, bug: 0 }
  for (const f of normalizedFindings.value) {
    t[findingTier(f.severity)]++
  }
  return t
})

const categories = computed(() => {
  const set = new Set()
  for (const f of normalizedFindings.value) {
    if (f.category) set.add(f.category)
  }
  return [...set].sort()
})

const activeFilterCount = computed(() => {
  let n = 0
  if (activeTier.value !== 'all') n++
  if (activeCategory.value !== 'all') n++
  if (debouncedSearch.value.trim()) n++
  return n
})

const filteredFindings = computed(() => {
  let result = normalizedFindings.value
  if (activeTier.value !== 'all') {
    result = result.filter(f => findingTier(f.severity) === activeTier.value)
  }
  if (activeCategory.value !== 'all') {
    result = result.filter(f => f.category === activeCategory.value)
  }
  const q = debouncedSearch.value.trim().toLowerCase()
  if (q) {
    result = result.filter(f =>
      (f.description || '').toLowerCase().includes(q) ||
      (f.suggestion || '').toLowerCase().includes(q) ||
      (f.location || '').toLowerCase().includes(q)
    )
  }
  return result
})

const groupedFindings = computed(() => {
  const groups = { positive: [], suggestion: [], bug: [] }
  for (const f of filteredFindings.value) {
    groups[findingTier(f.severity)].push(f)
  }
  return groups
})

// Role checklist items — dynamic based on severity counts
const developerItems = computed(() => {
  const items = []
  if (counts.value.critical) items.push(`Fix ${counts.value.critical} critical ${pl(counts.value.critical, 'bug')}`)
  if (counts.value.major) items.push(`Address ${counts.value.major} major ${pl(counts.value.major, 'issue')}`)
  if (counts.value.suggestion) items.push(`Review ${counts.value.suggestion} ${pl(counts.value.suggestion, 'suggestion')} for implementation`)
  if (counts.value.minor) items.push(`Review ${counts.value.minor} minor ${pl(counts.value.minor, 'issue')}`)
  return items
})

const qaItems = computed(() => {
  const items = []
  const bugs = counts.value.critical + counts.value.major
  if (bugs) items.push(`Verify and reproduce ${bugs} reported ${pl(bugs, 'bug')}`)
  if (counts.value.critical) items.push('Create regression tests for critical issues')
  const suggestions = counts.value.suggestion + counts.value.minor
  if (suggestions) items.push(`Validate ${suggestions} ${pl(suggestions, 'suggestion')} against requirements`)
  return items
})

const designerItems = computed(() => {
  const items = []
  if (counts.value.positive) items.push(`Celebrate ${counts.value.positive} positive ${pl(counts.value.positive, 'finding')}`)
  const suggestions = counts.value.suggestion + counts.value.minor
  if (suggestions) items.push(`Review ${suggestions} UI/UX ${pl(suggestions, 'suggestion')}`)
  const bugs = counts.value.critical + counts.value.major
  if (bugs) items.push(`Assess visual impact of ${bugs} bug ${pl(bugs, 'fix', 'fixes')}`)
  return items
})

const pmItems = computed(() => {
  const items = []
  if (counts.value.critical) items.push(`Prioritize ${counts.value.critical} critical ${pl(counts.value.critical, 'bug')} for immediate fix`)
  if (counts.value.major) items.push(`Schedule ${counts.value.major} major ${pl(counts.value.major, 'issue')} in backlog`)
  const suggestions = counts.value.suggestion + counts.value.minor
  if (suggestions) items.push(`Evaluate ${suggestions} ${pl(suggestions, 'suggestion')} for roadmap`)
  if (counts.value.positive) items.push(`Share ${counts.value.positive} positive ${pl(counts.value.positive, 'highlight')} with stakeholders`)
  return items
})
</script>
