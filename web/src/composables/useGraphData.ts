// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted } from 'vue'
import { useGraphStore } from '@/stores/graph'
import { useWebSocket } from '@/composables/useWebSocket'

export function useGraphData(project: string) {
  const store = useGraphStore()

  useWebSocket(project, 'artifact.indexed', () => {
    store.fetchGraph(project)
  })

  onMounted(() => {
    store.fetchGraph(project)
  })

  return store
}
