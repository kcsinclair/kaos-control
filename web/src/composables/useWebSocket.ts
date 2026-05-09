// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMounted, onUnmounted } from 'vue'
import { getProjectWs } from '@/api/ws'
import type { WsEvent, WsEventType } from '@/types/api'

export function useWebSocket(project: string, type: WsEventType, handler: (e: WsEvent) => void): void {
  let unsub: (() => void) | null = null

  onMounted(() => {
    const ws = getProjectWs(project)
    unsub = ws.onType(type, handler)
  })

  onUnmounted(() => {
    unsub?.()
  })
}
