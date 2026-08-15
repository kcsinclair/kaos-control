// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 4 (FR-6, FR-7, FR-8, FR-9, OQ-4): refresh patches the six
// standard agents' allowed_write_paths and build/lint/test commands for the
// promoted stack, every standard agent's prompt carries the
// architecture-awareness clause, unrelated config is untouched, the patched
// file reloads via config.LoadProject, and a static-html-js promotion
// disables backend-developer.
//
// designBuildAgents, findAgentConfig, combinedPromptText,
// hasReadArchitectureDirective, and hasProposeADRDirective are shared
// helpers from architecture_directives_test.go (same package).

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

func TestDirectivesConfigPatch_GoVue_TunesStandardAgents(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)

	cfg, err := config.LoadProject(env.projectRoot)
	if err != nil {
		t.Fatalf("patched config does not reload via config.LoadProject: %v", err)
	}

	be := findAgentConfig(cfg, "backend-developer")
	if be == nil {
		t.Fatal("backend-developer agent missing")
	}
	for _, want := range []string{"internal", "cmd", "lifecycle/backend-plans", "lifecycle/architecture/decisions"} {
		found := false
		for _, p := range be.AllowedPaths {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("backend-developer.AllowedPaths missing %q: %v", want, be.AllowedPaths)
		}
	}
	if !strings.Contains(combinedPromptText(be), "go build ./...") {
		t.Errorf("backend-developer prompt missing build token:\n%s", combinedPromptText(be))
	}

	fe := findAgentConfig(cfg, "frontend-developer")
	if fe == nil {
		t.Fatal("frontend-developer agent missing")
	}
	if !strings.Contains(combinedPromptText(fe), "pnpm build") {
		t.Errorf("frontend-developer prompt missing build token:\n%s", combinedPromptText(fe))
	}

	for _, name := range designBuildAgents {
		ag := findAgentConfig(cfg, name)
		if ag == nil {
			t.Fatalf("agent %q missing after patch", name)
		}
		text := combinedPromptText(ag)
		if !hasReadArchitectureDirective(text) {
			t.Errorf("agent %q missing a \"read lifecycle/architecture/\" clause (FR-8)", name)
		}
		if !hasProposeADRDirective(text) {
			t.Errorf("agent %q missing a \"propose an ADR\" clause (FR-8)", name)
		}
	}
}

func TestDirectivesConfigPatch_UnrelatedBlocksUntouched(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)

	raw := string(readFileMust(t, filepath.Join(env.projectRoot, "lifecycle", "config.yaml")))
	if !strings.Contains(raw, "A hand-written custom agent prompt that must never be touched.") {
		t.Error("custom agent prompt_templates content was modified")
	}
	if !strings.Contains(raw, "email: qa@test.local") {
		t.Error("users: block was modified")
	}

	idx := strings.Index(raw, "my-custom-agent")
	if idx < 0 {
		t.Fatal("custom agent missing entirely")
	}
	if !strings.Contains(raw[idx:], "lifecycle/ideas") {
		t.Error("custom agent allowed_write_paths was modified")
	}
}

func TestDirectivesConfigPatch_Idempotent_SecondPatchIsNoOp(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)
	cfgPath := filepath.Join(env.projectRoot, "lifecycle", "config.yaml")
	before := string(readFileMust(t, cfgPath))

	refreshDirectives(t, env, false)
	after := string(readFileMust(t, cfgPath))

	if before != after {
		t.Error("expected lifecycle/config.yaml to be byte-identical after a second refresh with the same selection (NFR-2)")
	}
}

func TestDirectivesConfigPatch_StaticSite_DisablesBackendDeveloper(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "static-html-js.md")}, directivesCfgYAML)
	data := refreshDirectives(t, env, false)

	disabled, _ := data["disabledAgents"].([]any)
	found := false
	for _, d := range disabled {
		if d == "backend-developer" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected backend-developer in disabledAgents, got %v", disabled)
	}

	raw := string(readFileMust(t, filepath.Join(env.projectRoot, "lifecycle", "config.yaml")))
	idx := strings.Index(raw, "name: backend-developer")
	if idx < 0 {
		t.Fatal("backend-developer agent missing")
	}
	block := raw[idx:]
	if next := strings.Index(block, "\n  - name:"); next > 0 {
		block = block[:next]
	}
	if !strings.Contains(block, "enabled: false") {
		t.Errorf("expected enabled: false in the backend-developer block:\n%s", block)
	}

	if _, err := config.LoadProject(env.projectRoot); err != nil {
		t.Fatalf("patched config does not reload: %v", err)
	}
}
