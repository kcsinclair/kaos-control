// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { WsClient } from '../ws'

// Frontend plan: lifecycle/frontend-plans/rate-limit-event-detection-4-fe.md
// Milestone 3 — an agent.quota_status event with no active consumer must be a
// silent no-op at the WS dispatch layer: no console.warn, and it must not
// disrupt delivery of a subsequent event to a subscribed handler.

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: ((e: CloseEvent) => void) | null = null
  onerror: (() => void) | null = null

  constructor(public url: string) {
    FakeWebSocket.instances.push(this)
  }

  close(): void {}
}

describe('WsClient — unmodelled/no-consumer event types', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('does not warn and does not disrupt a sibling handler for a following event', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const client = new WsClient('ws://test')
    client.connect()
    const ws = FakeWebSocket.instances[0]

    const progressEvents: string[] = []
    client.onType('agent.progress', (e) => progressEvents.push(e.type))

    ws.onmessage?.({ data: JSON.stringify({ type: 'agent.quota_status', payload: { run_id: 'run-1' } }) } as MessageEvent)
    ws.onmessage?.({ data: JSON.stringify({ type: 'agent.progress', payload: { run_id: 'run-1' } }) } as MessageEvent)

    expect(warnSpy).not.toHaveBeenCalled()
    expect(progressEvents).toEqual(['agent.progress'])

    warnSpy.mockRestore()
  })
})
