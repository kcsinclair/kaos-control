// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 2 (FR-1, FR-3, FR-4, FR-5, NFR-2): POST .../directives/refresh
// renders the AGENTS.md-primary directive set — canonical AGENTS.md with
// the standing content, @AGENTS.md pointers for CLAUDE.md/GEMINI.md,
// stack-aware layout/commands — and a second run is a byte-identical no-op.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectivesGenerate_GoVue_RendersFullSetWithStandingContent(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)

	data := refreshDirectives(t, env, false)

	for _, want := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		fw := fileWriteByPath(t, data, want)
		if created, _ := fw["created"].(bool); !created {
			t.Errorf("%s: expected created=true, got %v", want, fw)
		}
	}
	if skipped, _ := data["skipped"].([]any); len(skipped) != 0 {
		t.Errorf("expected nothing skipped (qa runs on gemini-cli), got %v", skipped)
	}

	agentsBody := string(readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md")))
	if !strings.HasPrefix(agentsBody, "<!-- kaos-control:generated:start -->") {
		t.Error("AGENTS.md missing managed-region start marker")
	}
	if !strings.HasSuffix(strings.TrimRight(agentsBody, "\n"), "<!-- kaos-control:generated:end -->") {
		t.Error("AGENTS.md missing managed-region end marker")
	}
	for _, want := range []string{
		"internal/", "cmd/", "web/src/", // FR-4(a) repo layout
		"Lineage Filename Convention", "monotonic", // FR-4(b)
		"Frontmatter Requirements", "type vocabulary", // FR-4(c) (case-insensitive check below)
		"Commit Conventions",      // FR-4(d)
		"Agent Roles",             // FR-4(e)
		"lifecycle/architecture/", // FR-4(f) required-reading pointer
	} {
		if !strings.Contains(strings.ToLower(agentsBody), strings.ToLower(want)) {
			t.Errorf("AGENTS.md missing %q:\n%s", want, agentsBody)
		}
	}

	claudeBody := string(readFileMust(t, filepath.Join(env.projectRoot, "CLAUDE.md")))
	if claudeBody != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q, want the bare @AGENTS.md pointer", claudeBody)
	}
	geminiBody := string(readFileMust(t, filepath.Join(env.projectRoot, "GEMINI.md")))
	if geminiBody != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md: got %q, want the bare @AGENTS.md pointer", geminiBody)
	}
}

func TestDirectivesGenerate_NonGoVueStack_UsesThatStacksLayout(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "python-fastapi.md")}, directivesCfgYAML)

	refreshDirectives(t, env, false)

	agentsBody := string(readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md")))
	if strings.Contains(agentsBody, "internal/") || strings.Contains(agentsBody, "cmd/") || strings.Contains(agentsBody, "web/src/") {
		t.Errorf("expected no Go+Vue layout in a python-fastapi AGENTS.md:\n%s", agentsBody)
	}
	for _, want := range []string{"app/", "uvicorn", "pytest"} {
		if !strings.Contains(agentsBody, want) {
			t.Errorf("AGENTS.md missing python-fastapi-specific detail %q:\n%s", want, agentsBody)
		}
	}
}

func TestDirectivesGenerate_Idempotent_SecondRunIsNoOp(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)

	refreshDirectives(t, env, false)
	firstAgents := readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md"))

	data2 := refreshDirectives(t, env, false)

	files, _ := data2["files"].([]any)
	if len(files) == 0 {
		t.Fatal("expected FileWrite entries on the second run")
	}
	for _, f := range files {
		fw, _ := f.(map[string]any)
		if created, _ := fw["created"].(bool); created {
			t.Errorf("second run: expected created=false, got %+v", fw)
		}
		if changed, _ := fw["changed"].(bool); changed {
			t.Errorf("second run: expected changed=false, got %+v", fw)
		}
		if skipped, _ := fw["skipped"].(bool); !skipped {
			t.Errorf("second run: expected skipped=true (no-op), got %+v", fw)
		}
	}

	secondAgents := readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md"))
	if string(firstAgents) != string(secondAgents) {
		t.Error("expected AGENTS.md to be byte-identical across two runs with the same selection (NFR-2)")
	}
}
