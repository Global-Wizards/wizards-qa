<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">{{ isEdit ? 'Edit Project' : 'New Project' }}</h2>
        <p class="text-muted-foreground">{{ isEdit ? 'Update project details' : 'Create a new project to organize your work' }}</p>
      </div>
    </div>

    <Card class="max-w-2xl">
      <CardContent class="pt-6 space-y-6">
        <!-- Name -->
        <FormField name="name" v-slot="{ componentField }">
          <FormItem>
            <FormLabel>Project Name *</FormLabel>
            <FormControl>
              <Input v-bind="componentField" placeholder="e.g. My Awesome Game" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <!-- URL -->
        <FormField name="gameUrl" v-slot="{ componentField }">
          <FormItem>
            <FormLabel>Game URL</FormLabel>
            <FormControl>
              <Input v-bind="componentField" placeholder="https://your-game.example.com" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <!-- Description (Echo Editor) -->
        <div class="space-y-2">
          <label class="text-sm font-medium">Description</label>
          <echo-editor
            v-model="description"
            :extensions="editorExtensions"
            class="min-h-[120px] rounded-md border border-input"
          />
        </div>

        <!-- Color -->
        <div class="space-y-2">
          <label class="text-sm font-medium">Color</label>
          <div class="flex gap-2">
            <button
              v-for="color in colors"
              :key="color"
              class="h-8 w-8 rounded-full border-2 transition-all"
              :class="form.color === color ? 'border-foreground scale-110' : 'border-transparent'"
              :style="{ backgroundColor: color }"
              @click="form.color = color"
            />
          </div>
        </div>

        <!-- Tags -->
        <div class="space-y-2">
          <label class="text-sm font-medium">Tags</label>
          <div class="flex flex-wrap gap-2 mb-2">
            <Badge v-for="(tag, i) in form.tags" :key="i" variant="secondary" class="gap-1">
              {{ tag }}
              <button class="ml-1 text-xs hover:text-destructive" @click="removeTag(i)">&times;</button>
            </Badge>
          </div>
          <div class="flex gap-2">
            <Input v-model="newTag" placeholder="Add a tag..." @keyup.enter="addTag" class="max-w-xs" />
            <Button variant="outline" size="sm" @click="addTag" :disabled="!newTag.trim()">Add</Button>
          </div>
        </div>

        <Separator />

        <!-- Actions -->
        <div class="flex items-center justify-between">
          <Button variant="outline" @click="router.back()">Cancel</Button>
          <Button :disabled="!meta.valid || saving" @click="handleSave">
            {{ saving ? 'Saving...' : (isEdit ? 'Save Changes' : 'Create Project') }}
          </Button>
        </div>

        <div v-if="saveError" class="text-sm text-destructive">{{ saveError }}</div>
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { projectsApi } from '@/lib/api'
import { projectSchema } from '@/lib/formSchemas'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'

const router = useRouter()
const route = useRoute()

const isEdit = computed(() => !!route.params.projectId)
const saving = ref(false)
const saveError = ref(null)
const newTag = ref('')
const description = ref('')

const editorExtensions = []

const colors = [
  '#6366f1', '#8b5cf6', '#ec4899', '#ef4444', '#f97316',
  '#eab308', '#22c55e', '#06b6d4', '#3b82f6', '#64748b',
]

const form = reactive({
  color: '#6366f1',
  icon: 'gamepad-2',
  tags: [],
})

const { meta, values, setValues } = useForm({
  validationSchema: toTypedSchema(projectSchema),
  initialValues: {
    name: '',
    gameUrl: '',
    description: '',
  },
})

function addTag() {
  const tag = newTag.value.trim()
  if (tag && !form.tags.includes(tag)) {
    form.tags.push(tag)
  }
  newTag.value = ''
}

function removeTag(index) {
  form.tags.splice(index, 1)
}

async function handleSave() {
  saving.value = true
  saveError.value = null

  const payload = {
    name: values.name,
    gameUrl: values.gameUrl,
    description: description.value,
    color: form.color,
    icon: form.icon,
    tags: form.tags,
  }

  try {
    if (isEdit.value) {
      await projectsApi.update(route.params.projectId, payload)
      router.push(`/projects/${route.params.projectId}`)
    } else {
      const created = await projectsApi.create(payload)
      router.push(`/projects/${created.id}`)
    }
  } catch (err) {
    saveError.value = err.message
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (isEdit.value) {
    try {
      const project = await projectsApi.get(route.params.projectId)
      setValues({
        name: project.name || '',
        gameUrl: project.gameUrl || '',
        description: project.description || '',
      })
      description.value = project.description || ''
      form.color = project.color || '#6366f1'
      form.icon = project.icon || 'gamepad-2'
      form.tags = project.tags || []
    } catch (err) {
      saveError.value = 'Failed to load project: ' + err.message
    }
  }
})
</script>
