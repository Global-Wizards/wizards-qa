<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Projects</h2>
        <p class="text-muted-foreground">Organize your testing work by game or application</p>
      </div>
      <router-link to="/projects/new">
        <Button>
          <Plus class="h-4 w-4 mr-2" />
          New Project
        </Button>
      </router-link>
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
      <!-- Project Grid -->
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <router-link
          v-for="project in projects"
          :key="project.id"
          :to="`/projects/${project.id}`"
          class="block group"
        >
          <Card class="hover:shadow-md transition-all duration-200 hover:border-primary/20 h-full">
            <CardHeader class="pb-3">
              <div class="flex items-center gap-3">
                <div
                  class="h-10 w-10 rounded-lg flex items-center justify-center text-white text-sm font-bold shrink-0"
                  :style="{ backgroundColor: project.color || '#6366f1' }"
                >
                  {{ project.name.charAt(0).toUpperCase() }}
                </div>
                <div class="flex-1 min-w-0">
                  <CardTitle class="text-base truncate">{{ project.name }}</CardTitle>
                  <p class="text-xs text-muted-foreground truncate mt-0.5">{{ project.gameUrl || 'No URL set' }}</p>
                </div>
              </div>
            </CardHeader>
            <CardContent class="pt-0">
              <p v-if="project.description" class="text-sm text-muted-foreground mb-3 line-clamp-2">
                {{ project.description }}
              </p>

              <div class="flex items-center gap-4 text-xs text-muted-foreground">
                <span>{{ project.analysisCount }} analyses</span>
                <span>{{ project.planCount }} plans</span>
                <span>{{ project.testCount }} tests</span>
              </div>

              <div v-if="project.tags?.length" class="flex flex-wrap gap-1 mt-2">
                <Badge v-for="tag in project.tags.slice(0, 3)" :key="tag" variant="secondary" class="text-xs">
                  {{ tag }}
                </Badge>
                <Badge v-if="project.tags.length > 3" variant="secondary" class="text-xs">
                  +{{ project.tags.length - 3 }}
                </Badge>
              </div>
            </CardContent>
          </Card>
        </router-link>

        <!-- New Project CTA Card -->
        <router-link to="/projects/new" class="block group">
          <Card class="hover:shadow-md transition-all duration-200 border-dashed h-full flex items-center justify-center min-h-[180px]">
            <CardContent class="flex flex-col items-center justify-center text-center py-8">
              <div class="rounded-full bg-primary/10 p-3 mb-3">
                <Plus class="h-6 w-6 text-primary" />
              </div>
              <p class="font-medium">Create New Project</p>
              <p class="text-sm text-muted-foreground">Group analyses and tests together</p>
            </CardContent>
          </Card>
        </router-link>
      </div>

      <div v-if="!projects.length" class="text-center py-12">
        <div class="rounded-full bg-primary/10 p-4 mx-auto mb-4 w-fit">
          <FolderKanban class="h-10 w-10 text-primary" />
        </div>
        <h3 class="text-lg font-semibold mb-1">No projects yet</h3>
        <p class="text-muted-foreground mb-4">Create your first project to organize your testing work</p>
        <router-link to="/projects/new">
          <Button>
            <Plus class="h-4 w-4 mr-2" />
            Create Project
          </Button>
        </router-link>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Plus, AlertCircle, FolderKanban } from 'lucide-vue-next'
import { projectsApi } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import LoadingSkeleton from '@/components/LoadingSkeleton.vue'

const loading = ref(true)
const error = ref(null)
const projects = ref([])

onMounted(async () => {
  try {
    const data = await projectsApi.list()
    projects.value = data.projects || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>
