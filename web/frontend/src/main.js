import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import { useTheme } from './composables/useTheme'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/Dashboard.vue') },
    { path: '/analyze', component: () => import('./views/Analyze.vue') },
    { path: '/tests', component: () => import('./views/Tests.vue') },
    { path: '/tests/new', component: () => import('./views/NewTestPlan.vue') },
    { path: '/reports', component: () => import('./views/Reports.vue') },
    { path: '/flows', component: () => import('./views/Flows.vue') },
  ],
})

const { initTheme } = useTheme()
initTheme()

const app = createApp(App)
app.use(router)
app.mount('#app')
