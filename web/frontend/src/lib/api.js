import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: attach access token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken')
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
  (response) => response,
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
        const refreshToken = localStorage.getItem('refreshToken')
        if (!refreshToken) {
          throw new Error('No refresh token')
        }

        const { data } = await axios.post('/api/auth/refresh', { refreshToken })
        localStorage.setItem('accessToken', data.accessToken)
        localStorage.setItem('refreshToken', data.refreshToken)

        processQueue(null, data.accessToken)
        originalRequest.headers.Authorization = `Bearer ${data.accessToken}`
        return api(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError, null)
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    const message = error.response?.data?.error || error.message || 'An error occurred'
    return Promise.reject(new Error(message))
  }
)

export const authApi = {
  register: (data) => api.post('/auth/register', data).then((r) => r.data),
  login: (data) => api.post('/auth/login', data).then((r) => r.data),
  refresh: (data) => api.post('/auth/refresh', data).then((r) => r.data),
  me: () => api.get('/auth/me').then((r) => r.data),
}

export const statsApi = {
  getStats: () => api.get('/stats').then((r) => r.data),
}

export const testsApi = {
  list: () => api.get('/tests').then((r) => r.data),
  get: (id) => api.get(`/tests/${id}`).then((r) => r.data),
  live: (id) => api.get(`/tests/${id}/live`).then((r) => r.data),
  run: (payload) => api.post('/tests/run', payload).then((r) => r.data),
  delete: (id) => api.delete(`/tests/${id}`).then((r) => r.data),
  deleteBatch: (ids) => api.post('/tests/delete-batch', { ids }).then((r) => r.data),
}

export const reportsApi = {
  list: () => api.get('/reports').then((r) => r.data),
  get: (id) => api.get(`/reports/${id}`).then((r) => r.data),
}

export const flowsApi = {
  list: () => api.get('/flows').then((r) => r.data),
  get: (name) => api.get(`/flows/${name}`).then((r) => r.data),
  validate: (content) => api.post('/flows/validate', { content }).then((r) => r.data),
}

export const configApi = {
  get: () => api.get('/config').then((r) => r.data),
}

export const performanceApi = {
  get: () => api.get('/performance').then((r) => r.data),
}

export const templatesApi = {
  list: () => api.get('/templates').then((r) => r.data),
}

export const testPlansApi = {
  list: () => api.get('/test-plans').then((r) => r.data),
  get: (id, params) => api.get(`/test-plans/${id}`, { params }).then((r) => r.data),
  create: (plan) => api.post('/test-plans', plan).then((r) => r.data),
  update: (id, data) => api.put(`/test-plans/${id}`, data).then((r) => r.data),
  delete: (id) => api.delete(`/test-plans/${id}`).then((r) => r.data),
  run: (id, opts = {}) => api.post(`/test-plans/${id}/run`, opts).then((r) => r.data),
}

export const analyzeApi = {
  start: (gameUrl, projectId, agentMode = false, profileParams = {}, modules = {}) =>
    api.post('/analyze', { gameUrl, projectId: projectId || '', agentMode, modules, ...profileParams }).then((r) => r.data),
  batchAnalyze: (request) =>
    api.post('/analyze/batch', request).then((r) => r.data),
  sendHint: (analysisId, message) =>
    api.post(`/analyses/${analysisId}/message`, { message }).then((r) => r.data),
  continue: (analysisId) =>
    api.post(`/analyses/${analysisId}/continue`).then((r) => r.data),
}

export const analysesApi = {
  list: () => api.get('/analyses').then((r) => r.data),
  get: (id) => api.get(`/analyses/${id}`).then((r) => r.data),
  status: (id) => api.get(`/analyses/${id}/status`).then((r) => r.data),
  delete: (id) => api.delete(`/analyses/${id}`).then((r) => r.data),
  steps: (id) => api.get(`/analyses/${id}/steps`).then((r) => r.data),
  flows: (id) => api.get(`/analyses/${id}/flows`).then((r) => r.data),
  stepScreenshotUrl: (id, stepNumber) => `/api/analyses/${id}/steps/${stepNumber}/screenshot`,
  exportUrl: (id, format = 'json') => `/api/analyses/${id}/export?format=${format}`,
  export: (id, format = 'json') =>
    api.get(`/analyses/${id}/export?format=${format}`, { responseType: 'blob' }).then((r) => {
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
  delete: (id) => api.delete(`/test-plans/${id}`).then((r) => r.data),
}

export const projectsApi = {
  list: () => api.get('/projects').then((r) => r.data),
  get: (id) => api.get(`/projects/${id}`).then((r) => r.data),
  create: (data) => api.post('/projects', data).then((r) => r.data),
  update: (id, data) => api.put(`/projects/${id}`, data).then((r) => r.data),
  delete: (id) => api.delete(`/projects/${id}`).then((r) => r.data),
  stats: (id) => api.get(`/projects/${id}/stats`).then((r) => r.data),
  analyses: (id) => api.get(`/projects/${id}/analyses`).then((r) => r.data),
  testPlans: (id) => api.get(`/projects/${id}/test-plans`).then((r) => r.data),
  tests: (id) => api.get(`/projects/${id}/tests`).then((r) => r.data),
  members: {
    list: (pid) => api.get(`/projects/${pid}/members`).then((r) => r.data),
    add: (pid, data) => api.post(`/projects/${pid}/members`, data).then((r) => r.data),
    updateRole: (pid, uid, data) => api.put(`/projects/${pid}/members/${uid}`, data).then((r) => r.data),
    remove: (pid, uid) => api.delete(`/projects/${pid}/members/${uid}`).then((r) => r.data),
  },
}

export const versionApi = {
  get: () => api.get('/version').then((r) => r.data),
  changelog: () => api.get('/changelog').then((r) => r.data),
}

export default api
