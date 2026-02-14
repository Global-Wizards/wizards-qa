import { ref, computed } from 'vue'
import { authApi } from '@/lib/api'
import { STORAGE_KEYS } from '@/lib/constants'

const user = ref(null)
const loading = ref(true)

function saveTokens(data) {
  localStorage.setItem(STORAGE_KEYS.accessToken, data.accessToken)
  localStorage.setItem(STORAGE_KEYS.refreshToken, data.refreshToken)
}

export function clearTokens() {
  localStorage.removeItem(STORAGE_KEYS.accessToken)
  localStorage.removeItem(STORAGE_KEYS.refreshToken)
}

export function useAuth() {
  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(email, password) {
    const data = await authApi.login({ email, password })
    saveTokens(data)
    user.value = data.user
    return data.user
  }

  async function register(email, password, displayName) {
    const data = await authApi.register({ email, password, displayName })
    saveTokens(data)
    user.value = data.user
    return data.user
  }

  function logout() {
    clearTokens()
    user.value = null
    window.location.href = '/login'
  }

  async function loadUser() {
    const token = localStorage.getItem(STORAGE_KEYS.accessToken)
    if (!token) {
      loading.value = false
      return
    }
    try {
      const data = await authApi.me()
      user.value = data
    } catch {
      // Token invalid — try refresh
      try {
        const refreshToken = localStorage.getItem(STORAGE_KEYS.refreshToken)
        if (refreshToken) {
          const data = await authApi.refresh({ refreshToken })
          saveTokens(data)
          user.value = data.user
        }
      } catch {
        clearTokens()
        user.value = null
      }
    } finally {
      loading.value = false
    }
  }

  return { user, loading, isAuthenticated, isAdmin, login, register, logout, loadUser }
}

export function getAccessToken() {
  return localStorage.getItem(STORAGE_KEYS.accessToken)
}
