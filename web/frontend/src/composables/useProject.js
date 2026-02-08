import { ref, computed } from 'vue'
import { projectsApi } from '@/lib/api'

const currentProject = ref(null)
const loading = ref(false)

export function useProject() {
  const projectId = computed(() => currentProject.value?.id || null)
  const projectName = computed(() => currentProject.value?.name || '')
  const isInProject = computed(() => !!currentProject.value)

  async function loadProject(id) {
    if (currentProject.value?.id === id) return
    loading.value = true
    try {
      currentProject.value = await projectsApi.get(id)
    } catch (err) {
      console.error('Failed to load project:', err)
      currentProject.value = null
    } finally {
      loading.value = false
    }
  }

  function clearProject() {
    currentProject.value = null
  }

  function projectPath(subpath) {
    if (!currentProject.value) return subpath
    return `/projects/${currentProject.value.id}${subpath ? '/' + subpath : ''}`
  }

  return {
    currentProject,
    loading,
    projectId,
    projectName,
    isInProject,
    loadProject,
    clearProject,
    projectPath,
  }
}
