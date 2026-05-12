// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
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
