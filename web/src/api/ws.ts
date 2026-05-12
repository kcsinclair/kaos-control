// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WsEvent, WsEventType } from '@/types/api'

type WsHandler = (event: WsEvent) => void

// Custom WebSocket close code used by the backend when the session is invalid
// during the HTTP upgrade handshake.
const AUTH_FAILURE_CLOSE_CODE = 4401

export class WsClient {
  private ws: WebSocket | null = null
  private handlers = new Set<WsHandler>()
  private timer: ReturnType<typeof setTimeout> | null = null
  private delay = 100
  private closed = false

  /** Called when the server rejects the connection with an auth-failure code. */
  onAuthFailure?: () => void

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
    this.ws.onclose = (e: CloseEvent) => {
      if (e.code === AUTH_FAILURE_CLOSE_CODE) {
        // Mark as closed so scheduleReconnect is never called, then notify.
        this.closed = true
        this.onAuthFailure?.()
        return
      }
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

    // Dynamic imports avoid circular deps (router → stores/auth → api/auth → api/client).
    // On auth failure the client is already marked closed, so no reconnect loop occurs.
    client.onAuthFailure = async () => {
      const [{ useAuthStore }, { default: router }] = await Promise.all([
        import('@/stores/auth'),
        import('@/router'),
      ])
      useAuthStore().clearSession()
      _clients.delete(project)
      if (router.currentRoute.value.path !== '/login') {
        await router.push({ path: '/login', query: { expired: '1' } })
      }
    }

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

// Sentinel key for the app-level WS (not project-scoped).
const APP_WS_KEY = '__app__'

// getAppWs returns the singleton WsClient connected to /api/ws.
// This endpoint receives app-level events such as queue.* broadcasts.
export function getAppWs(): WsClient {
  if (!_clients.has(APP_WS_KEY)) {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/api/ws`
    const client = new WsClient(url)

    client.onAuthFailure = async () => {
      const [{ useAuthStore }, { default: router }] = await Promise.all([
        import('@/stores/auth'),
        import('@/router'),
      ])
      useAuthStore().clearSession()
      _clients.delete(APP_WS_KEY)
      if (router.currentRoute.value.path !== '/login') {
        await router.push({ path: '/login', query: { expired: '1' } })
      }
    }

    client.connect()
    _clients.set(APP_WS_KEY, client)
  }
  return _clients.get(APP_WS_KEY)!
}
