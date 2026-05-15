// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestBuildArgs_DualFlagOrder verifies that --permission-mode bypassPermissions
// appears before --dangerously-skip-permissions in the argument list.
func TestBuildArgs_DualFlagOrder(t *testing.T) {
	d := &ClaudeCodeDriver{}
	args := d.buildArgs(Run{PromptText: "hello"})

	pmIdx := slices.Index(args, "--permission-mode")
	if pmIdx < 0 {
		t.Fatal("--permission-mode not found in args")
	}
	if pmIdx+1 >= len(args) || args[pmIdx+1] != "bypassPermissions" {
		t.Fatalf("expected --permission-mode bypassPermissions, got args[%d+1]=%q", pmIdx, args[pmIdx+1])
	}

	dspIdx := slices.Index(args, "--dangerously-skip-permissions")
	if dspIdx < 0 {
		t.Fatal("--dangerously-skip-permissions not found in args")
	}

	if pmIdx >= dspIdx {
		t.Errorf("--permission-mode (%d) must appear before --dangerously-skip-permissions (%d)", pmIdx, dspIdx)
	}
}

// TestExtractRateLimitText covers both stream-json rate-limit formats.
func TestExtractRateLimitText(t *testing.T) {
	type tc struct {
		name    string
		rawJSON string
		wantOK  bool
		wantTxt string
	}

	// Helper: wrap event JSON in an agent.progress payload map.
	wrap := func(evJSON string) map[string]any {
		var ev map[string]any
		_ = json.Unmarshal([]byte(evJSON), &ev)
		return map[string]any{"event": ev}
	}

	cases := []tc{
		{
			name:    "format1 top-level error string",
			rawJSON: `{"error":"rate_limit","message":{"content":[{"type":"text","text":"rate limit hit; resets 8pm (Australia/Brisbane)"}]}}`,
			wantOK:  true,
			wantTxt: "rate limit hit; resets 8pm (Australia/Brisbane)",
		},
		{
			name:    "format2 nested error object",
			rawJSON: `{"type":"error","error":{"type":"rate_limit_error","message":"Too many requests; retry after 120 seconds"}}`,
			wantOK:  true,
			// No message.content — falls back to error.message string.
			wantTxt: "Too many requests; retry after 120 seconds",
		},
		{
			name:    "format2 nested with message.content",
			rawJSON: `{"type":"error","error":{"type":"rate_limit_error"},"message":{"content":[{"type":"text","text":"resets 8:00pm (Australia/Brisbane)"}]}}`,
			wantOK:  true,
			wantTxt: "resets 8:00pm (Australia/Brisbane)",
		},
		{
			name:    "non-rate-limit error",
			rawJSON: `{"error":"invalid_request","message":"bad input"}`,
			wantOK:  false,
		},
		{
			name:    "normal result event",
			rawJSON: `{"type":"result","subtype":"success","result":"done"}`,
			wantOK:  false,
		},
		{
			name:    "format3 result is_error out-of-usage with timezone",
			rawJSON: `{"type":"result","subtype":"success","is_error":true,"result":"You're out of extra usage · resets 11:10pm (Australia/Brisbane)"}`,
			wantOK:  true,
			wantTxt: "You're out of extra usage · resets 11:10pm (Australia/Brisbane)",
		},
		{
			name:    "format3 result is_error rate-limit phrasing",
			rawJSON: `{"type":"result","is_error":true,"result":"Rate limit exceeded — try again in 5 minutes"}`,
			wantOK:  true,
			wantTxt: "Rate limit exceeded — try again in 5 minutes",
		},
		{
			name:    "format3 result is_error but non-quota message — not classified as rate limit",
			rawJSON: `{"type":"result","is_error":true,"result":"Internal server error"}`,
			wantOK:  false,
		},
		{
			name:    "empty event",
			rawJSON: `{}`,
			wantOK:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := wrap(tc.rawJSON)
			text, ok := extractRateLimitText(payload)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOK)
			}
			if tc.wantOK && text != tc.wantTxt {
				t.Errorf("text=%q, want %q", text, tc.wantTxt)
			}
		})
	}
}

// TestBuildArgs_ModelFlag verifies that --model is appended when run.Model is set.
func TestBuildArgs_ModelFlag(t *testing.T) {
	d := &ClaudeCodeDriver{}

	argsWithout := d.buildArgs(Run{PromptText: "x"})
	for i, a := range argsWithout {
		if a == "--model" {
			t.Errorf("unexpected --model flag at index %d when Model is empty", i)
		}
	}

	argsWithModel := d.buildArgs(Run{PromptText: "x", Model: "claude-opus-4-6"})
	mIdx := slices.Index(argsWithModel, "--model")
	if mIdx < 0 {
		t.Fatal("--model not found when Model is set")
	}
	if mIdx+1 >= len(argsWithModel) || argsWithModel[mIdx+1] != "claude-opus-4-6" {
		t.Errorf("expected --model claude-opus-4-6, got %v", argsWithModel[mIdx:])
	}
}
