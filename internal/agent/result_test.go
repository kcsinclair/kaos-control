// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"testing"
)

// validResultLine is a realistic type:result JSON line from a Claude Code run.
const validResultLine = `{"type":"result","subtype":"success","total_cost_usd":0.0234,"duration_ms":12345,"duration_api_ms":9800,"num_turns":3,"usage":{"input_tokens":1500,"cache_creation_input_tokens":200,"cache_read_input_tokens":50,"output_tokens":400},"permission_denials":[],"session_id":"ses_abc123"}`

func TestParseResultLine_ValidResult(t *testing.T) {
	log := "some prefix line\n" +
		`{"type":"assistant","content":"hello"}` + "\n" +
		validResultLine + "\n"

	result, err := ParseResultLine(log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Subtype != "success" {
		t.Errorf("subtype: got %q, want %q", result.Subtype, "success")
	}
	if result.TotalCostUSD != 0.0234 {
		t.Errorf("total_cost_usd: got %v, want 0.0234", result.TotalCostUSD)
	}
	if result.DurationMs != 12345 {
		t.Errorf("duration_ms: got %v, want 12345", result.DurationMs)
	}
	if result.DurationApiMs != 9800 {
		t.Errorf("duration_api_ms: got %v, want 9800", result.DurationApiMs)
	}
	if result.NumTurns != 3 {
		t.Errorf("num_turns: got %v, want 3", result.NumTurns)
	}
	if result.Usage.InputTokens != 1500 {
		t.Errorf("usage.input_tokens: got %v, want 1500", result.Usage.InputTokens)
	}
	if result.Usage.CacheCreationInputTokens != 200 {
		t.Errorf("usage.cache_creation_input_tokens: got %v, want 200", result.Usage.CacheCreationInputTokens)
	}
	if result.Usage.CacheReadInputTokens != 50 {
		t.Errorf("usage.cache_read_input_tokens: got %v, want 50", result.Usage.CacheReadInputTokens)
	}
	if result.Usage.OutputTokens != 400 {
		t.Errorf("usage.output_tokens: got %v, want 400", result.Usage.OutputTokens)
	}
	if result.SessionID != "ses_abc123" {
		t.Errorf("session_id: got %q, want %q", result.SessionID, "ses_abc123")
	}
}

func TestParseResultLine_NoResultLine(t *testing.T) {
	log := `{"type":"assistant","content":"hello"}` + "\n" +
		`{"type":"user","content":"world"}` + "\n"

	result, err := ParseResultLine(log)
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
	if err == nil {
		t.Error("expected non-nil error")
	}
}

func TestParseResultLine_MalformedJSON(t *testing.T) {
	// A line that contains "type":"result" but is invalid JSON.
	log := `{"type":"assistant","content":"hello"}` + "\n" +
		`{"type":"result","broken":` + "\n"

	result, err := ParseResultLine(log)
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
	if err == nil {
		t.Error("expected non-nil error for malformed JSON")
	}
}

func TestParseResultLine_ResultNotLastLine(t *testing.T) {
	// Result line appears mid-log; trailing lines are non-result.
	log := "first line\n" +
		validResultLine + "\n" +
		`{"type":"system","content":"cleanup"}` + "\n" +
		`{"type":"system","content":"done"}` + "\n"

	result, err := ParseResultLine(log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Subtype != "success" {
		t.Errorf("subtype: got %q, want %q", result.Subtype, "success")
	}
}

func TestParseResultLine_EmptyLog(t *testing.T) {
	result, err := ParseResultLine("")
	if result != nil {
		t.Errorf("expected nil result for empty log, got %+v", result)
	}
	if err == nil {
		t.Error("expected non-nil error for empty log")
	}
}

func TestParseResultLine_ZeroUsage(t *testing.T) {
	line := `{"type":"result","subtype":"success","total_cost_usd":0,"duration_ms":0,"duration_api_ms":0,"num_turns":0,"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0},"permission_denials":[],"session_id":""}`
	result, err := ParseResultLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Usage.InputTokens != 0 || result.Usage.OutputTokens != 0 {
		t.Errorf("expected all zero usage, got %+v", result.Usage)
	}
	if result.TotalCostUSD != 0 {
		t.Errorf("expected zero cost, got %v", result.TotalCostUSD)
	}
}

func TestParseResultLine_PermissionDenials(t *testing.T) {
	line := `{"type":"result","subtype":"success","total_cost_usd":0,"duration_ms":0,"duration_api_ms":0,"num_turns":1,"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":50},"permission_denials":[{"tool":"bash","reason":"blocked"},{"tool":"write","reason":"path_denied"}],"session_id":"ses_xyz"}`

	result, err := ParseResultLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.PermissionDenials) != 2 {
		t.Fatalf("expected 2 permission_denials, got %d", len(result.PermissionDenials))
	}

	// Verify the raw JSON is preserved by re-decoding the first entry.
	var first map[string]string
	if err := json.Unmarshal(result.PermissionDenials[0], &first); err != nil {
		t.Fatalf("could not unmarshal first permission_denial: %v", err)
	}
	if first["tool"] != "bash" {
		t.Errorf("first denial tool: got %q, want %q", first["tool"], "bash")
	}
}
