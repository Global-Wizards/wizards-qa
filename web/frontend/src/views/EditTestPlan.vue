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
                <Textarea
                  v-model="plan.description"
                  rows="3"
                  placeholder="Optional description of this test plan..."
                />
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
                <!-- Validate All button -->
                <div class="flex items-center justify-between mb-4">
                  <div class="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      :disabled="validatingAll"
                      @click="validateAllFlows"
                    >
                      <ShieldCheck class="h-3.5 w-3.5 mr-1" />
                      {{ validatingAll ? 'Validating...' : 'Validate All Flows' }}
                    </Button>
                    <span v-if="validationSummary" class="text-xs" :class="validationSummary.allValid ? 'text-green-600 dark:text-green-400' : 'text-destructive'">
                      {{ validationSummary.text }}
                    </span>
                  </div>
                </div>

                <Tabs v-model="activeFlowTab" class="space-y-4">
                  <TabsList class="flex-wrap h-auto gap-1">
                    <TabsTrigger
                      v-for="flow in flows"
                      :key="flow.name"
                      :value="flow.name"
                      class="text-xs"
                    >
                      <ShieldCheck
                        v-if="flow.validation && flow.validation.valid && !flow.validation.warnings.length"
                        class="h-3 w-3 mr-1 text-green-600 dark:text-green-400"
                      />
                      <ShieldAlert
                        v-else-if="flow.validation && flow.validation.valid && flow.validation.warnings.length"
                        class="h-3 w-3 mr-1 text-yellow-600 dark:text-yellow-400"
                      />
                      <ShieldX
                        v-else-if="flow.validation && !flow.validation.valid"
                        class="h-3 w-3 mr-1 text-destructive"
                      />
                      {{ flow.name }}
                      <span v-if="dirtyFlows.has(flow.name)" class="ml-1 text-primary">*</span>
                    </TabsTrigger>
                  </TabsList>

                  <TabsContent v-for="flow in flows" :key="flow.name" :value="flow.name">
                    <div v-if="flow.error" class="text-sm text-destructive mb-2">
                      Could not load flow: {{ flow.error }}
                    </div>
                    <template v-else>
                      <!-- Per-flow validate button -->
                      <div class="flex items-center gap-2 mb-2">
                        <Button
                          variant="outline"
                          size="sm"
                          :disabled="flow.validating"
                          @click="validateFlow(flow)"
                        >
                          <ShieldCheck class="h-3.5 w-3.5 mr-1" />
                          {{ flow.validating ? 'Validating...' : 'Validate' }}
                        </Button>
                      </div>

                      <!-- Validation results -->
                      <div v-if="flow.validation" class="mb-2 space-y-1">
                        <!-- Valid -->
                        <div
                          v-if="flow.validation.valid && !flow.validation.errors.length && !flow.validation.warnings.length"
                          class="flex items-start gap-2 rounded-md border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-950/30 px-3 py-2"
                        >
                          <ShieldCheck class="h-4 w-4 mt-0.5 text-green-600 dark:text-green-400 shrink-0" />
                          <span class="text-sm text-green-800 dark:text-green-300">Valid Maestro flow — ready to run</span>
                        </div>

                        <!-- Errors -->
                        <div
                          v-for="(err, i) in flow.validation.errors"
                          :key="'e' + i"
                          class="flex items-start gap-2 rounded-md border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/30 px-3 py-2"
                        >
                          <ShieldX class="h-4 w-4 mt-0.5 text-destructive shrink-0" />
                          <span class="text-sm text-red-800 dark:text-red-300">{{ err }}</span>
                        </div>

                        <!-- Warnings -->
                        <div
                          v-for="(warn, i) in flow.validation.warnings"
                          :key="'w' + i"
                          class="flex items-start gap-2 rounded-md border border-yellow-200 dark:border-yellow-900 bg-yellow-50 dark:bg-yellow-950/30 px-3 py-2"
                        >
                          <ShieldAlert class="h-4 w-4 mt-0.5 text-yellow-600 dark:text-yellow-400 shrink-0" />
                          <span class="text-sm text-yellow-800 dark:text-yellow-300">{{ warn }}</span>
                        </div>
                      </div>

                      <!-- Debug console (shows when validation has errors) -->
                      <div
                        v-if="flow.validation && !flow.validation.valid && flow.validation.debug"
                        class="mb-2 border border-border rounded-md overflow-hidden"
                      >
                        <button
                          class="flex items-center gap-2 w-full px-3 py-2 text-xs font-medium text-muted-foreground bg-muted/50 hover:bg-muted transition-colors cursor-pointer"
                          @click="toggleDebug(flow.name)"
                        >
                          <Bug class="h-3.5 w-3.5" />
                          <component :is="debugOpen.has(flow.name) ? ChevronDown : ChevronRight" class="h-3 w-3" />
                          Debug Console
                          <span class="ml-auto flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              class="h-5 px-1.5 text-xs"
                              @click.stop="copyDebugInfo(flow)"
                            >
                              <component :is="debugCopied ? Check : Copy" class="h-3 w-3 mr-1" />
                              {{ debugCopied ? 'Copied' : 'Copy' }}
                            </Button>
                          </span>
                        </button>
                        <div v-if="debugOpen.has(flow.name)" class="px-3 py-2 text-xs font-mono bg-background space-y-3 max-h-[400px] overflow-auto">
                          <div>
                            <p class="text-muted-foreground font-sans font-medium mb-1">Content Info</p>
                            <p>Length: {{ flow.validation.debug.contentLength }} chars, {{ flow.validation.debug.lineCount }} lines</p>
                            <p>Separator (---): {{ flow.validation.debug.separatorFound ? 'found' : 'NOT found' }}</p>
                          </div>
                          <div v-if="flow.validation.debug.metadataSection">
                            <p class="text-muted-foreground font-sans font-medium mb-1">Metadata Section (above ---)</p>
                            <pre class="whitespace-pre-wrap break-all bg-muted/30 rounded p-2 border border-border">{{ flow.validation.debug.metadataSection }}</pre>
                          </div>
                          <div v-if="flow.validation.debug.commandsSection">
                            <p class="text-muted-foreground font-sans font-medium mb-1">Commands Section (below ---)</p>
                            <pre class="whitespace-pre-wrap break-all bg-muted/30 rounded p-2 border border-border">{{ flow.validation.debug.commandsSection }}</pre>
                          </div>
                          <div v-if="flow.validation.debug.parsedMetadata">
                            <p class="text-muted-foreground font-sans font-medium mb-1">Parsed Metadata (JSON)</p>
                            <pre class="whitespace-pre-wrap break-all bg-muted/30 rounded p-2 border border-border">{{ JSON.stringify(flow.validation.debug.parsedMetadata, null, 2) }}</pre>
                          </div>
                          <div v-if="flow.validation.debug.parsedCommands">
                            <p class="text-muted-foreground font-sans font-medium mb-1">Parsed Commands (JSON)</p>
                            <pre class="whitespace-pre-wrap break-all bg-muted/30 rounded p-2 border border-border">{{ JSON.stringify(flow.validation.debug.parsedCommands, null, 2) }}</pre>
                          </div>
                          <div>
                            <p class="text-muted-foreground font-sans font-medium mb-1">Errors</p>
                            <ul class="list-disc list-inside space-y-0.5">
                              <li v-for="(err, i) in flow.validation.errors" :key="i" class="text-destructive">{{ err }}</li>
                            </ul>
                          </div>
                          <div v-if="flow.validation.warnings?.length">
                            <p class="text-muted-foreground font-sans font-medium mb-1">Warnings</p>
                            <ul class="list-disc list-inside space-y-0.5">
                              <li v-for="(w, i) in flow.validation.warnings" :key="i" class="text-yellow-600 dark:text-yellow-400">{{ w }}</li>
                            </ul>
                          </div>
                          <div>
                            <p class="text-muted-foreground font-sans font-medium mb-1">Raw Content</p>
                            <pre class="whitespace-pre-wrap break-all bg-muted/30 rounded p-2 border border-border max-h-[200px] overflow-auto">{{ flow.validation.debug.rawContent }}</pre>
                          </div>
                        </div>
                      </div>

                      <div class="border rounded-md overflow-hidden">
                        <Codemirror
                          :model-value="flow.content"
                          @update:model-value="onFlowEdit(flow, $event)"
                          :extensions="cmExtensions"
                          :style="{ minHeight: '400px' }"
                        />
                      </div>
                    </template>
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
import { useClipboard } from '@vueuse/core'
import { ArrowLeft, Save, Play, AlertCircle, CheckCircle, Trash2, Plus, ShieldCheck, ShieldAlert, ShieldX, Bug, Copy, Check, ChevronDown, ChevronRight } from 'lucide-vue-next'
import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { oneDark } from '@codemirror/theme-one-dark'
import { testPlansApi, flowsApi } from '@/lib/api'
import { formatDate } from '@/lib/dateUtils'
import { useTheme } from '@/composables/useTheme'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
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

