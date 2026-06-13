// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"errors"
	"strings"
)

// errNoResultLine is returned by ParseResultLine when no type:result JSON line
// is found in the log content. This is expected for Ollama runs.
var errNoResultLine = errors.New("no result line found in log")

// RunResult holds the parsed fields from a Claude Code type:result JSON line.
type RunResult struct {
	Subtype           string                   `json:"subtype"`
	TotalCostUSD      float64                  `json:"total_cost_usd"`
	DurationMs        int64                    `json:"duration_ms"`
	DurationApiMs     int64                    `json:"duration_api_ms"`
	NumTurns          int                      `json:"num_turns"`
	Usage             RunResultUsage           `json:"usage"`
	PermissionDenials []json.RawMessage        `json:"permission_denials"`
	SessionID         string                   `json:"session_id"`
	ModelUsage        map[string]RunModelUsage `json:"modelUsage"`
	// Model is the run's primary model, derived from ModelUsage by
	// ParseResultLine (the model with the most output tokens). Claude Code may
	// use a cheaper background model (e.g. haiku) alongside the primary, so the
	// dominant-by-output-tokens entry is the one worth reporting. Empty when the
	// result line carries no modelUsage (older logs, non-Claude drivers).
	Model string `json:"-"`
}

// RunModelUsage is the per-model usage block from a type:result `modelUsage`
// object. Only the fields needed to pick the primary model are decoded.
type RunModelUsage struct {
	OutputTokens int64   `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// RunResultUsage holds token usage fields from a type:result JSON line.
type RunResultUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
}

// ParseResultLine scans log content from the end and returns the parsed
// RunResult, or nil and an error description if none is found/parseable.
// It is expected that Ollama runs will have no result line; callers should
// treat errNoResultLine as a normal (non-fatal) condition.
func ParseResultLine(logContent string) (*RunResult, error) {
	// Trim trailing newlines so the last Split element is not empty.
	trimmed := strings.TrimRight(logContent, "\n\r")
	if trimmed == "" {
		return nil, errNoResultLine
	}

	lines := strings.Split(trimmed, "\n")
	// Scan backwards: the result line is typically the last or near-last line.
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Quick pre-check to avoid full JSON decode on every line.
		if !strings.Contains(line, `"type"`) {
			continue
		}

		// Decode enough to check the type field.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			// Malformed JSON on a line that looks like it has "type" — report it.
			if strings.Contains(line, `"result"`) {
				return nil, errors.New("malformed JSON in result line: " + err.Error())
			}
			continue
		}

		var typVal string
		if raw["type"] == nil {
			continue
		}
		if err := json.Unmarshal(raw["type"], &typVal); err != nil {
			continue
		}
		if typVal != "result" {
			continue
		}

		// Found a type:result line — decode the full struct.
		var result RunResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, errors.New("malformed JSON in result line: " + err.Error())
		}
		result.Model = dominantModel(result.ModelUsage)
		return &result, nil
	}

	return nil, errNoResultLine
}

// dominantModel returns the run's primary model: the entry in modelUsage with
// the most output tokens (tie-break: higher cost, then lexicographically first
// name for determinism). Returns "" for an empty/nil map.
func dominantModel(usage map[string]RunModelUsage) string {
	best := ""
	var bestTokens int64 = -1
	var bestCost float64 = -1
	for name, u := range usage {
		better := u.OutputTokens > bestTokens ||
			(u.OutputTokens == bestTokens && u.CostUSD > bestCost) ||
			(u.OutputTokens == bestTokens && u.CostUSD == bestCost && (best == "" || name < best))
		if better {
			best, bestTokens, bestCost = name, u.OutputTokens, u.CostUSD
		}
	}
	return best
}
