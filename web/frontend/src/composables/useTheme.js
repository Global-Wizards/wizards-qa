import { ref, watch } from 'vue'
import { STORAGE_KEYS } from '@/lib/constants'

const isDark = ref(false)

function initTheme() {
  const stored = localStorage.getItem(STORAGE_KEYS.theme)
  if (stored) {
    isDark.value = stored === 'dark'
  } else {
    isDark.value = window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  applyTheme()
}

function applyTheme() {
  if (isDark.value) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
}

watch(isDark, () => {
  localStorage.setItem(STORAGE_KEYS.theme, isDark.value ? 'dark' : 'light')
  applyTheme()
})

export function useTheme() {
  return {
    isDark,
    toggleTheme,
    initTheme,
  }
}
