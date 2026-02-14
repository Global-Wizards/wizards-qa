import { computed } from 'vue'
import { useRoute } from 'vue-router'

/**
 * Composable that provides the project-scoped base path for routing.
 * Returns `/projects/:id` when inside a project, or `''` otherwise.
 */
export function useProjectPath() {
  const route = useRoute()
  const projectId = computed(() => route.params.projectId || '')
  const basePath = computed(() => (projectId.value ? `/projects/${projectId.value}` : ''))
  return { projectId, basePath }
}
