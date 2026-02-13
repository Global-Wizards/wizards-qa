<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStorage } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { LayoutDashboard, Sparkles, FlaskConical, FileText, GitBranch, PanelLeftClose, PanelLeft, LogOut, FolderKanban, Settings, Users, ChevronsUpDown, Plus, Globe, Gamepad2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { cn } from '@/lib/utils'
import { useAuth } from '@/composables/useAuth'
import { useProject, clearLastProjectId } from '@/composables/useProject'
import { useConnectionStatus } from '@/composables/useConnectionStatus'
import { versionApi } from '@/lib/api'

const route = useRoute()
const router = useRouter()
const collapsed = useStorage('sidebar-collapsed', false)
const version = ref('')
const changelogOpen = ref(false)
const changelogContent = ref('')
const changelogLoading = ref(false)
const { user, isAdmin, logout } = useAuth()
const { currentProject, isInProject, projects, projectsLoaded, loadProjects, clearProject } = useProject()
const { wsConnected, wsReconnecting } = useConnectionStatus()

onMounted(() => {
  if (!projectsLoaded.value) {
    loadProjects()
  }
})

async function openChangelog() {
  changelogOpen.value = true
  if (changelogContent.value) return
  changelogLoading.value = true
  try {
    const data = await versionApi.changelog()
    changelogContent.value = data.content || ''
  } catch {
    changelogContent.value = 'Failed to load changelog.'
  } finally {
    changelogLoading.value = false
  }
}

function switchToProject(project) {
  router.push(`/projects/${project.id}`)
}

function goToGlobalDashboard() {
  clearProject()
  clearLastProjectId()
  router.push('/')
}

function truncateUrl(url) {
  if (!url) return ''
  try {
    const parsed = new URL(url)
    return parsed.hostname
  } catch {
    return url.length > 30 ? url.slice(0, 30) + '...' : url
  }
}

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/analyses', label: 'Analyses', icon: Sparkles },
  { path: '/tests', label: 'Tests', icon: FlaskConical },
  { path: '/reports', label: 'Reports', icon: FileText },
  { path: '/flows', label: 'Flows', icon: GitBranch },
]

