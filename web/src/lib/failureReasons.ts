// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FailureReason } from '@/types/api'

/**
 * Static diagnostic copy for the local-model-operability structured failure
 * taxonomy (FR-3/FR-4). The backend already sends `remediation` steps
 * per-run (ClassifyRunError); this is the fallback used when that's absent,
 * plus the heading/explanation text the backend doesn't send at all.
 */
export interface FailureReasonInfo {
  heading: string
  explanation: (details?: Record<string, unknown> | null, providerName?: string) => string
  remediation: string[]
}

function detailStr(details: Record<string, unknown> | null | undefined, key: string): string | undefined {
  const v = details?.[key]
  return typeof v === 'string' && v ? v : undefined
}

const contextLimitInfo: FailureReasonInfo = {
  heading: 'Model context limit or token ceiling reached',
  explanation: () =>
    "The prompt and artifact history exceeded the model's context window, or generation hit the per-turn token ceiling before completing a tool call.",
  remediation: [
    'Reduce the input artifact length.',
    'Increase the server context size (`-c 16384`).',
    'Tune the agent prompt template to be more concise.',
  ],
}

export const FAILURE_REASON_INFO: Partial<Record<FailureReason, FailureReasonInfo>> = {
  tools_unsupported: {
    heading: 'Model does not support tool calling (Function Calling)',
    explanation: () =>
      'The selected model or its chat template silently dropped or rejected tool schemas, preventing it from reading or writing files.',
    remediation: [
      'Switch this agent to a model with verified tool-calling support (e.g. `gemma-4-26B`, `qwen3-coder:30b`, `gpt-oss-20b`).',
      'If using `llama-server`, ensure the `--jinja` flag is enabled.',
      'Open Provider Settings to test model capabilities.',
    ],
  },
  model_not_found: {
    heading: 'Model not found on provider',
    explanation: (details, providerName) => {
      const model = detailStr(details, 'model')
      const provider = providerName ? ` on \`${providerName}\`` : ' on the configured provider'
      return `The model${model ? ` \`${model}\`` : ''} is not loaded or registered${provider}.`
    },
    remediation: [
      'Verify the model name is spelled correctly in Agent Config.',
      'Run `ollama pull <model>` or launch `llama-server -m <model.gguf>` to make the model available.',
    ],
  },
  model_unloaded: {
    heading: 'Model failed to load into memory',
    explanation: () =>
      'The upstream server returned HTTP 503 or ran out of VRAM/RAM while loading the model weights.',
    remediation: [
      'Check GPU/CPU memory availability on the inference host.',
      'Reduce the context size, or select a smaller quantization.',
    ],
  },
  endpoint_unreachable: {
    heading: 'Cannot connect to inference provider',
    explanation: (_details, providerName) =>
      `Connection to${providerName ? ` \`${providerName}\`` : ' the provider'} failed (connection refused or DNS lookup failure).`,
    remediation: [
      'Check that the local inference server (Ollama or llama.cpp) is running on the expected port.',
    ],
  },
  context_window_exceeded: contextLimitInfo,
  turn_token_ceiling: contextLimitInfo,
  max_iterations_reached: {
    heading: 'Maximum tool iterations cap reached',
    explanation: () =>
      'The agent reached its maximum allowed tool iterations without completing the artifact.',
    remediation: [
      'Check the prompt instructions for ambiguity or loops.',
      "Increase `max_tool_iterations` in this agent's settings.",
    ],
  },
  auth_error: {
    heading: 'Authentication failed with provider',
    explanation: () => 'The upstream provider rejected the API key or credentials (HTTP 401/403).',
    remediation: ['Verify the configured API key in Provider Settings.'],
  },
  timeout: {
    heading: 'Agent run timed out',
    explanation: () => 'The run exceeded its configured timeout_minutes.',
    remediation: [
      "Increase `timeout_minutes` on this agent's configuration.",
      'Check whether the provider is unusually slow or overloaded.',
    ],
  },
}

/** Looks up diagnostic copy for a structured failure_reason code; null for legacy/unclassified codes. */
export function getFailureReasonInfo(reason?: string | null): FailureReasonInfo | null {
  if (!reason) return null
  return FAILURE_REASON_INFO[reason as FailureReason] ?? null
}
