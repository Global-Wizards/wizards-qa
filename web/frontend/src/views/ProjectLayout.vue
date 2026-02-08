<script setup>
import { watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProject } from '@/composables/useProject'
import { Button } from '@/components/ui/button'

const route = useRoute()
const router = useRouter()
const { loadProject, clearProject, currentProject, loading, loadError } = useProject()

watch(
  () => route.params.projectId,
  (id) => {
    if (id) loadProject(id)
  },
  { immediate: true }
)

onUnmounted(() => {
  clearProject()
})
</script>

<template>
  <div :style="currentProject?.color ? { '--project-color': currentProject.color } : {}">
    <!-- Loading -->
    <div v-if="loading && !currentProject" class="flex items-center justify-center py-20 text-muted-foreground">
      Loading project...
    </div>

    <!-- Error -->
    <div v-else-if="loadError" class="flex flex-col items-center justify-center py-20 gap-4">
      <p class="text-destructive">{{ loadError }}</p>
      <Button variant="outline" @click="router.push('/projects')">Back to Projects</Button>
    </div>

    <!-- Content -->
    <router-view v-else />
  </div>
</template>
