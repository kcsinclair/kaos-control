// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"encoding/json"
	"os"
	"testing"
)

// TestWriteHookSettings_MatcherSchema verifies that the generated settings.json
// uses Claude Code's matcher-wrapped PreToolUse schema. Without the matcher
// wrapper, Claude silently ignores the hook config and falls back to
// interactive-prompt mode (which fails in headless -p runs).
func TestWriteHookSettings_MatcherSchema(t *testing.T) {
	dir := t.TempDir()
	path, cleanup, err := WriteHookSettings(dir, "/usr/local/bin/kaos-control", "127.0.0.1:9600", "run-abc")
	if err != nil {
		t.Fatalf("WriteHookSettings: %v", err)
	}
	defer cleanup()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}

	hooks, _ := got["hooks"].(map[string]any)
	if hooks == nil {
		t.Fatal("missing top-level hooks map")
	}
	preToolUse, _ := hooks["PreToolUse"].([]any)
	if len(preToolUse) != 1 {
		t.Fatalf("PreToolUse: expected 1 matcher entry, got %d", len(preToolUse))
	}
	entry, _ := preToolUse[0].(map[string]any)
	if entry["matcher"] != "*" {
		t.Errorf("matcher: want %q, got %v", "*", entry["matcher"])
	}
	nested, _ := entry["hooks"].([]any)
	if len(nested) != 1 {
		t.Fatalf("nested hooks: expected 1 entry, got %d", len(nested))
	}
	hook, _ := nested[0].(map[string]any)
	if hook["type"] != "command" {
		t.Errorf("hook.type: want %q, got %v", "command", hook["type"])
	}
	cmd, _ := hook["command"].(string)
	if cmd == "" || !contains(cmd, "hook-helper") || !contains(cmd, "run-abc") {
		t.Errorf("hook.command unexpected: %q", cmd)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
