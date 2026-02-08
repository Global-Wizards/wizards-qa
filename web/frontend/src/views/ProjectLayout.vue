<script setup>
import { watch, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProject } from '@/composables/useProject'

const route = useRoute()
const { loadProject, clearProject, currentProject } = useProject()

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
    <router-view />
  </div>
</template>
