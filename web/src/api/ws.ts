import type { WsEvent, WsEventType } from '@/types/api'

type WsHandler = (event: WsEvent) => void

export class WsClient {
  private ws: WebSocket | null = null
  private handlers = new Set<WsHandler>()
  private timer: ReturnType<typeof setTimeout> | null = null
  private delay = 100
  private closed = false

  constructor(private readonly url: string) {}

  connect(): void {
    if (this.closed) return
    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      this.delay = 100
    }
    this.ws.onmessage = (e: MessageEvent) => {
      try {
        const evt = JSON.parse(e.data as string) as WsEvent
        this.handlers.forEach((h) => h(evt))
      } catch {
        // ignore malformed messages
      }
    }
    this.ws.onclose = () => {
      if (!this.closed) this.scheduleReconnect()
    }
    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  private scheduleReconnect(): void {
    if (this.timer) return
    this.timer = setTimeout(() => {
      this.timer = null
      this.delay = Math.min(this.delay * 2, 30_000)
      this.connect()
    }, this.delay)
  }

  on(handler: WsHandler): () => void {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler)
  }

  onType(type: WsEventType, handler: WsHandler): () => void {
    const wrapped: WsHandler = (e) => { if (e.type === type) handler(e) }
    return this.on(wrapped)
  }

  disconnect(): void {
    this.closed = true
    if (this.timer) { clearTimeout(this.timer); this.timer = null }
    this.ws?.close()
    this.ws = null
  }
}

// One WsClient instance per project, created lazily.
const _clients = new Map<string, WsClient>()

export function getProjectWs(project: string): WsClient {
  if (!_clients.has(project)) {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/api/p/${encodeURIComponent(project)}/ws`
    const client = new WsClient(url)
    client.connect()
    _clients.set(project, client)
  }
  return _clients.get(project)!
}

export function closeProjectWs(project: string): void {
  const client = _clients.get(project)
  if (client) {
    client.disconnect()
    _clients.delete(project)
  }
}
