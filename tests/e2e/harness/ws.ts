export interface WsEvent {
  type: string
  payload: unknown
}

/** Subscribe to a project WebSocket and collect events. */
export function connectProjectWs(baseURL: string, project: string): {
  events: WsEvent[]
  waitFor: (type: string, timeoutMs?: number) => Promise<WsEvent>
  close: () => void
} {
  const wsURL = baseURL.replace(/^http/, 'ws') + `/api/p/${project}/ws`
  const ws = new WebSocket(wsURL)
  const events: WsEvent[] = []
  const listeners: Array<{ type: string; resolve: (e: WsEvent) => void }> = []

  ws.addEventListener('message', (msg) => {
    let ev: WsEvent
    try {
      ev = JSON.parse(msg.data as string) as WsEvent
    } catch {
      return
    }
    events.push(ev)
    for (let i = listeners.length - 1; i >= 0; i--) {
      if (listeners[i].type === ev.type) {
        const [matched] = listeners.splice(i, 1)
        matched.resolve(ev)
      }
    }
  })

  const waitFor = (type: string, timeoutMs = 5_000): Promise<WsEvent> =>
    new Promise<WsEvent>((resolve, reject) => {
      // Check already-received events first
      const existing = events.find((e) => e.type === type)
      if (existing) {
        resolve(existing)
        return
      }
      const timer = setTimeout(
        () => reject(new Error(`Timed out waiting for WS event: ${type}`)),
        timeoutMs,
      )
      listeners.push({
        type,
        resolve: (e) => {
          clearTimeout(timer)
          resolve(e)
        },
      })
    })

  const close = () => ws.close()

  return { events, waitFor, close }
}
