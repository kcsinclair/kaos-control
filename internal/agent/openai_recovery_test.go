// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"testing"
)

func TestOpenAIRecovery(t *testing.T) {
	t.Run("xml function call single param", func(t *testing.T) {
		input := `I will read the file.
<function=read_file>
<parameter=path>lifecycle/ideas/idea-1.md</parameter>
</function>`
		calls, rem := ParseNativeCalls(input)
		if len(calls) != 1 {
			t.Fatalf("expected 1 recovered call, got %d", len(calls))
		}
		if calls[0].Function.Name != "read_file" {
			t.Errorf("function name: got %q, want %q", calls[0].Function.Name, "read_file")
		}
		var args map[string]string
		if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
			t.Fatalf("unmarshal arguments error: %v", err)
		}
		if args["path"] != "lifecycle/ideas/idea-1.md" {
			t.Errorf("path arg: got %q, want %q", args["path"], "lifecycle/ideas/idea-1.md")
		}
		if rem != "I will read the file." {
			t.Errorf("remaining content: got %q, want %q", rem, "I will read the file.")
		}
	})

	t.Run("xml function call multiple params", func(t *testing.T) {
		input := `<function=write_file>
<parameter=path>lifecycle/requirements/req-2.md</parameter>
<parameter=content># Requirement Content</parameter>
</function>`
		calls, rem := ParseNativeCalls(input)
		if len(calls) != 1 {
			t.Fatalf("expected 1 recovered call, got %d", len(calls))
		}
		if calls[0].Function.Name != "write_file" {
			t.Errorf("function name: got %q, want %q", calls[0].Function.Name, "write_file")
		}
		var args map[string]string
		if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
			t.Fatalf("unmarshal arguments error: %v", err)
		}
		if args["path"] != "lifecycle/requirements/req-2.md" || args["content"] != "# Requirement Content" {
			t.Errorf("args mismatch: %+v", args)
		}
		if rem != "" {
			t.Errorf("remaining content: got %q, want empty", rem)
		}
	})

	t.Run("json tool call tag format", func(t *testing.T) {
		input := `Thinking...
<tool_call>
{"name": "grep", "arguments": {"pattern": "TODO", "path": "internal"}}
</tool_call>
Let me check that.`
		calls, rem := ParseNativeCalls(input)
		if len(calls) != 1 {
			t.Fatalf("expected 1 recovered call, got %d", len(calls))
		}
		if calls[0].Function.Name != "grep" {
			t.Errorf("function name: got %q, want %q", calls[0].Function.Name, "grep")
		}
		var args map[string]string
		if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
			t.Fatalf("unmarshal arguments error: %v", err)
		}
		if args["pattern"] != "TODO" || args["path"] != "internal" {
			t.Errorf("args mismatch: %+v", args)
		}
		wantRem := "Thinking...\n\nLet me check that."
		if rem != wantRem {
			t.Errorf("remaining content: got %q, want %q", rem, wantRem)
		}
	})

	t.Run("multiple json tool calls", func(t *testing.T) {
		input := `<tool_call>{"name": "read_file", "arguments": {"path": "a.txt"}}</tool_call>
<tool_call>{"name": "read_file", "arguments": {"path": "b.txt"}}</tool_call>`
		calls, rem := ParseNativeCalls(input)
		if len(calls) != 2 {
			t.Fatalf("expected 2 recovered calls, got %d", len(calls))
		}
		if calls[0].ID != "call_recov_1" || calls[1].ID != "call_recov_2" {
			t.Errorf("IDs: %q, %q", calls[0].ID, calls[1].ID)
		}
		if rem != "" {
			t.Errorf("remaining content: got %q, want empty", rem)
		}
	})

	t.Run("plain message without tool calls", func(t *testing.T) {
		input := "Hello! Here is the response you requested."
		calls, rem := ParseNativeCalls(input)
		if len(calls) != 0 {
			t.Fatalf("expected 0 recovered calls, got %d", len(calls))
		}
		if rem != input {
			t.Errorf("remaining content: got %q, want %q", rem, input)
		}
	})

	t.Run("empty message", func(t *testing.T) {
		calls, rem := ParseNativeCalls("")
		if len(calls) != 0 {
			t.Errorf("expected 0 calls, got %d", len(calls))
		}
		if rem != "" {
			t.Errorf("expected empty remaining content, got %q", rem)
		}
	})
}
