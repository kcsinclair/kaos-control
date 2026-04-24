import { ref, onUnmounted } from 'vue'
import { getProjectWs } from '@/api/ws'

// Grace period after our own save during which fsnotify-triggered file.changed events are ignored.
const SAVE_GRACE_MS = 3_000

export function useExternalChange(project: string, artifactPath: string) {
  const hasExternalChange = ref(false)
  let lastSaveMs = 0

  const ws = getProjectWs(project)
  const unsub = ws.onType('file.changed', (e) => {
    const path = e.payload?.path as string | undefined
    if (path !== artifactPath) return
    if (Date.now() - lastSaveMs < SAVE_GRACE_MS) return
    hasExternalChange.value = true
  })

  onUnmounted(unsub)

  function markSaved(): void {
    lastSaveMs = Date.now()
    hasExternalChange.value = false
  }

  function acknowledge(): void {
    hasExternalChange.value = false
  }

  return { hasExternalChange, markSaved, acknowledge }
}
