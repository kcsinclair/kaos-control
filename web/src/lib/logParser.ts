// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RunTurn, ToolCallRecord } from '@/types/api'

/**
 * parseLogTurns parses an agent run log file (especially OpenAI-compatible multi-turn logs)
 * into a structured list of turns for the timeline display.
 */
export function parseLogTurns(logContent: string): RunTurn[] {
  if (!logContent || !logContent.trim()) return []

  const turns: RunTurn[] = []
  const lines = logContent.split('\n')

  let currentTurnNumber = 0
  let readingSystem = false
  let readingUser = false
  let readingCompleted = false

  let systemPromptLines: string[] = []
  let userPromptLines: string[] = []
  let completedLines: string[] = []

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
