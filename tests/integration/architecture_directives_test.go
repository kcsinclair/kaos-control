// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 6
// (FR-21, FR-22): every design/build agent's prompt template directs it to
// read lifecycle/architecture/ first and to propose an ADR in
// lifecycle/architecture/decisions/ on deviation, and agents permitted to
// author a proposed ADR (the analyst and developer agents named by FR-13,
// not qa) have lifecycle/architecture/decisions in allowed_write_paths.
//
// Concrete directive prose is owned by [[agent-directives-generation]] — this
// test only checks for the presence of the two directive slots, not their
// exact wording, matching the acceptance criteria in
// lifecycle/backend-plans/architectural-artefacts-3-be.md Milestone 5.

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// designBuildAgents are the six agents named by the test plan's Milestone 6.
var designBuildAgents = []string{
	"requirements-analyst", "planning-analyst",
	"backend-developer", "frontend-developer", "test-developer", "qa",
}

// adrAuthoringAgents are the analyst/developer agents FR-13 names as
// permitted to propose an ADR (qa files defects, not ADRs, so it is excluded).
var adrAuthoringAgents = []string{
	"requirements-analyst", "planning-analyst",
	"backend-developer", "frontend-developer", "test-developer",
}

// loadRealProjectConfig loads this repository's own lifecycle/config.yaml via
// the same loader the server uses, resolving the repo root relative to this
// test file so it works regardless of the invoking working directory.
func loadRealProjectConfig(t *testing.T) *config.Project {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	cfg, err := config.LoadProject(repoRoot)
	if err != nil {
		t.Fatalf("loading lifecycle/config.yaml: %v", err)
	}
	return cfg
}

func findAgentConfig(cfg *config.Project, name string) *config.AgentConfig {
	for i := range cfg.Agents {
		if cfg.Agents[i].Name == name {
			return &cfg.Agents[i]
		}
	}
	return nil
}

// combinedPromptText concatenates all of an agent's prompt templates (keyed
// by role) into one string to search for directive text.
func combinedPromptText(ag *config.AgentConfig) string {
	var sb strings.Builder
	for _, tmpl := range ag.PromptTemplates {
		sb.WriteString(tmpl)
		sb.WriteString("\n")
	}
	return sb.String()
}

// hasReadArchitectureDirective reports whether the prompt text directs the
// agent to read lifecycle/architecture/ (FR-21).
func hasReadArchitectureDirective(text string) bool {
	return strings.Contains(text, "lifecycle/architecture/")
}

// hasProposeADRDirective reports whether the prompt text directs the agent to
// propose an ADR under lifecycle/architecture/decisions/ on deviation (FR-22).
func hasProposeADRDirective(text string) bool {
	return strings.Contains(text, "ADR") && strings.Contains(text, "lifecycle/architecture/decisions")
}

// TestAgentDirectives_ReadArchitectureFirst asserts every design/build agent's
// prompt template(s) direct it to read lifecycle/architecture/ before
// substantive work (FR-21).
func TestAgentDirectives_ReadArchitectureFirst(t *testing.T) {
	cfg := loadRealProjectConfig(t)
	for _, name := range designBuildAgents {
		ag := findAgentConfig(cfg, name)
		if ag == nil {
			t.Errorf("agent %q not found in lifecycle/config.yaml", name)
			continue
		}
		text := combinedPromptText(ag)
		if !hasReadArchitectureDirective(text) {
			t.Errorf("agent %q prompt template(s) missing a \"read lifecycle/architecture/\" directive (FR-21)", name)
		}
	}
}

// TestAgentDirectives_ProposeADROnDeviation asserts every design/build agent's
// prompt template(s) direct it to propose an ADR in
// lifecycle/architecture/decisions/ rather than deviate silently (FR-22).
func TestAgentDirectives_ProposeADROnDeviation(t *testing.T) {
	cfg := loadRealProjectConfig(t)
	for _, name := range designBuildAgents {
		ag := findAgentConfig(cfg, name)
		if ag == nil {
			t.Errorf("agent %q not found in lifecycle/config.yaml", name)
			continue
		}
		text := combinedPromptText(ag)
		if !hasProposeADRDirective(text) {
			t.Errorf("agent %q prompt template(s) missing a \"propose an ADR in lifecycle/architecture/decisions/\" directive (FR-22)", name)
		}
	}
}

// TestAgentDirectives_ADRAuthoringWritePath asserts that the analyst/developer
// agents FR-13 permits to author a proposed ADR have
// lifecycle/architecture/decisions in allowed_write_paths.
func TestAgentDirectives_ADRAuthoringWritePath(t *testing.T) {
	cfg := loadRealProjectConfig(t)
	for _, name := range adrAuthoringAgents {
		ag := findAgentConfig(cfg, name)
		if ag == nil {
			t.Errorf("agent %q not found in lifecycle/config.yaml", name)
			continue
		}
		found := false
		for _, p := range ag.AllowedPaths {
			if p == "lifecycle/architecture/decisions" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("agent %q missing \"lifecycle/architecture/decisions\" in allowed_write_paths (FR-13)", name)
		}
	}
}
