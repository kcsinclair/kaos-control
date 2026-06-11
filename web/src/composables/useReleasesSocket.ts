// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted, onUnmounted } from 'vue'
import { useReleasesStore } from '@/stores/releases'

/**
 * Connects the releases store to the project WebSocket on mount and
 * disconnects on unmount. Prefer this over calling connectWs/disconnectWs
 * directly so the lifecycle is managed declaratively.
 */
export function useReleasesSocket(project: string): void {
  const store = useReleasesStore()

  onMounted(() => {
    store.connectWs(project)
  })

  onUnmounted(() => {
    store.disconnectWs()
  })
}
