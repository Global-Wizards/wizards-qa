import axios from 'axios'
import { STORAGE_KEYS } from '@/lib/constants'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: attach access token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401 with token refresh
let isRefreshing = false
let failedQueue = []

function processQueue(error, token = null) {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })
  failedQueue = []
}

api.interceptors.response.use(
  (response) => response.data,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      // Don't try to refresh for auth endpoints
      if (originalRequest.url?.includes('/auth/')) {
        const message = error.response?.data?.error || error.message || 'An error occurred'
        return Promise.reject(new Error(message))
      }

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then((token) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            return api(originalRequest)
          })
          .catch((err) => Promise.reject(err))
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const refreshToken = localStorage.getItem(STORAGE_KEYS.refreshToken)
        if (!refreshToken) {
          throw new Error('No refresh token')
        }

        const { data } = await axios.post('/api/auth/refresh', { refreshToken })
        localStorage.setItem(STORAGE_KEYS.accessToken, data.accessToken)
        localStorage.setItem(STORAGE_KEYS.refreshToken, data.refreshToken)

        processQueue(null, data.accessToken)
        originalRequest.headers.Authorization = `Bearer ${data.accessToken}`
        return api(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError, null)
        localStorage.removeItem(STORAGE_KEYS.accessToken)
        localStorage.removeItem(STORAGE_KEYS.refreshToken)
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    const message = error.response?.data?.error || error.message || 'An error occurred'
    const err = new Error(message)
    err.response = error.response
    return Promise.reject(err)
  }
)

/**
 * Append the current access token as a query param for URLs loaded by
 * <img src>, <video src>, etc. that can't use the axios Authorization header.
 */
export function authUrl(path) {
  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (!token) return path
  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}token=${encodeURIComponent(token)}`
}

export const authApi = {
  register: (data) => api.post('/auth/register', data),
  login: (data) => api.post('/auth/login', data),
  refresh: (data) => api.post('/auth/refresh', data),
  me: () => api.get('/auth/me'),
}

export const statsApi = {
  getStats: () => api.get('/stats'),
}

export const testsApi = {
  list: () => api.get('/tests'),
  get: (id) => api.get(`/tests/${id}`),
  live: (id) => api.get(`/tests/${id}/live`),
  run: (payload) => api.post('/tests/run', payload),
  delete: (id) => api.delete(`/tests/${id}`),
  deleteBatch: (ids) => api.post('/tests/delete-batch', { ids }),
}

export const reportsApi = {
  list: () => api.get('/reports'),
  get: (id) => api.get(`/reports/${id}`),
}

export const flowsApi = {
  list: () => api.get('/flows'),
  get: (name) => api.get(`/flows/${name}`),
  validate: (content) => api.post('/flows/validate', { content }),
}

export const configApi = {
  get: () => api.get('/config'),
}

export const performanceApi = {
  get: () => api.get('/performance'),
}

export const templatesApi = {
  list: () => api.get('/templates'),
}

export const testPlansApi = {
  list: () => api.get('/test-plans'),
  get: (id, params) => api.get(`/test-plans/${id}`, { params }),
  create: (plan) => api.post('/test-plans', plan),
  update: (id, data) => api.put(`/test-plans/${id}`, data),
  delete: (id) => api.delete(`/test-plans/${id}`),
  run: (id, opts = {}) => api.post(`/test-plans/${id}/run`, opts),
}

export const analyzeApi = {
  start: (gameUrl, projectId, agentMode = false, profileParams = {}, modules = {}) =>
    api.post('/analyze', { gameUrl, projectId: projectId || '', agentMode, modules, ...profileParams }),
  batchAnalyze: (request) => api.post('/analyze/batch', request),
  sendHint: (analysisId, message) => api.post(`/analyses/${analysisId}/message`, { message }),
  continue: (analysisId) => api.post(`/analyses/${analysisId}/continue`),
}

export const analysesApi = {
  list: () => api.get('/analyses'),
  get: (id) => api.get(`/analyses/${id}`),
  status: (id) => api.get(`/analyses/${id}/status`),
  delete: (id) => api.delete(`/analyses/${id}`),
  steps: (id) => api.get(`/analyses/${id}/steps`),
  flows: (id) => api.get(`/analyses/${id}/flows`),
  stepScreenshotUrl: (id, stepNumber) => authUrl(`/api/analyses/${id}/steps/${stepNumber}/screenshot`),
  screenshotUrl: (id, filename) => authUrl(`/api/analyses/${id}/screenshots/${encodeURIComponent(filename)}`),
  exportUrl: (id, format = 'json') => `/api/analyses/${id}/export?format=${format}`,
  export: (id, format = 'json') =>
    axios.get(`/api/analyses/${id}/export?format=${format}`, {
      responseType: 'blob',
      headers: { Authorization: `Bearer ${localStorage.getItem(STORAGE_KEYS.accessToken)}` },
    }).then((r) => {
      const ext = format === 'markdown' ? 'md' : 'json'
      const blob = new Blob([r.data])
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${id}.${ext}`
      a.click()
      URL.revokeObjectURL(url)
    }),
}

export const testPlansDeleteApi = {
  delete: (id) => api.delete(`/test-plans/${id}`),
}

export const projectsApi = {
  list: () => api.get('/projects'),
  get: (id) => api.get(`/projects/${id}`),
  create: (data) => api.post('/projects', data),
  update: (id, data) => api.put(`/projects/${id}`, data),
  delete: (id) => api.delete(`/projects/${id}`),
  stats: (id) => api.get(`/projects/${id}/stats`),
  analyses: (id) => api.get(`/projects/${id}/analyses`),
  testPlans: (id) => api.get(`/projects/${id}/test-plans`),
  tests: (id) => api.get(`/projects/${id}/tests`),
  members: {
    list: (pid) => api.get(`/projects/${pid}/members`),
    add: (pid, data) => api.post(`/projects/${pid}/members`, data),
    updateRole: (pid, uid, data) => api.put(`/projects/${pid}/members/${uid}`, data),
    remove: (pid, uid) => api.delete(`/projects/${pid}/members/${uid}`),
  },
}

export const versionApi = {
  get: () => api.get('/version'),
  changelog: () => api.get('/changelog'),
}

export default api