// Validation state
const validatingAll = ref(false)
const validationSummary = ref(null)

// Debug console state
const debugOpen = ref(new Set())
const { copy: copyToClipboard, copied: debugCopied } = useClipboard({ copiedDuring: 2000 })

function toggleDebug(flowName) {
  const next = new Set(debugOpen.value)
  if (next.has(flowName)) next.delete(flowName)
  else next.add(flowName)
  debugOpen.value = next
}

function copyDebugInfo(flow) {
  const d = flow.validation?.debug
  if (!d) return
  const lines = [
    `=== FLOW VALIDATION DEBUG ===`,
    `Flow: ${flow.name}`,
    `Timestamp: ${new Date().toISOString()}`,
    `Content: ${d.contentLength} chars, ${d.lineCount} lines`,
    `Separator (---): ${d.separatorFound ? 'found' : 'NOT found'}`,
    ``,
    `--- ERRORS ---`,
    ...flow.validation.errors.map(e => `  • ${e}`),
  ]
  if (flow.validation.warnings?.length) {
    lines.push(``, `--- WARNINGS ---`, ...flow.validation.warnings.map(w => `  • ${w}`))
  }
  if (d.metadataSection) {
    lines.push(``, `--- METADATA SECTION (above ---) ---`, d.metadataSection)
  }
  if (d.commandsSection) {
    lines.push(``, `--- COMMANDS SECTION (below ---) ---`, d.commandsSection)
  }
  if (d.parsedMetadata) {
    lines.push(``, `--- PARSED METADATA (JSON) ---`, JSON.stringify(d.parsedMetadata, null, 2))
  }
  if (d.parsedCommands) {
    lines.push(``, `--- PARSED COMMANDS (JSON) ---`, JSON.stringify(d.parsedCommands, null, 2))
  }
  lines.push(``, `--- RAW CONTENT ---`, d.rawContent)
  copyToClipboard(lines.join('\n'))
}

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
  flow.validation = null
  validationSummary.value = null
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

