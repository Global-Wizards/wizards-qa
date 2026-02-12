<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-4">
        <Button variant="ghost" size="sm" @click="goBack">
          <ArrowLeft class="h-4 w-4 mr-1" />
          Back
        </Button>
        <div>
          <h2 class="text-3xl font-bold tracking-tight">Edit Test Plan</h2>
          <p class="text-muted-foreground">{{ plan.name || 'Loading...' }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button :disabled="!hasChanges || saving" @click="save">
          <Save class="h-4 w-4 mr-1" />
          {{ saving ? 'Saving...' : 'Save Changes' }}
        </Button>
        <Button variant="outline" :disabled="saving || running" @click="runPlan">
          <Play class="h-4 w-4 mr-1" />
          {{ running ? 'Starting...' : 'Run' }}
        </Button>
      </div>
    </div>

    <!-- Loading -->
    <Card v-if="loading">
      <CardContent class="pt-6">
        <div class="space-y-4">
          <Skeleton class="h-8 w-64" />
          <Skeleton class="h-4 w-48" />
          <Skeleton class="h-32 w-full" />
        </div>
      </CardContent>
    </Card>

    <!-- Error -->
    <Alert v-else-if="error" variant="destructive" class="mb-6">
      <AlertCircle class="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Editor -->
    <template v-else>
      <!-- Save success/warning -->
      <Alert v-if="saveSuccess" class="mb-4">
        <CheckCircle class="h-4 w-4" />
        <AlertTitle>Saved</AlertTitle>
        <AlertDescription>{{ saveSuccess }}</AlertDescription>
      </Alert>
      <Alert v-if="saveError" variant="destructive" class="mb-4">
        <AlertCircle class="h-4 w-4" />
        <AlertTitle>Save Error</AlertTitle>
        <AlertDescription>{{ saveError }}</AlertDescription>
      </Alert>

      <Tabs v-model="activeTab" class="space-y-4">
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="flows">Flows ({{ flows.length }})</TabsTrigger>
          <TabsTrigger value="variables">Variables</TabsTrigger>
        </TabsList>

        <!-- Details Tab -->
        <TabsContent value="details">
          <Card>
            <CardContent class="pt-6 space-y-4">
              <div class="space-y-2">
                <label class="text-sm font-medium">Plan Name *</label>
                <Input v-model="plan.name" placeholder="e.g. Smoke Test - Game Mechanics" />
              </div>
              <div class="space-y-2">
                <label class="text-sm font-medium">Game URL</label>
                <Input v-model="plan.gameUrl" placeholder="https://your-game.example.com" />
                <p v-if="plan.gameUrl && !isValidUrl(plan.gameUrl)" class="text-xs text-destructive">
                  Enter a valid URL starting with http:// or https://
                </p>
              </div>
              <div class="space-y-2">
                <label class="text-sm font-medium">Description</label>
                <textarea
                  v-model="plan.description"
                  rows="3"
                  placeholder="Optional description of this test plan..."
                  class="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                ></textarea>
              </div>
              <div class="space-y-2">
                <label class="text-sm font-medium">Status</label>
                <Badge variant="secondary">{{ plan.status }}</Badge>
              </div>
              <div v-if="plan.createdAt" class="space-y-2">
                <label class="text-sm font-medium">Created</label>
                <p class="text-sm text-muted-foreground">{{ formatDate(plan.createdAt) }}</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <!-- Flows Tab -->
        <TabsContent value="flows">
          <Card>
            <CardContent class="pt-6">
              <div v-if="!flows.length" class="text-center py-8 text-muted-foreground">
                No flows in this test plan
              </div>

              <template v-else>
                <Tabs v-model="activeFlowTab" class="space-y-4">
                  <TabsList class="flex-wrap h-auto gap-1">
                    <TabsTrigger
                      v-for="flow in flows"
                      :key="flow.name"
                      :value="flow.name"
                      class="text-xs"
                    >
                      {{ flow.name }}
                      <span v-if="dirtyFlows.has(flow.name)" class="ml-1 text-primary">*</span>
                    </TabsTrigger>
                  </TabsList>

                  <TabsContent v-for="flow in flows" :key="flow.name" :value="flow.name">
                    <div v-if="flow.error" class="text-sm text-destructive mb-2">
                      Could not load flow: {{ flow.error }}
                    </div>
                    <div v-else class="border rounded-md overflow-hidden">
                      <Codemirror
                        :model-value="flow.content"
                        @update:model-value="onFlowEdit(flow, $event)"
                        :extensions="cmExtensions"
                        :style="{ minHeight: '400px' }"
                      />
                    </div>
                  </TabsContent>
                </Tabs>
              </template>
            </CardContent>
          </Card>
        </TabsContent>

        <!-- Variables Tab -->
        <TabsContent value="variables">
          <Card>
            <CardContent class="pt-6 space-y-4">
              <div v-if="!variableEntries.length" class="text-center py-8 text-muted-foreground">
                No variables configured
              </div>

              <div v-for="(entry, idx) in variableEntries" :key="idx" class="flex items-center gap-2">
                <Input
                  :model-value="entry.key"
                  @update:model-value="updateVariableKey(idx, $event)"
                  placeholder="Variable name"
                  class="flex-1"
                />
                <Input
                  :model-value="entry.value"
                  @update:model-value="updateVariableValue(idx, $event)"
                  placeholder="Value"
                  class="flex-1"
                />
                <Button variant="ghost" size="sm" @click="removeVariable(idx)">
                  <Trash2 class="h-3 w-3 text-destructive" />
                </Button>
              </div>

              <Button variant="outline" size="sm" @click="addVariable">
                <Plus class="h-3 w-3 mr-1" />
                Add Variable
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Save, Play, AlertCircle, CheckCircle, Trash2, Plus } from 'lucide-vue-next'
import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { oneDark } from '@codemirror/theme-one-dark'
import { testPlansApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { useTheme } from '@/composables/useTheme'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

const route = useRoute()
const router = useRouter()
const { isDark } = useTheme()

const planId = computed(() => route.params.planId)
const projectId = computed(() => route.params.projectId || '')

const loading = ref(true)
const error = ref(null)
const saving = ref(false)
const running = ref(false)
const saveSuccess = ref(null)
const saveError = ref(null)
const activeTab = ref('details')
const activeFlowTab = ref('')

// Plan data
const plan = ref({ name: '', description: '', gameUrl: '', status: '', createdAt: '', flowNames: [], variables: {} })
const flows = ref([])
const dirtyFlows = ref(new Set())

// Snapshot for change detection
let snapshot = null

// Variables as entries for editing
const variableEntries = ref([])

// CodeMirror extensions
const cmExtensions = computed(() => {
  const exts = [yaml()]
  if (isDark.value) exts.push(oneDark)
  return exts
})

// Change detection
const hasChanges = computed(() => {
  if (!snapshot) return false
  if (plan.value.name !== snapshot.name) return true
  if (plan.value.description !== snapshot.description) return true
  if (plan.value.gameUrl !== snapshot.gameUrl) return true
  if (dirtyFlows.value.size > 0) return true
  const currentVars = entriesToMap(variableEntries.value)
  if (JSON.stringify(currentVars) !== JSON.stringify(snapshot.variables)) return true
  return false
})

function isValidUrl(str) {
  try {
    const url = new URL(str)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function entriesToMap(entries) {
  const map = {}
  for (const e of entries) {
    if (e.key.trim()) map[e.key.trim()] = e.value
  }
  return map
}

function mapToEntries(map) {
  return Object.entries(map || {}).map(([key, value]) => ({ key, value }))
}

function onFlowEdit(flow, value) {
  flow.content = value
  const next = new Set(dirtyFlows.value)
  next.add(flow.name)
  dirtyFlows.value = next
}

function addVariable() {
  variableEntries.value.push({ key: '', value: '' })
}

function removeVariable(idx) {
  variableEntries.value.splice(idx, 1)
}

function updateVariableKey(idx, val) {
  variableEntries.value[idx].key = val
}

function updateVariableValue(idx, val) {
  variableEntries.value[idx].value = val
}

function goBack() {
  const base = projectId.value ? `/projects/${projectId.value}` : ''
  router.push(`${base}/tests?tab=plans`)
}

async function save() {
  saving.value = true
  saveSuccess.value = null
  saveError.value = null

  const flowContents = {}
  for (const flow of flows.value) {
    if (dirtyFlows.value.has(flow.name)) {
      flowContents[flow.name] = flow.content
    }
  }

  try {
    const result = await testPlansApi.update(planId.value, {
      name: plan.value.name,
      description: plan.value.description,
      gameUrl: plan.value.gameUrl,
      flowNames: plan.value.flowNames,
      variables: entriesToMap(variableEntries.value),
      flowContents,
    })

    // Update snapshot
    snapshot = {
      name: plan.value.name,
      description: plan.value.description,
      gameUrl: plan.value.gameUrl,
      variables: entriesToMap(variableEntries.value),
    }
    dirtyFlows.value = new Set()

    if (result.flowWarnings?.length) {
      saveSuccess.value = `Plan saved. Flow warnings: ${result.flowWarnings.join(', ')}`
    } else {
      saveSuccess.value = 'Changes saved successfully'
    }
    setTimeout(() => { saveSuccess.value = null }, 3000)
  } catch (err) {
    saveError.value = err.message || 'Failed to save'
  } finally {
    saving.value = false
  }
}

async function runPlan() {
  running.value = true
  try {
    if (hasChanges.value) await save()
    const data = await testPlansApi.run(planId.value)
    const base = projectId.value ? `/projects/${projectId.value}` : ''
    router.push({
      path: `${base}/tests/run/${data.testId}`,
      query: { fresh: '1', planId: planId.value, planName: plan.value.name },
    })
  } catch (err) {
    saveError.value = err.message || 'Failed to run test plan'
  } finally {
    running.value = false
  }
}

onMounted(async () => {
  try {
    const data = await testPlansApi.get(planId.value, { include: 'flows' })
    const p = data.plan
    plan.value = {
      name: p.name,
      description: p.description || '',
      gameUrl: p.gameUrl || '',
      status: p.status,
      createdAt: p.createdAt,
      flowNames: p.flowNames || [],
      variables: p.variables || {},
    }
    flows.value = (data.flows || []).map(f => ({ name: f.name, content: f.content || '', error: f.error || '' }))
    variableEntries.value = mapToEntries(p.variables)

    if (flows.value.length) {
      activeFlowTab.value = flows.value[0].name
    }

    // Create snapshot for change detection
    snapshot = {
      name: plan.value.name,
      description: plan.value.description,
      gameUrl: plan.value.gameUrl,
      variables: { ...(p.variables || {}) },
    }
  } catch (err) {
    error.value = err.message || 'Failed to load test plan'
  } finally {
    loading.value = false
  }
})
</script>
