// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/config"
)

const failoverWideTestConfig = `stages:
  - {name: ideas, dir: ideas}
git:
  default_branch: main
roles:
  - analyst
agents:
  - name: agent-with-fallback
    role: [analyst]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    prompt_templates:
      analyst: "x"
  - name: agent-no-fallback
    role: [analyst]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    prompt_templates:
      analyst: "x"
  - name: agent-other-provider
    role: [analyst]
    driver: openai-compatible
    provider: gemini-cloud
    model: gemini-2.5-flash
    prompt_templates:
      analyst: "x"
`

func openFailoverWideTestProject(t *testing.T) *Project {
	t.Helper()
	dir := t.TempDir()

	gr, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lifecycle", "config.yaml"), []byte(failoverWideTestConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := gr.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("lifecycle/config.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	entry := &config.ProjectEntry{Name: "failover-wide-test", Path: dir}
	dbDir := t.TempDir()
	p, err := Open(entry, dbDir, OpenOptions{SkipArchitectureScaffold: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func TestFailoverProviderWide_SwitchesAndPartialPauses(t *testing.T) {
	p := openFailoverWideTestProject(t)

	switched, noSecondary, err := p.FailoverProviderWide("anthropic-cloud", "HTTP 529 Overloaded", 1234, "five_hour")
	if err != nil {
		t.Fatalf("FailoverProviderWide: %v", err)
	}
	if len(switched) != 1 || switched[0] != "agent-with-fallback" {
		t.Errorf("expected agent-with-fallback switched, got %v", switched)
	}
	if len(noSecondary) != 1 || noSecondary[0] != "agent-no-fallback" {
		t.Errorf("expected agent-no-fallback partially paused, got %v", noSecondary)
	}

	// The unrelated agent on a different provider is untouched.
	if _, ok := p.Operations().AgentState("agent-other-provider"); ok {
		t.Error("expected agent-other-provider untouched")
	}

	switchedState, ok := p.Operations().AgentState("agent-with-fallback")
	if !ok {
		t.Fatal("expected operations state for agent-with-fallback")
	}
	if switchedState.Active.Provider != "gemini-cloud" || switchedState.Active.Model != "gemini-2.5-flash" {
		t.Errorf("expected switched to gemini-cloud/gemini-2.5-flash, got %+v", switchedState)
	}
	if switchedState.ResetsAtUnix != 1234 || switchedState.Bucket != "five_hour" {
		t.Errorf("expected resets_at_unix/bucket recorded, got %+v", switchedState)
	}
	if !p.IsAgentFailedOver("agent-with-fallback") {
		t.Error("expected agent-with-fallback IsAgentFailedOver true")
	}

	pausedState, ok := p.Operations().AgentState("agent-no-fallback")
	if !ok {
		t.Fatal("expected operations state for agent-no-fallback")
	}
	if !pausedState.PartialPause {
		t.Error("expected agent-no-fallback PartialPause true")
	}
	if pausedState.Active.Provider != "anthropic-cloud" {
		t.Errorf("expected partial-pause agent to stay on its (unreachable) primary, got %+v", pausedState)
	}
	if p.IsAgentFailedOver("agent-no-fallback") {
		t.Error("a partial pause is not a failover (Active == Primary)")
	}

	paused := p.PartiallyPausedAgents()
	if len(paused) != 1 || paused[0] != "agent-no-fallback" {
		t.Errorf("expected PartiallyPausedAgents = [agent-no-fallback], got %v", paused)
	}

	provider, model, ok := p.EffectiveAgentProvider("agent-with-fallback")
	if !ok || provider != "gemini-cloud" || model != "gemini-2.5-flash" {
		t.Errorf("EffectiveAgentProvider: got (%q, %q, %v)", provider, model, ok)
	}

	// lifecycle/config.yaml is never mutated by project-wide failover.
	assertConfigYAMLUnchangedAndNoNewCommits2(t, p, failoverWideTestConfig)
}

func TestFailoverProviderWide_OneLevelCap(t *testing.T) {
	p := openFailoverWideTestProject(t)

	// First failover: agent-with-fallback moves to gemini-cloud.
	if _, _, err := p.FailoverProviderWide("anthropic-cloud", "first failure", 0, ""); err != nil {
		t.Fatal(err)
	}
	if !p.IsAgentFailedOver("agent-with-fallback") {
		t.Fatal("expected agent-with-fallback failed over after first call")
	}

	// A second failover attempt against the NEW active provider
	// (gemini-cloud) must not move an already-failed-over agent again
	// (NFR-6 one-level cap) — it is simply skipped, left on its secondary.
	// (agent-other-provider is also bound to gemini-cloud in the fixture and
	// has no fallback, so it legitimately ends up partially paused here —
	// the assertion below only cares that agent-with-fallback is untouched.)
	switched, _, err := p.FailoverProviderWide("gemini-cloud", "second failure", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range switched {
		if name == "agent-with-fallback" {
			t.Fatalf("expected the one-level cap to skip the already-failed-over agent, got switched=%v", switched)
		}
	}

	state, ok := p.Operations().AgentState("agent-with-fallback")
	if !ok || state.Active.Provider != "gemini-cloud" {
		t.Errorf("expected agent-with-fallback to remain on gemini-cloud, got %+v", state)
	}
}

// assertConfigYAMLUnchangedAndNoNewCommits2 mirrors
// assertConfigYAMLUnchangedAndNoNewCommits (project_switch_test.go) but
// takes the expected content explicitly, since this file's fixture differs.
func assertConfigYAMLUnchangedAndNoNewCommits2(t *testing.T, p *Project, want string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(p.Entry.Path, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != want {
		t.Error("expected lifecycle/config.yaml to be byte-for-byte unchanged")
	}
	commits, err := p.Git.Log("lifecycle/config.yaml", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected only the initial commit to lifecycle/config.yaml, got %d: %+v", len(commits), commits)
	}
}
