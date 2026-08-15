// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 5 (FR-12, NFR-3): GEMINI.md is only written when a
// gemini/gemini-cli driver is configured; skipping it is reported, never
// silent; adding a driver and re-running picks it up.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noGeminiDriverCfgYAML is directivesCfgYAML with the qa agent's driver
// switched from gemini-cli to claude-code-cli, so no agent in the project
// runs on a gemini/gemini-cli driver.
var noGeminiDriverCfgYAML = strings.Replace(directivesCfgYAML, "role: [qa]\n    driver: gemini-cli", "role: [qa]\n    driver: claude-code-cli", 1)

func TestDirectivesSelectivity_NoGeminiDriver_SkipsGeminiMd(t *testing.T) {
	if noGeminiDriverCfgYAML == directivesCfgYAML {
		t.Fatal("fixture setup: failed to remove the gemini-cli driver from directivesCfgYAML")
	}
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, noGeminiDriverCfgYAML)

	data := refreshDirectives(t, env, false)

	files, _ := data["files"].([]any)
	for _, f := range files {
		fw, _ := f.(map[string]any)
		if fw["path"] == "GEMINI.md" {
			t.Fatalf("did not expect a FileWrite for GEMINI.md, got %+v", fw)
		}
	}
	skipped, _ := data["skipped"].([]any)
	if len(skipped) != 1 || skipped[0] != "GEMINI.md" {
		t.Errorf("expected GEMINI.md reported in skipped, got %v", skipped)
	}
	if _, err := os.Stat(filepath.Join(env.projectRoot, "GEMINI.md")); !os.IsNotExist(err) {
		t.Error("GEMINI.md should not have been written (no orphan file, NFR-3)")
	}
}

func TestDirectivesSelectivity_AddingGeminiDriver_EmitsGeminiMdOnRerun(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, noGeminiDriverCfgYAML)
	refreshDirectives(t, env, false)

	if _, err := os.Stat(filepath.Join(env.projectRoot, "GEMINI.md")); !os.IsNotExist(err) {
		t.Fatal("fixture setup: GEMINI.md should not exist yet")
	}

	cfgPath := filepath.Join(env.projectRoot, "lifecycle", "config.yaml")
	raw := string(readFileMust(t, cfgPath))
	withGemini := strings.Replace(raw, "role: [qa]\n    driver: claude-code-cli", "role: [qa]\n    driver: gemini-cli", 1)
	if withGemini == raw {
		t.Fatal("fixture setup: failed to add a gemini-cli driver to the qa agent")
	}
	if err := os.WriteFile(cfgPath, []byte(withGemini), 0o644); err != nil {
		t.Fatal(err)
	}

	data := refreshDirectives(t, env, false)
	if skipped, _ := data["skipped"].([]any); len(skipped) != 0 {
		t.Errorf("expected nothing skipped once a gemini driver is configured, got %v", skipped)
	}
	geminiBody := string(readFileMust(t, filepath.Join(env.projectRoot, "GEMINI.md")))
	if geminiBody != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md: got %q, want the bare @AGENTS.md pointer", geminiBody)
	}
}
