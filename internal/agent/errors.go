// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// Structured failure-reason codes (local-model-operability FR-3/NFR-2).
// These classify why an agent run failed so the UI can show actionable
// remediation instead of a raw stack trace. Not every run failure is
// classifiable — ClassifyRunError returns "" when it can't confidently
// match one of these.
const (
	FailureReasonToolsUnsupported      = "tools_unsupported"
	FailureReasonModelNotFound         = "model_not_found"
	FailureReasonModelUnloaded         = "model_unloaded"
	FailureReasonEndpointUnreachable   = "endpoint_unreachable"
	FailureReasonContextWindowExceeded = "context_window_exceeded"
	FailureReasonTurnTokenCeiling      = "turn_token_ceiling"
	FailureReasonMaxIterationsReached  = "max_iterations_reached"
	FailureReasonAuthError             = "auth_error"
	FailureReasonTimeout               = "timeout"
)

// ErrMaxIterationsReached indicates the openai-compatible driver hit its
// max_tool_iterations cap without the model ever returning finish_reason:
// stop — i.e. the agent never completed its task.
var ErrMaxIterationsReached = errors.New("max tool iterations reached")

var (
	toolsUnsupportedRemediation = []string{
		"The model or provider rejected or silently dropped the tools parameter.",
		"Verify the model supports function calling (check the provider's model card).",
		"For llama-server, ensure it was started with --jinja and a tool-calling chat template.",
	}
	modelUnloadedRemediation = []string{
		"The provider reported the model as unloaded or failed to allocate memory for it.",
		"For Ollama/llama.cpp, confirm the host has enough free RAM/VRAM for this model's quantization.",
		"Retry once the server finishes loading, or choose a smaller quantization.",
	}
	contextWindowRemediation = []string{
		"The model's context window was exceeded during prompt generation or a multi-turn loop.",
		"Shorten the input artifact, reduce max_tool_iterations, or choose a model with a larger context window.",
	}
	turnTokenCeilingRemediation = []string{
		"The model hit its per-turn generation token limit without completing a tool call.",
		"Increase the provider's per-request max_tokens, if configurable, or simplify the prompt.",
	}
	maxIterationsRemediation = []string{
		"The agent reached max_tool_iterations without finishing its task.",
		"Increase 'max_tool_iterations' on this agent's config, or split the target artifact into smaller milestones.",
	}
	authErrorFailureRemediation = []string{
		"The provider rejected the request as unauthorized (401/403).",
		"Verify the provider's 'api_key' is set and has not expired or been revoked.",
	}
	runTimeoutRemediation = []string{
		"The run exceeded its configured timeout_minutes.",
		"Increase 'timeout_minutes' on this agent's config, or check whether the provider is unusually slow/overloaded.",
	}
)

// ClassifyRunError inspects a driver failure (err), the run's captured
// stderr tail, and any extra structured details, and returns a
// FailureReason* code with matching remediation steps and a sanitized
// errorDetails map suitable for persistence and WebSocket broadcast.
// Returns reason == "" when the failure doesn't match a known pattern —
// callers should treat that as "unclassified", not an error.
//
// Matching precedence: known sentinel errors (from this driver's own
// preflight/streaming code) first, then *RunError's HTTP classification,
// then best-effort text matching against err/stderr for patterns emitted by
// third-party providers we don't control (llama.cpp, Ollama, OpenRouter,
// etc). All returned text is passed through maskSecretsInText first
// (standards/secrets-handling.md).
func ClassifyRunError(err error, stderr string, details map[string]any) (reason string, remediation []string, errorDetails map[string]any) {
	if err == nil && strings.TrimSpace(stderr) == "" {
		return "", nil, nil
	}

	reason, remediation = classifyReason(err, stderr)
	if reason == "" {
		return "", nil, nil
	}

	errorDetails = make(map[string]any, len(details)+2)
	for k, v := range details {
		if s, ok := v.(string); ok {
			errorDetails[k] = maskSecretsInText(s)
		} else {
			errorDetails[k] = v
		}
	}
	if err != nil {
		errorDetails["message"] = maskSecretsInText(err.Error())
	}
	if excerpt := strings.TrimSpace(stderr); excerpt != "" {
		if len(excerpt) > 500 {
			excerpt = excerpt[len(excerpt)-500:]
		}
		errorDetails["stderr_excerpt"] = maskSecretsInText(excerpt)
	}

	return reason, remediation, errorDetails
}

