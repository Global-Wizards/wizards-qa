<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">New Test Plan</h2>
        <p class="text-muted-foreground">Create a reusable test plan with selected flows and variables</p>
      </div>
    </div>

    <!-- Step Tabs -->
    <Tabs v-model="currentStep" class="space-y-6">
      <TabsList class="grid w-full grid-cols-4">
        <TabsTrigger value="details" :disabled="false">1. Details</TabsTrigger>
        <TabsTrigger value="flows" :disabled="!detailsMeta.valid">2. Flows</TabsTrigger>
        <TabsTrigger value="variables" :disabled="!flowsValid">3. Variables</TabsTrigger>
        <TabsTrigger value="review" :disabled="!variablesValid">4. Review</TabsTrigger>
      </TabsList>

      <!-- Step 1: Plan Details -->
      <TabsContent value="details">
        <Card>
          <CardHeader>
            <CardTitle>Plan Details</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <FormField name="name" v-slot="{ componentField }">
              <FormItem>
                <FormLabel>Plan Name *</FormLabel>
                <FormControl>
                  <Input v-bind="componentField" placeholder="e.g. Smoke Test - Game Mechanics" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField name="gameUrl" v-slot="{ componentField }">
              <FormItem>
                <FormLabel>Game URL *</FormLabel>
                <FormControl>
                  <Input v-bind="componentField" placeholder="https://your-game.example.com" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField name="description" v-slot="{ componentField }">
              <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                  <Textarea
                    v-bind="componentField"
                    rows="3"
                    placeholder="Optional description of this test plan..."
                  />
                </FormControl>
              </FormItem>
            </FormField>

            <div class="flex justify-end">
              <Button :disabled="!detailsMeta.valid" @click="currentStep = 'flows'">
                Next: Select Flows
              </Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Step 2: Select Flows -->
      <TabsContent value="flows">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle>Select Flows</CardTitle>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" @click="selectAll">Select All</Button>
                <Button variant="outline" size="sm" @click="deselectAll">Deselect All</Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <!-- Loading -->
            <div v-if="templatesLoading" class="text-center py-8 text-muted-foreground">
              Loading templates...
            </div>

            <!-- Error -->
            <div v-else-if="templatesError" class="text-center py-8 text-destructive">
              {{ templatesError }}
            </div>

            <!-- Templates grouped by category -->
            <div v-else-if="!templatesError" class="space-y-6">
              <div v-for="(group, category) in groupedTemplates" :key="category">
                <h4 class="text-sm font-medium mb-3 capitalize">{{ category }}</h4>
                <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                  <Card
                    v-for="tmpl in group"
                    :key="tmpl.name"
                    class="cursor-pointer transition-all"
                    :class="selectedFlows.has(tmpl.name)
                      ? 'ring-2 ring-primary bg-primary/5'
                      : 'hover:shadow-sm'"
                    @click="toggleFlow(tmpl.name)"
                  >
                    <CardContent class="pt-4 pb-3">
                      <div class="flex items-start justify-between">
                        <div>
                          <p class="text-sm font-medium">{{ tmpl.name }}</p>
                          <p class="text-xs text-muted-foreground mt-1">
                            {{ tmpl.variables?.length || 0 }} variable{{ tmpl.variables?.length === 1 ? '' : 's' }}
                          </p>
                        </div>
                        <div
                          class="h-5 w-5 rounded border flex items-center justify-center shrink-0"
                          :class="selectedFlows.has(tmpl.name)
                            ? 'bg-primary border-primary text-primary-foreground'
                            : 'border-input'"
                        >
                          <svg v-if="selectedFlows.has(tmpl.name)" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                            <polyline points="20 6 9 17 4 12" />
                          </svg>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>

              <div v-if="!templates.length" class="text-center py-8 text-muted-foreground">
                No flow templates found
              </div>
            </div>

            <Separator class="my-4" />
            <div class="flex items-center justify-between">
              <span class="text-sm text-muted-foreground">{{ selectedFlows.size }} flow(s) selected</span>
              <div class="flex gap-2">
                <Button variant="outline" @click="currentStep = 'details'">Back</Button>
                <Button :disabled="!flowsValid" @click="currentStep = 'variables'">
                  Next: Configure Variables
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Step 3: Configure Variables -->
      <TabsContent value="variables">
        <Card>
          <CardHeader>
            <CardTitle>Configure Variables</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <div v-if="!uniqueVariables.length" class="text-center py-8 text-muted-foreground">
              Selected flows have no configurable variables
            </div>

            <div v-for="varName in uniqueVariables" :key="varName" class="space-y-2">
              <label class="text-sm font-medium">{{ varName }}</label>
              <Input
                :model-value="variables[varName] || ''"
                @update:model-value="variables[varName] = $event"
                :placeholder="`Value for {{${varName}}}`"
              />
            </div>

            <Separator />
            <div class="flex justify-between">
              <Button variant="outline" @click="currentStep = 'flows'">Back</Button>
              <Button @click="currentStep = 'review'">Next: Review</Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Step 4: Review & Create -->
      <TabsContent value="review">
        <Card>
          <CardHeader>
            <CardTitle>Review Test Plan</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="grid gap-4 md:grid-cols-2">
              <div>
                <span class="text-sm text-muted-foreground">Name</span>
                <p class="font-medium">{{ detailsValues.name }}</p>
              </div>
              <div>
                <span class="text-sm text-muted-foreground">Game URL</span>
                <p class="font-medium">{{ detailsValues.gameUrl }}</p>
              </div>
            </div>

            <div v-if="detailsValues.description">
              <span class="text-sm text-muted-foreground">Description</span>
              <p class="text-sm">{{ detailsValues.description }}</p>
            </div>

            <Separator />

            <div>
              <span class="text-sm text-muted-foreground">Selected Flows ({{ selectedFlows.size }})</span>
              <div class="flex flex-wrap gap-2 mt-2">
                <Badge v-for="name in [...selectedFlows]" :key="name" variant="secondary">
                  {{ name }}
                </Badge>
              </div>
            </div>

            <div v-if="Object.keys(variables).length">
              <Separator />
              <span class="text-sm text-muted-foreground">Variables</span>
              <div class="mt-2 space-y-1">
                <div v-for="(val, key) in variables" :key="key" class="flex gap-2 text-sm">
                  <span class="text-muted-foreground">{{ key }}:</span>
                  <span class="font-medium">{{ val || '(empty)' }}</span>
                </div>
              </div>
            </div>

            <Separator />
            <div class="flex justify-between">
              <Button variant="outline" @click="currentStep = 'variables'">Back</Button>
              <Button :disabled="creating" @click="createPlan">
                {{ creating ? 'Creating...' : 'Create Test Plan' }}
              </Button>
            </div>

            <div v-if="createError" class="text-sm text-red-500">{{ createError }}</div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useProjectPath } from '@/composables/useProjectPath'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { templatesApi, testPlansApi, analysesApi } from '@/lib/api'
