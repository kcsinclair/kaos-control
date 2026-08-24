// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { parseLogTurns } from '../logParser'

describe('parseLogTurns', () => {
  it('returns empty array for empty or empty-like log', () => {
    expect(parseLogTurns('')).toEqual([])
    expect(parseLogTurns('   \n  ')).toEqual([])
  })

  it('parses system prompt, user prompt, and multi-turn tool execution', () => {
    const log = `
# kaos-control agent run run-123
# agent=backend-dev role=backend-developer driver=openai-compatible provider=llama-cpp base_url=http://127.0.0.1:8080 model=gemma
# started=2026-08-25T10:00:00Z

# system_prompt:
You are a backend developer.

# user_prompt:
Implement feature X.

# turn 1
{"id":"chatcmpl-1","choices":[{"delta":{"content":"Let me inspect files."}}]}
# executing tool list_dir (id: call_1) with args: {"path":"internal"}
# tool result (call_1): ["agent","artifact","config"]

# turn 2
{"id":"chatcmpl-2","choices":[{"delta":{"content":"Now reading agent.go."}}]}
# recovered 1 native tool call(s) (FR-5a)
# executing tool read_file (id: call_2) with args: {"path":"internal/agent/agent.go"}
# tool result (call_2): package agent

# event: completed
Feature X implementation completed successfully.
# summary: recovered_native_tool_calls=1 finish_reason=stop
# finished=2026-08-25T10:00:15Z
`

    const turns = parseLogTurns(log)
    expect(turns.length).toBeGreaterThanOrEqual(4)

    // System prompt turn
    const sysTurn = turns.find((t) => t.role === 'system')
    expect(sysTurn).toBeDefined()
    expect(sysTurn?.content).toBe('You are a backend developer.')

    // User prompt turn
    const userTurn = turns.find((t) => t.role === 'user')
    expect(userTurn).toBeDefined()
    expect(userTurn?.content).toBe('Implement feature X.')

    // Turn 1 with tool call
    const t1 = turns.find((t) => t.tool_calls?.some((tc) => tc.id === 'call_1'))
    expect(t1).toBeDefined()
    expect(t1?.content).toContain('Let me inspect files.')
    expect(t1?.tool_calls?.[0].name).toBe('list_dir')
    expect(t1?.tool_calls?.[0].arguments).toBe('{"path":"internal"}')
    expect(t1?.tool_calls?.[0].result).toBe('["agent","artifact","config"]')

    // Turn 2 with recovered tool call
    const t2 = turns.find((t) => t.tool_calls?.some((tc) => tc.id === 'call_2'))
    expect(t2).toBeDefined()
    expect(t2?.is_recovered).toBe(true)
    expect(t2?.tool_calls?.[0].is_recovered).toBe(true)
    expect(t2?.tool_calls?.[0].name).toBe('read_file')
    expect(t2?.tool_calls?.[0].result).toBe('package agent')

    // Completed turn
    const lastTurn = turns[turns.length - 1]
    expect(lastTurn.role).toBe('assistant')
    expect(lastTurn.content).toBe('Feature X implementation completed successfully.')
  })
})