func classifyReason(err error, stderr string) (string, []string) {
	switch {
	case errors.Is(err, ErrModelNotFound):
		return FailureReasonModelNotFound, modelNotFoundRemediation
	case errors.Is(err, ErrEndpointUnreachable):
		return FailureReasonEndpointUnreachable, endpointUnreachableRemediation
	case errors.Is(err, ErrToolsUnsupported), errors.Is(err, ErrToolsSilentlyDropped):
		return FailureReasonToolsUnsupported, toolsUnsupportedRemediation
	case errors.Is(err, ErrModelLoadTimeout):
		return FailureReasonModelUnloaded, modelUnloadedRemediation
	case errors.Is(err, ErrMaxIterationsReached):
		return FailureReasonMaxIterationsReached, maxIterationsRemediation
	case errors.Is(err, context.DeadlineExceeded):
		return FailureReasonTimeout, runTimeoutRemediation
	}

	var runErr *RunError
	if errors.As(err, &runErr) {
		if strings.Contains(runErr.Error(), "HTTP 503") {
			return FailureReasonModelUnloaded, modelUnloadedRemediation
		}
		if runErr.Kind == RateLimitKindUnreachable {
			return FailureReasonEndpointUnreachable, endpointUnreachableRemediation
		}
	}

	var combined strings.Builder
	if err != nil {
		combined.WriteString(err.Error())
		combined.WriteByte(' ')
	}
	combined.WriteString(stderr)
	lower := strings.ToLower(combined.String())
	if lower == "" {
		return "", nil
	}

	switch {
	case containsAny(lower, "401", "403", "unauthorized", "invalid api key", "invalid_api_key", "authentication failed"):
		return FailureReasonAuthError, authErrorFailureRemediation
	case containsAny(lower, "context_length_exceeded", "context length exceeded", "context window", "maximum context length", "exceeds the model's context", "too many tokens"):
		return FailureReasonContextWindowExceeded, contextWindowRemediation
	case containsAny(lower, "max tool iterations reached", "max tool iterations cap"):
		return FailureReasonMaxIterationsReached, maxIterationsRemediation
	case containsAny(lower, `"finish_reason":"length"`, "finish_reason: length"):
		return FailureReasonTurnTokenCeiling, turnTokenCeilingRemediation
	case containsAny(lower, "model not found", "no such model", "model_not_found", "does not exist"):
		return FailureReasonModelNotFound, modelNotFoundRemediation
	case containsAny(lower, "connection refused", "no such host", "dial tcp", "i/o timeout", "unreachable", "eof"):
		return FailureReasonEndpointUnreachable, endpointUnreachableRemediation
	}

	return "", nil
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

// maskSecretsInText redacts bearer tokens and common API-key/auth-header
// patterns before error text is persisted or broadcast
// (standards/secrets-handling.md: keep secrets out of logs and API
// responses). Best-effort: it catches the shapes providers commonly echo
// back in error bodies, not an exhaustive secret scanner.
var (
	bearerPattern = regexp.MustCompile(`(?i)(Bearer\s+)\S+`)
	// apiKeyPattern intentionally excludes "authorization": that header is
	// almost always "Bearer <token>", already handled by bearerPattern; a
	// key=value match here would instead mask the word "Bearer" itself and
	// leave the actual token exposed right after it.
	apiKeyPattern = regexp.MustCompile(`(?i)("?(?:api[_-]?key|access[_-]?token|auth[_-]?token)"?\s*[:=]\s*"?)[^"\s,}]+`)
)

func maskSecretsInText(s string) string {
	s = bearerPattern.ReplaceAllString(s, "${1}***")
	s = apiKeyPattern.ReplaceAllString(s, "${1}***")
	return s
}
