export interface WsEvent {
  type: string
  payload: unknown
}

/**
 * Build a Cookie header string from a Playwright BrowserContext's cookies.
 * Used to authenticate Node-side WebSocket connections against kaos-control's
 * session-cookie auth (server closes unauthenticated WS with code 4401).
 */
export async function cookieHeaderFromContext(
  context: { cookies: () => Promise<Array<{ name: string; value: string }>> },
): Promise<string> {
  const cookies = await context.cookies()
  return cookies.map((c) => `${c.name}=${c.value}`).join('; ')
}

/** Subscribe to a project WebSocket and collect events. */
export function connectProjectWs(
  baseURL: string,
  project: string,
  cookieHeader = '',
): {
  events: WsEvent[]
  waitFor: (type: string, timeoutMs?: number) => Promise<WsEvent>
  close: () => void
} {
  const wsURL = baseURL.replace(/^http/, 'ws') + `/api/p/${project}/ws`
  // Node's global WebSocket is undici-backed and accepts a `headers` option
  // (the WHATWG types don't expose it, so we cast through `unknown`). Server
  // closes unauthenticated WS with code 4401, so we must pass session cookie.
  const wsOpts = cookieHeader
    ? ({ headers: { Cookie: cookieHeader } } as unknown as string[] | undefined)
    : undefined
  const ws = new WebSocket(wsURL, wsOpts)
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
