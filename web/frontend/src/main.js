import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import { useTheme } from './composables/useTheme'
import { useAuth } from './composables/useAuth'
import { getLastProjectId } from './composables/useProject'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('./views/Login.vue'), meta: { public: true } },
    { path: '/', component: () => import('./views/Dashboard.vue') },
    { path: '/analyze', component: () => import('./views/Analyze.vue') },
    { path: '/tests', component: () => import('./views/Tests.vue') },
    { path: '/tests/new', component: () => import('./views/NewTestPlan.vue') },
    { path: '/reports', component: () => import('./views/Reports.vue') },
    { path: '/flows', component: () => import('./views/Flows.vue') },
    { path: '/analyses/:id', component: () => import('./views/AnalysisDetail.vue') },
    { path: '/projects', component: () => import('./views/ProjectList.vue') },
    { path: '/projects/new', component: () => import('./views/ProjectForm.vue') },
    {
      path: '/projects/:projectId',
      component: () => import('./views/ProjectLayout.vue'),
      children: [
        { path: '', component: () => import('./views/ProjectDashboard.vue') },
        { path: 'analyze', component: () => import('./views/Analyze.vue') },
        { path: 'tests', component: () => import('./views/Tests.vue') },
        { path: 'tests/new', component: () => import('./views/NewTestPlan.vue') },
        { path: 'reports', component: () => import('./views/Reports.vue') },
        { path: 'flows', component: () => import('./views/Flows.vue') },
        { path: 'analyses/:id', component: () => import('./views/AnalysisDetail.vue') },
        { path: 'settings', component: () => import('./views/ProjectSettings.vue') },
        { path: 'members', component: () => import('./views/ProjectMembers.vue') },
        { path: 'edit', component: () => import('./views/ProjectForm.vue') },
      ],
    },
  ],
})

// Navigation guard: require auth for non-public routes
router.beforeEach((to, from, next) => {
  const { isAuthenticated, loading } = useAuth()

  // If still loading auth state, allow navigation (loadUser will redirect if needed)
  if (loading.value && to.path !== '/login') {
    next()
    return
  }

  if (!to.meta.public && !isAuthenticated.value) {
    next('/login')
  } else if (to.path === '/login' && isAuthenticated.value) {
    next('/')
  } else if (to.path === '/' && isAuthenticated.value) {
    const lastProjectId = getLastProjectId()
    if (lastProjectId) {
      next(`/projects/${lastProjectId}`)
    } else {
      next()
    }
  } else {
    next()
  }
})

const { initTheme } = useTheme()
initTheme()

const app = createApp(App)
app.use(router)

// Wait for router to resolve initial route before mounting,
// so $route.meta is available and App.vue renders the correct branch
router.isReady().then(() => {
  app.mount('#app')

  // Load user on startup
  const { loadUser } = useAuth()
  loadUser().then(() => {
    const { isAuthenticated } = useAuth()
    if (!isAuthenticated.value && window.location.pathname !== '/login') {
      router.push('/login')
    }
  })
})
