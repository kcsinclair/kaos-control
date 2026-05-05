import { ref, onUnmounted } from 'vue'
import { getProjectWs } from '@/api/ws'

// Grace period after our own save during which fsnotify-triggered file.changed events are ignored.
// This correctly distinguishes user-initiated saves from server-side auto-block rewrites:
// the 3 s window closes long before the backend indexer triggers file.changed, so an
// auto-block while the user is editing will show the conflict banner as expected.
const SAVE_GRACE_MS = 3_000
const AUTO_REFRESH_DEBOUNCE_MS = 300

interface ExternalChangeOptions {
  isDirty?: () => boolean
  onAutoRefresh?: () => void
}

export function useExternalChange(
  project: string,
  artifactPath: string,
  options?: ExternalChangeOptions,
) {
  const hasExternalChange = ref(false)
  let lastSaveMs = 0
  let debounceTimer: ReturnType<typeof setTimeout> | undefined

  const ws = getProjectWs(project)
  const unsub = ws.onType('file.changed', (e) => {
    const path = e.payload?.path as string | undefined
    if (path !== artifactPath) return
    if (Date.now() - lastSaveMs < SAVE_GRACE_MS) return

    if (options?.isDirty?.() === false && options?.onAutoRefresh) {
      // Auto-refresh path: debounce to coalesce rapid successive events
      clearTimeout(debounceTimer)
      debounceTimer = setTimeout(() => {
        options.onAutoRefresh!()
      }, AUTO_REFRESH_DEBOUNCE_MS)
    } else {
      // Conflict banner path (dirty or no auto-refresh callback)
      clearTimeout(debounceTimer)
      hasExternalChange.value = true
    }
  })

  onUnmounted(() => {
    unsub()
    clearTimeout(debounceTimer)
  })

  function markSaved(): void {
    lastSaveMs = Date.now()
    hasExternalChange.value = false
  }

  function acknowledge(): void {
    hasExternalChange.value = false
  }

  return { hasExternalChange, markSaved, acknowledge }
}
