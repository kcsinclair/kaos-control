import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type Theme = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'kaos-theme'

function systemIsDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(t: Theme): void {
  const dark = t === 'dark' || (t === 'system' && systemIsDark())
  document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
}

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem(STORAGE_KEY) as Theme | null) ?? 'system')

  const isDark = computed(() =>
    theme.value === 'dark' || (theme.value === 'system' && systemIsDark()),
  )

  function setTheme(t: Theme): void {
    theme.value = t
    localStorage.setItem(STORAGE_KEY, t)
    applyTheme(t)
  }

  function toggle(): void {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  function init(): void {
    applyTheme(theme.value)
  }

  return { theme, isDark, setTheme, toggle, init }
})