import { testPlanDetailsSchema } from '@/lib/formSchemas'
import { useProject } from '@/composables/useProject'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'

const router = useRouter()
const route = useRoute()
const { currentProject } = useProject()
const { projectId, basePath } = useProjectPath()
const currentStep = ref('details')
const templatesLoading = ref(true)
const templatesError = ref(null)
const templates = ref([])
const selectedFlows = ref(new Set())
const creating = ref(false)
const createError = ref(null)
const analysisId = ref(route.query.analysisId || '')
const variables = reactive({})

const { meta: detailsMeta, values: detailsValues, setValues } = useForm({
  validationSchema: toTypedSchema(testPlanDetailsSchema),
  initialValues: {
    name: '',
    gameUrl: '',
    description: '',
  },
})

const flowsValid = computed(() => selectedFlows.value.size > 0)
const variablesValid = computed(() => flowsValid.value)

const groupedTemplates = computed(() => {
  const groups = {}
  for (const tmpl of templates.value) {
    const cat = tmpl.category || 'general'
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(tmpl)
  }
  return groups
})

const uniqueVariables = computed(() => {
  const vars = new Set()
  for (const tmpl of templates.value) {
    if (selectedFlows.value.has(tmpl.name)) {
      for (const v of tmpl.variables || []) {
        vars.add(v)
      }
    }
  }
  return [...vars].sort()
})

function toggleFlow(name) {
  const next = new Set(selectedFlows.value)
  if (next.has(name)) {
    next.delete(name)
  } else {
    next.add(name)
  }
  selectedFlows.value = next
}

function selectAll() {
  selectedFlows.value = new Set(templates.value.map((t) => t.name))
}

function deselectAll() {
  selectedFlows.value = new Set()
}

async function createPlan() {
  creating.value = true
  createError.value = null

  // Pre-fill url variable with gameUrl if applicable
  if (uniqueVariables.value.includes('GAME_URL') && !variables.GAME_URL) {
    variables.GAME_URL = detailsValues.gameUrl
  }

  try {
    await testPlansApi.create({
      name: detailsValues.name,
      gameUrl: detailsValues.gameUrl,
      description: detailsValues.description,
      flowNames: [...selectedFlows.value],
      variables: { ...variables },
      projectId: projectId.value,
      analysisId: analysisId.value || undefined,
    })
    router.push(`${basePath.value}/tests`)
  } catch (err) {
    createError.value = err.message
  } finally {
    creating.value = false
  }
}

onMounted(async () => {
  try {
    const data = await templatesApi.list()
    templates.value = data.templates || []
  } catch (err) {
    templatesError.value = err.message || 'Failed to load templates'
  } finally {
    templatesLoading.value = false
  }

  // Pre-select flows from analysis (filename-based names that match ListTemplates)
  if (route.query.analysisId) {
    try {
      const data = await analysesApi.flows(route.query.analysisId)
      const flowNames = data.flowNames || []
      if (flowNames.length) {
        selectedFlows.value = new Set(flowNames)
      }
    } catch {
      // Fall back to query.flows if analysis flow lookup fails
    }
  }
  // Backward compat: pre-select from comma-separated flow names
  if (!selectedFlows.value.size && route.query.flows) {
    const names = route.query.flows.split(',')
    selectedFlows.value = new Set(names)
  }

  // Pre-fill game URL
  const initialGameUrl = route.query.gameUrl || currentProject.value?.gameUrl || ''
  if (initialGameUrl) {
    setValues({ gameUrl: initialGameUrl })
  }
})
</script>
