import { ref, computed } from 'vue'
import { projectsApi } from '@/lib/api'

const LAST_PROJECT_KEY = 'wizards-qa-last-project'

const currentProject = ref(null)
const loading = ref(false)
const loadError = ref(null)
const projects = ref([])
const projectsLoaded = ref(false)

export function getLastProjectId() {
  return localStorage.getItem(LAST_PROJECT_KEY)
}

export function clearLastProjectId() {
  localStorage.removeItem(LAST_PROJECT_KEY)
}

export function useProject() {
  const projectId = computed(() => currentProject.value?.id || null)
  const projectName = computed(() => currentProject.value?.name || '')
  const isInProject = computed(() => !!currentProject.value)

  async function loadProjects() {
    try {
      const data = await projectsApi.list()
      projects.value = data.projects || []
      projectsLoaded.value = true
    } catch (err) {
      console.error('Failed to load projects:', err)
    }
  }

  async function loadProject(id) {
    if (currentProject.value?.id === id) return
    loading.value = true
    loadError.value = null
    try {
      currentProject.value = await projectsApi.get(id)
      localStorage.setItem(LAST_PROJECT_KEY, id)
      if (!projectsLoaded.value) {
        loadProjects()
      }
    } catch (err) {
      console.error('Failed to load project:', err)
      currentProject.value = null
      loadError.value = err.message || 'Failed to load project'
    } finally {
      loading.value = false
    }
  }

  function clearProject() {
    currentProject.value = null
    clearLastProjectId()
  }

  function projectPath(subpath) {
    if (!currentProject.value) return subpath
    return `/projects/${currentProject.value.id}${subpath ? '/' + subpath : ''}`
  }

  return {
    currentProject,
    loading,
    loadError,
    projectId,
    projectName,
    isInProject,
    projects,
    projectsLoaded,
    loadProject,
    loadProjects,
    clearProject,
    projectPath,
  }
}
