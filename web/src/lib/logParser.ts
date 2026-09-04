// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RunTurn, ToolCallRecord } from '@/types/api'

/**
 * isClaudeCodeStreamLog reports whether a log was written by the claude-code-cli
 * driver, which emits stream-json NDJSON rather than the "# turn N" markers the
 * openai-compatible driver writes.
 *
 * Prefers the driver= header, which the runner always writes; falls back to
 * sniffing for a stream-json event so a truncated or header-less log still
 * parses.
 */
function isClaudeCodeStreamLog(logContent: string): boolean {
  const head = logContent.slice(0, 4096)
  if (/^#.*\bdriver=claude-code-cli\b/m.test(head)) return true
  if (/^#.*\bdriver=/m.test(head)) return false
  return /"type"\s*:\s*"(assistant|result)"/.test(head)
}

/**
 * parseClaudeCodeTurns builds the same RunTurn[] from a claude-code-cli
 * stream-json log.
 *
 * The events carry everything the timeline needs — assistant messages holding
 * text/thinking/tool_use blocks, and user messages holding the matching
 * tool_result blocks — they are just shaped differently from the
 * openai-compatible log. Without this, the turn timeline was empty for every
 * Claude Code run regardless of whether it succeeded or failed.
 */
function parseClaudeCodeTurns(logContent: string): RunTurn[] {
  const turns: RunTurn[] = []
  // tool_result blocks arrive in a later event than the tool_use they answer,
  // so collect them first and attach once every turn is built.
  const toolResults = new Map<string, string>()
  const pendingCalls: ToolCallRecord[] = []

  for (const line of logContent.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed.startsWith('{')) continue

    let ev: Record<string, unknown>
    try {
      ev = JSON.parse(trimmed)
    } catch {
      continue // partial line from a truncated or still-writing log
    }

    const msg = ev.message as { content?: unknown } | undefined
    const blocks = Array.isArray(msg?.content) ? (msg.content as Record<string, unknown>[]) : []

    if (ev.type === 'user') {
      for (const b of blocks) {
        if (b?.type === 'tool_result' && typeof b.tool_use_id === 'string') {
          toolResults.set(b.tool_use_id, stringifyBlock(b.content))
        }
      }
      continue
    }

    if (ev.type !== 'assistant' || blocks.length === 0) continue

    const text: string[] = []
    const calls: ToolCallRecord[] = []
    for (const b of blocks) {
      if (b?.type === 'text' && typeof b.text === 'string') {
        text.push(b.text)
      } else if (b?.type === 'thinking' && typeof b.thinking === 'string') {
        // Keep reasoning visible but labelled, so it is not mistaken for the
        // assistant's actual reply.
        text.push(`[thinking]\n${b.thinking}`)
      } else if (b?.type === 'tool_use' && typeof b.id === 'string') {
        const call: ToolCallRecord = {
          id: b.id,
          name: typeof b.name === 'string' ? b.name : 'unknown',
          arguments: stringifyBlock(b.input),
        }
        calls.push(call)
        pendingCalls.push(call)
      }
    }

    const content = text.join('\n').trim()
    if (!content && calls.length === 0) continue

    turns.push({
      turn_number: turns.length + 1,
      role: 'assistant',
      ...(content ? { content } : {}),
      ...(calls.length ? { tool_calls: calls } : {}),
    })
  }

  for (const call of pendingCalls) {
    const res = toolResults.get(call.id)
    if (res) call.result = res
  }

  return turns
}

/** stringifyBlock renders a stream-json block payload as displayable text. */
function stringifyBlock(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') return v
  // tool_result content is sometimes an array of {type:"text",text:"..."} parts.
  if (Array.isArray(v)) {
    const parts = v
      .map((p) =>
        p && typeof p === 'object' && typeof (p as { text?: unknown }).text === 'string'
          ? (p as { text: string }).text
          : null,
      )
      .filter((p): p is string => p !== null)
    if (parts.length) return parts.join('\n')
  }
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

/**
 * parseLogTurns parses an agent run log file into a structured list of turns for
 * the timeline display. Dispatches on log format: the openai-compatible driver
 * writes "# turn N" markers, the claude-code-cli driver writes stream-json.
 */
export function parseLogTurns(logContent: string): RunTurn[] {
  if (!logContent || !logContent.trim()) return []
  if (isClaudeCodeStreamLog(logContent)) return parseClaudeCodeTurns(logContent)

  const turns: RunTurn[] = []
  const lines = logContent.split('\n')

  let currentTurnNumber = 0
  let readingSystem = false
  let readingUser = false
  let readingCompleted = false

  const systemPromptLines: string[] = []
  const userPromptLines: string[] = []
  const completedLines: string[] = []

  let currentTurn: RunTurn | null = null
  let currentToolCall: ToolCallRecord | null = null
  let currentAssistantText: string[] = []
  let recoveredInCurrentTurn = false

  function flushCurrentTurn() {
    if (currentTurn) {
      if (currentToolCall) {
        currentTurn.tool_calls = currentTurn.tool_calls || []
        currentTurn.tool_calls.push(currentToolCall)
        currentToolCall = null
      }
      if (currentAssistantText.length > 0) {
        currentTurn.content = currentAssistantText.join('\n').trim()
      }
      if (currentTurn.content || (currentTurn.tool_calls && currentTurn.tool_calls.length > 0)) {
        turns.push(currentTurn)
      }
      currentTurn = null
      currentAssistantText = []
      recoveredInCurrentTurn = false
    }
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()

    // System prompt section
    if (trimmed.startsWith('# system_prompt:')) {
      readingSystem = true
      readingUser = false
      readingCompleted = false
      continue
    }

    // User prompt section
    if (trimmed.startsWith('# user_prompt:')) {
      if (readingSystem) {
        readingSystem = false
        const sysContent = systemPromptLines.join('\n').trim()
        if (sysContent) {
          turns.push({ turn_number: 0, role: 'system', content: sysContent })
        }
      }
      readingUser = true
      readingCompleted = false
      continue
    }

    // Start of a numbered turn (e.g. "# turn 1")
    const turnMatch = trimmed.match(/^#\s*turn\s+(\d+)/i)
    if (turnMatch) {
      if (readingSystem) {
        readingSystem = false
        const sysContent = systemPromptLines.join('\n').trim()
        if (sysContent) {
          turns.push({ turn_number: 0, role: 'system', content: sysContent })
        }
      }
      if (readingUser) {
        readingUser = false
        const usrContent = userPromptLines.join('\n').trim()
        if (usrContent) {
          turns.push({ turn_number: 1, role: 'user', content: usrContent })
        }
      }

      flushCurrentTurn()
      currentTurnNumber = parseInt(turnMatch[1], 10)
      currentTurn = {
        turn_number: currentTurnNumber + 1, // Offset for 1-based indexing after system/user
        role: 'assistant',
        tool_calls: [],
      }
      continue
    }

    // Completed event section
    if (trimmed.startsWith('# event: completed')) {
      flushCurrentTurn()
      readingSystem = false
      readingUser = false
      readingCompleted = true
      continue
    }

    // Finished or summary line ends completed reading
    if (trimmed.startsWith('# summary:') || trimmed.startsWith('# finished=')) {
      if (readingCompleted) {
        readingCompleted = false
        const compContent = completedLines.join('\n').trim()
        if (compContent) {
          turns.push({
            turn_number: turns.length + 1,
            role: 'assistant',
            content: compContent,
          })
        }
      }
      continue
    }

    // Collecting system prompt
    if (readingSystem) {
      systemPromptLines.push(line)
      continue
    }

    // Collecting user prompt
    if (readingUser) {
      userPromptLines.push(line)
      continue
    }

    // Collecting final completed response
    if (readingCompleted) {
      completedLines.push(line)
      continue
    }

    // Inside a turn:
    if (currentTurn) {
      // Check for FR-5a recovery notice
      if (trimmed.includes('recovered') && trimmed.includes('native tool call')) {
        recoveredInCurrentTurn = true
        currentTurn.is_recovered = true
        continue
      }

      // Check for tool execution: "# executing tool <name> (id: <id>) with args: <args>"
      const execMatch = trimmed.match(/^#\s*executing tool\s+(\S+)\s+\(id:\s*([^)]+)\)\s+with args:\s*(.*)$/i)
      if (execMatch) {
        if (currentToolCall) {
          currentTurn.tool_calls = currentTurn.tool_calls || []
          currentTurn.tool_calls.push(currentToolCall)
        }
        currentToolCall = {
          name: execMatch[1],
          id: execMatch[2],
          arguments: execMatch[3],
          is_recovered: recoveredInCurrentTurn,
        }
        continue
      }

      // Check for tool result: "# tool result (<id>): <result>"
      const resMatch = trimmed.match(/^#\s*tool result\s+\(([^)]+)\):\s*(.*)$/i)
      if (resMatch) {
        const resId = resMatch[1]
        const resVal = resMatch[2]
        if (currentToolCall && currentToolCall.id === resId) {
          currentToolCall.result = resVal
          currentTurn.tool_calls = currentTurn.tool_calls || []
          currentTurn.tool_calls.push(currentToolCall)
          currentToolCall = null
        } else if (currentTurn.tool_calls) {
          const existing = currentTurn.tool_calls.find((tc) => tc.id === resId)
          if (existing) existing.result = resVal
        }
        continue
      }

      // Check for tool error: "# error executing tool: <err>"
      if (trimmed.startsWith('# error executing tool:')) {
        const errMsg = trimmed.slice('# error executing tool:'.length).trim()
        if (currentToolCall) {
          currentToolCall.result = `Error: ${errMsg}`
          currentTurn.tool_calls = currentTurn.tool_calls || []
          currentTurn.tool_calls.push(currentToolCall)
          currentToolCall = null
        }
        continue
      }

      // Streaming JSON chunk from provider
      if (trimmed.startsWith('{') && trimmed.includes('"choices"')) {
        try {
          const parsed = JSON.parse(trimmed)
          if (parsed.choices?.[0]?.delta?.content) {
            currentAssistantText.push(parsed.choices[0].delta.content)
          }
        } catch {
          // ignore unparseable chunk
        }
        continue
      }

      // Other non-comment text lines
      if (!trimmed.startsWith('#')) {
        currentAssistantText.push(line)
      }
    }
  }

  // End of file flush
  if (readingSystem && systemPromptLines.length > 0) {
    const sysContent = systemPromptLines.join('\n').trim()
    if (sysContent) turns.push({ turn_number: 0, role: 'system', content: sysContent })
  }
  if (readingUser && userPromptLines.length > 0) {
    const usrContent = userPromptLines.join('\n').trim()
    if (usrContent) turns.push({ turn_number: 1, role: 'user', content: usrContent })
  }
  if (readingCompleted && completedLines.length > 0) {
    const compContent = completedLines.join('\n').trim()
    if (compContent) {
      turns.push({
        turn_number: turns.length + 1,
        role: 'assistant',
        content: compContent,
      })
    }
  }
  flushCurrentTurn()

  return turns
}
