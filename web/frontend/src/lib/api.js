import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.error || error.message || 'An error occurred'
    return Promise.reject(new Error(message))
  }
)

export const statsApi = {
  getStats: () => api.get('/stats').then((r) => r.data),
}

export const testsApi = {
  list: () => api.get('/tests').then((r) => r.data),
  get: (id) => api.get(`/tests/${id}`).then((r) => r.data),
  run: (payload) => api.post('/tests/run', payload).then((r) => r.data),
}

export const reportsApi = {
  list: () => api.get('/reports').then((r) => r.data),
  get: (id) => api.get(`/reports/${id}`).then((r) => r.data),
}

export const flowsApi = {
  list: () => api.get('/flows').then((r) => r.data),
  get: (name) => api.get(`/flows/${name}`).then((r) => r.data),
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
  get: (id) => api.get(`/test-plans/${id}`).then((r) => r.data),
  create: (plan) => api.post('/test-plans', plan).then((r) => r.data),
  run: (id) => api.post(`/test-plans/${id}/run`).then((r) => r.data),
}

export const analyzeApi = {
  start: (gameUrl) => api.post('/analyze', { gameUrl }).then((r) => r.data),
}

export const analysesApi = {
  list: () => api.get('/analyses').then((r) => r.data),
  get: (id) => api.get(`/analyses/${id}`).then((r) => r.data),
  delete: (id) => api.delete(`/analyses/${id}`).then((r) => r.data),
  exportUrl: (id, format = 'json') => `/api/analyses/${id}/export?format=${format}`,
}

export const testPlansDeleteApi = {
  delete: (id) => api.delete(`/test-plans/${id}`).then((r) => r.data),
}

export default api
