// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
)

// errNoAgyResultLine is returned by ParseAgyResultLine when no event:result
// JSON line is found in the log content. This is expected for a run that is
// still in progress, or one where agy exited before emitting a terminal
// event (see gemini-cli-stream-json FR-4 / NFR-2).
var errNoAgyResultLine = errors.New("no agy result line found in log")

// agyResultEvent is the top-level shape of an agy stream-json line. The
// discriminator is "event" (not "type", as in Claude's stream-json), and the
// payload nests under a key matching the event name.
type agyResultEvent struct {
	Event  string     `json:"event"`
	Result *agyResult `json:"result"`
}

// agyResult is the nested payload of an agy `{"event":"result",...}` line.
type agyResult struct {
	Status       string   `json:"status"`
	Response     string   `json:"response"`
	NumTurns     int      `json:"num_turns"`
	DurationSecs float64  `json:"duration_seconds"`
	Usage        agyUsage `json:"usage"`
}

// agyUsage is the nested usage object of an agy result event.
type agyUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ThinkingTokens  int64 `json:"thinking_tokens"`
	CacheReadTokens int64 `json:"cache_read_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// ParseAgyResultLine scans log content from the end and returns the parsed
// RunResult from an agy `{"event":"result",...}` line, or nil and
// errNoAgyResultLine if none is found. Deliberately separate from
// ParseResultLine: agy's NDJSON uses a different discriminator ("event" vs
// "type") and payload shape, and reusing the Claude parser would silently
// match nothing (gemini-cli-stream-json FR-2).
func ParseAgyResultLine(logContent string) (*RunResult, error) {
	trimmed := strings.TrimRight(logContent, "\n\r")
	if trimmed == "" {
		return nil, errNoAgyResultLine
	}

	lines := strings.Split(trimmed, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.Contains(line, `"event"`) {
			continue
		}

		var ev agyResultEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Event != "result" || ev.Result == nil {
			continue
		}

		r := ev.Result
		durationMs := int64(math.Round(r.DurationSecs * 1000))
		isError := r.Status != "SUCCESS"
		result := &RunResult{
			Subtype:       strings.ToLower(r.Status),
			IsError:       isError,
			DurationMs:    durationMs,
			DurationApiMs: durationMs,
			NumTurns:      r.NumTurns,
			Usage: RunResultUsage{
				InputTokens:          r.Usage.InputTokens,
				OutputTokens:         r.Usage.OutputTokens,
				CacheReadInputTokens: r.Usage.CacheReadTokens,
			},
		}
		if isError {
			result.Result = r.Status
			if r.Response != "" {
				result.Result = r.Status + ": " + r.Response
			}
		} else {
			result.Result = r.Response
		}
		return result, nil
	}

	return nil, errNoAgyResultLine
}
