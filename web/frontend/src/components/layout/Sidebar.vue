<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { LayoutDashboard, Sparkles, FlaskConical, FileText, GitBranch, PanelLeftClose, PanelLeft } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { cn } from '@/lib/utils'

const route = useRoute()
const collapsed = ref(false)
const version = ref('')

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/analyze', label: 'Analyze', icon: Sparkles },
  { path: '/tests', label: 'Tests', icon: FlaskConical },
  { path: '/reports', label: 'Reports', icon: FileText },
  { path: '/flows', label: 'Flows', icon: GitBranch },
]

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

onMounted(async () => {
  try {
    const res = await fetch('/api/version')
    const data = await res.json()
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
    <nav class="flex-1 p-2 space-y-1">
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
    </nav>

    <!-- Footer -->
    <div class="p-2 border-t space-y-2">
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