async function validateFlow(flow) {
  flow.validating = true
  flow.validation = null
  try {
    const result = await flowsApi.validate(flow.content)
    flow.validation = result
  } catch (err) {
    flow.validation = { valid: false, errors: ['Validation request failed: ' + err.message], warnings: [] }
  } finally {
    flow.validating = false
  }
}

async function validateAllFlows() {
  validatingAll.value = true
  validationSummary.value = null
  let valid = 0
  let invalid = 0
  let warnings = 0

  for (const flow of flows.value) {
    if (flow.error) continue
    await validateFlow(flow)
    if (flow.validation) {
      if (flow.validation.valid && !flow.validation.warnings.length) valid++
      else if (flow.validation.valid && flow.validation.warnings.length) warnings++
      else invalid++
    }
  }

  const total = valid + invalid + warnings
  if (invalid === 0 && warnings === 0) {
    validationSummary.value = { allValid: true, text: `All ${total} flow(s) valid` }
  } else if (invalid === 0) {
    validationSummary.value = { allValid: true, text: `${valid} valid, ${warnings} with warnings` }
  } else {
    validationSummary.value = { allValid: false, text: `${invalid} invalid, ${valid} valid${warnings ? `, ${warnings} with warnings` : ''}` }
  }
  validatingAll.value = false
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
