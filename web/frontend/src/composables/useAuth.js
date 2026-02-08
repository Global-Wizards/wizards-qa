import { ref, computed } from 'vue'
import { authApi } from '@/lib/api'

const user = ref(null)
const loading = ref(true)

function saveTokens(data) {
  localStorage.setItem('accessToken', data.accessToken)
  localStorage.setItem('refreshToken', data.refreshToken)
}

export function clearTokens() {
  localStorage.removeItem('accessToken')
  localStorage.removeItem('refreshToken')
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
    const token = localStorage.getItem('accessToken')
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
        const refreshToken = localStorage.getItem('refreshToken')
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
  return localStorage.getItem('accessToken')
}
