<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { LayoutDashboard, Sparkles, FlaskConical, FileText, GitBranch, PanelLeftClose, PanelLeft, LogOut, FolderKanban, ArrowLeft, Settings, Users } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { cn } from '@/lib/utils'
import { useAuth } from '@/composables/useAuth'
import { useProject } from '@/composables/useProject'
import { versionApi } from '@/lib/api'

const route = useRoute()
const collapsed = ref(false)
const version = ref('')
const { user, isAdmin, logout } = useAuth()
const { currentProject, isInProject } = useProject()

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/analyze', label: 'Analyze', icon: Sparkles },
  { path: '/tests', label: 'Tests', icon: FlaskConical },
  { path: '/reports', label: 'Reports', icon: FileText },
  { path: '/flows', label: 'Flows', icon: GitBranch },
  { path: '/projects', label: 'Projects', icon: FolderKanban },
]

const projectNavItems = computed(() => {
  if (!currentProject.value) return []
  const base = `/projects/${currentProject.value.id}`
  return [
    { path: base, label: 'Dashboard', icon: LayoutDashboard, exact: true },
    { path: `${base}/analyze`, label: 'Analyze', icon: Sparkles },
    { path: `${base}/tests`, label: 'Tests', icon: FlaskConical },
    { path: `${base}/reports`, label: 'Reports', icon: FileText },
    { path: `${base}/flows`, label: 'Flows', icon: GitBranch },
  ]
})

const projectSecondaryNav = computed(() => {
  if (!currentProject.value) return []
  const base = `/projects/${currentProject.value.id}`
  return [
    { path: `${base}/members`, label: 'Members', icon: Users },
    { path: `${base}/settings`, label: 'Settings', icon: Settings },
  ]
})

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function isExactActive(path) {
  return route.path === path
}

const userInitial = computed(() => {
  if (!user.value?.displayName) return '?'
  return user.value.displayName.charAt(0).toUpperCase()
})

onMounted(async () => {
  try {
    const data = await versionApi.get()
    version.value = data.version || ''
  } catch {
    // non-critical
  }
})
</script>

<template>
  <aside
    :class="cn(
      'flex flex-col border-r bg-card h-screen sticky top-0 transition-all duration-300',
      collapsed ? 'w-16' : 'w-60'
    )"
  >
    <!-- Logo -->
    <div class="flex items-center h-16 px-4 border-b">
      <span class="text-xl font-bold text-primary whitespace-nowrap">
        {{ collapsed ? 'W' : 'Wizards QA' }}
      </span>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 p-2 space-y-1 overflow-y-auto">
      <!-- Project Mode -->
      <template v-if="isInProject">
        <router-link
          to="/projects"
          :class="cn(
            'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors text-muted-foreground hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2'
          )"
        >
          <ArrowLeft class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">Back to Projects</span>
        </router-link>

        <!-- Project Header -->
        <div v-if="!collapsed" class="flex items-center gap-2 px-3 py-2">
          <div
            class="h-6 w-6 rounded flex items-center justify-center text-white text-xs font-bold shrink-0"
            :style="{ backgroundColor: currentProject?.color || '#6366f1' }"
          >
            {{ currentProject?.name?.charAt(0)?.toUpperCase() }}
          </div>
          <span class="text-sm font-semibold truncate">{{ currentProject?.name }}</span>
        </div>

        <Separator v-if="!collapsed" class="my-1" />

        <router-link
          v-for="item in projectNavItems"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
            (item.exact ? isExactActive(item.path) : isActive(item.path))
              ? 'bg-primary/10 text-primary'
              : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>

        <Separator v-if="!collapsed" class="my-1" />

        <router-link
          v-for="item in projectSecondaryNav"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
            isActive(item.path)
              ? 'bg-primary/10 text-primary'
              : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>
      </template>

      <!-- Global Mode -->
      <template v-else>
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
            isActive(item.path)
              ? 'bg-primary/10 text-primary'
              : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
            collapsed && 'justify-center px-2'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>
      </template>
    </nav>

    <!-- User + Footer -->
    <div class="p-2 border-t space-y-2">
      <!-- User info -->
      <div v-if="user" :class="cn('flex items-center gap-2 px-2 py-1', collapsed && 'justify-center px-0')">
        <div class="h-7 w-7 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-bold shrink-0">
          {{ userInitial }}
        </div>
        <div v-if="!collapsed" class="flex-1 min-w-0">
          <p class="text-xs font-medium truncate">{{ user.displayName }}</p>
          <p class="text-[10px] text-muted-foreground">{{ user.role }}</p>
        </div>
        <Button v-if="!collapsed" variant="ghost" size="icon" class="h-7 w-7 shrink-0" @click="logout" title="Logout">
          <LogOut class="h-3.5 w-3.5" />
        </Button>
      </div>

      <Separator v-if="!collapsed" />

      <div :class="cn('flex items-center', collapsed ? 'flex-col gap-1' : 'justify-between')">
        <ThemeToggle />
        <Button variant="ghost" size="icon" class="h-9 w-9" @click="collapsed = !collapsed">
          <PanelLeftClose v-if="!collapsed" class="h-4 w-4" />
          <PanelLeft v-else class="h-4 w-4" />
        </Button>
      </div>
      <div v-if="!collapsed" class="px-2 pb-1 text-center">
        <p v-if="version" class="text-[10px] text-muted-foreground">v{{ version }}</p>
        <p class="text-[10px] text-muted-foreground">
          Created by <a href="https://www.wizards.us" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">Wizards</a>
        </p>
      </div>
    </div>
  </aside>
</template>
