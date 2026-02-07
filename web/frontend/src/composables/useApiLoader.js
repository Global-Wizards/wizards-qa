import { ref } from 'vue'

/**
 * Reusable composable for API loading pattern.
 * Eliminates repeated try/catch/finally loading/error boilerplate.
 */
export function useApiLoader(apiFn) {
  const data = ref(null)
  const loading = ref(true)
  const error = ref(null)

  async function load() {
    loading.value = true
    error.value = null
    try {
      data.value = await apiFn()
    } catch (err) {
      error.value = err.message || 'An error occurred'
      console.error('API load failed:', err)
    } finally {
      loading.value = false
    }
  }

  async function reload() {
    return load()
  }

  return { data, loading, error, load, reload }
}
