import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

// Import views
import Dashboard from './views/Dashboard.vue'
import Tests from './views/Tests.vue'
import Reports from './views/Reports.vue'
import Flows from './views/Flows.vue'

// Router configuration
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/tests', component: Tests },
    { path: '/reports', component: Reports },
    { path: '/flows', component: Flows },
  ]
})

const app = createApp(App)
app.use(router)
app.mount('#app')
