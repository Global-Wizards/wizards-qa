<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Project Settings</h2>
        <p class="text-muted-foreground">Manage {{ currentProject?.name || 'project' }} settings</p>
      </div>
    </div>

    <div class="max-w-2xl space-y-6">
      <!-- Project Info -->
      <Card>
        <CardHeader>
          <CardTitle class="text-lg">Project Information</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm font-medium">Name</p>
              <p class="text-sm text-muted-foreground">{{ currentProject?.name }}</p>
            </div>
          </div>
          <Separator />
          <div>
            <p class="text-sm font-medium">Game URL</p>
            <p class="text-sm text-muted-foreground">{{ currentProject?.gameUrl || 'Not set' }}</p>
          </div>
          <Separator />
          <div>
            <p class="text-sm font-medium">Description</p>
            <p class="text-sm text-muted-foreground">{{ currentProject?.description || 'No description' }}</p>
          </div>
          <Separator />
          <div>
            <p class="text-sm font-medium">Created</p>
            <p class="text-sm text-muted-foreground">{{ currentProject?.createdAt ? new Date(currentProject.createdAt).toLocaleDateString() : '-' }}</p>
          </div>
          <Separator />
          <router-link :to="`/projects/${route.params.projectId}/edit`">
            <Button variant="outline" class="mt-2">Edit Project</Button>
          </router-link>
        </CardContent>
      </Card>

      <!-- Danger Zone -->
      <Card class="border-destructive/50">
        <CardHeader>
          <CardTitle class="text-lg text-destructive">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground mb-4">
            Deleting this project will unassign all analyses, test plans, and test results. This action cannot be undone.
          </p>
          <Button variant="destructive" @click="handleDelete" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete Project' }}
          </Button>
          <p v-if="deleteError" class="text-sm text-destructive mt-2">{{ deleteError }}</p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { projectsApi } from '@/lib/api'
import { useProject } from '@/composables/useProject'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

const route = useRoute()
const router = useRouter()
const { currentProject } = useProject()
const deleting = ref(false)
const deleteError = ref(null)

async function handleDelete() {
  if (!confirm('Are you sure you want to delete this project?')) return
  deleting.value = true
  deleteError.value = null
  try {
    await projectsApi.delete(route.params.projectId)
    router.push('/projects')
  } catch (err) {
    deleteError.value = err.message
  } finally {
    deleting.value = false
  }
}
</script>
