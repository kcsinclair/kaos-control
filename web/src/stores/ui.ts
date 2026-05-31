// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastType = 'info' | 'success' | 'error'

export interface Toast {
  id: number
  type: ToastType
  message: string
}

let _nextId = 0

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])

  const sidebarCollapsed = ref<boolean>(
    localStorage.getItem('sidebar-collapsed') === 'true'
  )

  function toggleSidebar(): void {
    sidebarCollapsed.value = !sidebarCollapsed.value
    localStorage.setItem('sidebar-collapsed', String(sidebarCollapsed.value))
  }

  // Mobile drawer state — separate from sidebarCollapsed (the desktop
  // expand/collapse preference). On mobile the sidebar is hidden by default
  // and slides in as a drawer; on desktop this flag is unused.
  // Session-only; resets to closed on every page load.
  const mobileSidebarOpen = ref<boolean>(false)

  function openMobileSidebar(): void {
    mobileSidebarOpen.value = true
  }
  function closeMobileSidebar(): void {
    mobileSidebarOpen.value = false
  }
  function toggleMobileSidebar(): void {
    mobileSidebarOpen.value = !mobileSidebarOpen.value
  }

  // Whether test artifacts are visible in the Kanban board.
  // Session-only (not persisted to localStorage per plan §F3).
  const showTestsOnKanban = ref<boolean>(false)

  function addToast(type: ToastType, message: string, duration = 4000): void {
    const id = ++_nextId
    toasts.value.push({ id, type, message })
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, duration)
  }

  function dismiss(id: number): void {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return {
    toasts,
    info: (msg: string) => addToast('info', msg),
    success: (msg: string) => addToast('success', msg),
    error: (msg: string) => addToast('error', msg),
    dismiss,
    sidebarCollapsed,
    toggleSidebar,
    mobileSidebarOpen,
    openMobileSidebar,
    closeMobileSidebar,
    toggleMobileSidebar,
    showTestsOnKanban,
  }
})