const projectNavItems = computed(() => {
  if (!currentProject.value) return []
  const base = `/projects/${currentProject.value.id}`
  return [
    { path: base, label: 'Dashboard', icon: LayoutDashboard, exact: true },
    { path: `${base}/analyses`, label: 'Analyses', icon: Sparkles },
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

function isNavActive(item) {
  return item.exact ? isExactActive(item.path) : isActive(item.path)
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
      'sidebar-shell flex flex-col h-screen sticky top-0 transition-all duration-300',
      collapsed ? 'w-16' : 'w-60'
    )"
  >
    <!-- Connection Status Banner -->
    <div v-if="wsReconnecting" class="bg-yellow-500/15 text-yellow-600 dark:text-yellow-400 text-[10px] text-center py-1 px-2 border-b border-yellow-500/30">
      {{ collapsed ? '...' : 'Reconnecting...' }}
    </div>
    <div v-else-if="!wsConnected" class="bg-red-500/15 text-red-600 dark:text-red-400 text-[10px] text-center py-1 px-2 border-b border-red-500/30">
      {{ collapsed ? '!' : 'Disconnected' }}
    </div>

    <!-- Project Switcher -->
    <div class="flex items-center h-14 px-2 border-b border-border/50">
      <DropdownMenu>
        <DropdownMenuTrigger>
          <button
            :class="cn(
              'switcher-trigger flex items-center gap-2.5 rounded-lg px-2 py-1.5 w-full text-left transition-all',
              collapsed && 'justify-center'
            )"
          >
            <div
              v-if="isInProject"
              class="project-avatar h-7 w-7 rounded-md flex items-center justify-center text-white text-xs font-bold shrink-0"
              :style="{ backgroundColor: currentProject?.color || '#8b5cf6' }"
            >
              {{ currentProject?.name?.charAt(0)?.toUpperCase() }}
            </div>
            <div
              v-else
              class="h-7 w-7 rounded-md bg-primary/15 flex items-center justify-center shrink-0"
            >
              <Gamepad2 class="h-4 w-4 text-primary" />
            </div>
            <template v-if="!collapsed">
              <span class="flex-1 text-sm font-semibold truncate">
                {{ isInProject ? currentProject?.name : 'Wizards QA' }}
              </span>
              <ChevronsUpDown class="h-3.5 w-3.5 text-muted-foreground/70 shrink-0" />
            </template>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent class="w-56" :side-offset="8">
          <template v-if="projects.length">
            <DropdownMenuItem
              v-for="project in projects"
              :key="project.id"
              class="cursor-pointer"
              @click="switchToProject(project)"
            >
              <div class="flex items-center gap-2.5 w-full">
                <div
                  class="h-5 w-5 rounded flex items-center justify-center text-white text-[10px] font-bold shrink-0"
                  :style="{ backgroundColor: project.color || '#8b5cf6' }"
                >
                  {{ project.name?.charAt(0)?.toUpperCase() }}
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm truncate">{{ project.name }}</p>
                  <p v-if="project.gameUrl" class="text-[10px] text-muted-foreground truncate">{{ truncateUrl(project.gameUrl) }}</p>
                </div>
              </div>
            </DropdownMenuItem>
          </template>
          <p v-else class="px-2 py-1.5 text-xs text-muted-foreground">No projects yet</p>

          <Separator class="my-1" />

          <DropdownMenuItem class="cursor-pointer" @click="$router.push('/projects')">
            <FolderKanban class="h-4 w-4 mr-2 text-muted-foreground" />
            All Projects
          </DropdownMenuItem>
          <DropdownMenuItem class="cursor-pointer" @click="$router.push('/projects/new')">
            <Plus class="h-4 w-4 mr-2 text-muted-foreground" />
            New Project
          </DropdownMenuItem>

          <template v-if="isInProject">
            <Separator class="my-1" />
            <DropdownMenuItem class="cursor-pointer" @click="goToGlobalDashboard">
              <Globe class="h-4 w-4 mr-2 text-muted-foreground" />
              Global Dashboard
            </DropdownMenuItem>
          </template>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 p-1.5 space-y-0.5 overflow-y-auto">
      <!-- Project Mode -->
      <template v-if="isInProject">
        <router-link
          v-for="item in projectNavItems"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'nav-item flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all relative',
            isNavActive(item)
              ? 'nav-active text-primary'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent/50',
            collapsed && 'justify-center px-2'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>

        <Separator class="my-1.5 opacity-50" />

        <router-link
          v-for="item in projectSecondaryNav"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'nav-item flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all relative',
            isActive(item.path)
              ? 'nav-active text-primary'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent/50',
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
            'nav-item flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all relative',
            isActive(item.path)
              ? 'nav-active text-primary'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent/50',
            collapsed && 'justify-center px-2'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </router-link>
      </template>
    </nav>

    <!-- User + Footer -->
    <div class="p-2 border-t border-border/50 space-y-2">
      <div v-if="user" :class="cn('flex items-center gap-2 px-2 py-1', collapsed && 'justify-center px-0')">
        <div class="h-7 w-7 rounded-full bg-primary/15 text-primary flex items-center justify-center text-xs font-bold shrink-0">
          {{ userInitial }}
        </div>
        <div v-if="!collapsed" class="flex-1 min-w-0">
          <p class="text-xs font-medium truncate">{{ user.displayName }}</p>
          <p class="text-[10px] text-muted-foreground">{{ user.role }}</p>
        </div>
        <Button v-if="!collapsed" variant="ghost" size="icon" class="h-7 w-7 shrink-0 text-muted-foreground hover:text-foreground" @click="logout" title="Logout" aria-label="Logout">
          <LogOut class="h-3.5 w-3.5" />
        </Button>
      </div>

      <Separator class="opacity-50" />

      <div :class="cn('flex items-center', collapsed ? 'flex-col gap-1' : 'justify-between')">
        <ThemeToggle />
        <Button variant="ghost" size="icon" class="h-9 w-9 text-muted-foreground hover:text-foreground" @click="collapsed = !collapsed" :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'">
          <PanelLeftClose v-if="!collapsed" class="h-4 w-4" />
          <PanelLeft v-else class="h-4 w-4" />
        </Button>
      </div>
      <div v-if="!collapsed" class="px-2 pb-1 text-center">
        <p v-if="version" class="text-[10px] text-muted-foreground/70">
          <button class="hover:text-primary hover:underline cursor-pointer transition-colors" @click="openChangelog">
            v{{ version }} — Changelog
          </button>
        </p>
        <p class="text-[10px] text-muted-foreground/70">
          Created by <a href="https://www.wizards.us" target="_blank" rel="noopener noreferrer" class="text-primary/80 hover:text-primary hover:underline transition-colors">Wizards</a>
        </p>
      </div>
    </div>

    <!-- Changelog Dialog -->
    <Dialog :open="changelogOpen" @update:open="changelogOpen = $event">
      <DialogContent class="max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Changelog</DialogTitle>
          <DialogDescription>What's new in Wizards QA v{{ version }}</DialogDescription>
        </DialogHeader>
        <div class="flex-1 overflow-y-auto pr-2">
          <div v-if="changelogLoading" class="py-8 text-center text-muted-foreground">Loading changelog...</div>
          <pre v-else class="text-sm whitespace-pre-wrap font-mono leading-relaxed">{{ changelogContent }}</pre>
        </div>
      </DialogContent>
    </Dialog>
  </aside>
</template>

<style scoped>
.sidebar-shell {
  background: hsl(var(--card));
  border-right: 1px solid hsl(var(--border) / 0.5);
}

.dark .sidebar-shell {
  background: hsl(240 10% 5.5%);
}

.switcher-trigger:hover {
  background: hsl(var(--accent) / 0.5);
}

.dark .switcher-trigger:hover {
  background: hsl(var(--primary) / 0.08);
}

.project-avatar {
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.1);
}

.dark .project-avatar {
  box-shadow: 0 0 8px -2px rgba(139, 92, 246, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.08);
}

.nav-active {
  background: hsl(var(--primary) / 0.1);
}

.dark .nav-active {
  background: hsl(var(--primary) / 0.12);
  box-shadow: inset 3px 0 0 0 hsl(var(--primary));
}

.nav-item {
  border-radius: 0.375rem;
}

.dark .nav-item:not(.nav-active):hover {
  background: hsl(var(--primary) / 0.06);
}
</style>
