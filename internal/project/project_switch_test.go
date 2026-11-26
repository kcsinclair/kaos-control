// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/config"
)

const providerSwitchTestConfig = `stages:
  - {name: ideas, dir: ideas}
git:
  default_branch: main
roles:
  - analyst
agents:
  - name: analyst-agent
    role: [analyst]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: local-ollama
    fallback_model: llama3
    prompt_templates:
      analyst: "x"
provider_templates:
  - name: local-ai
    agents:
      analyst-agent: {provider: gemini-cloud, model: gemini-2.5-flash}
`

// openTestProjectWithGit creates a real git repo with an initial commit and
// a lifecycle/config.yaml containing an agent configured for provider
// switching, then opens it via project.Open.
func openTestProjectWithGit(t *testing.T) *Project {
	t.Helper()
	dir := t.TempDir()

	gr, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lifecycle", "config.yaml"), []byte(providerSwitchTestConfig), 0o644); err != nil {
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

	entry := &config.ProjectEntry{Name: "switch-test", Path: dir}
	dbDir := t.TempDir()
	p, err := Open(entry, dbDir, OpenOptions{SkipArchitectureScaffold: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func agentByName(t *testing.T, p *Project, name string) config.AgentConfig {
	t.Helper()
	ag, ok := findAgentConfig(p.Config(), name)
	if !ok {
		t.Fatalf("agent %q not found", name)
	}
	return ag
}

// assertConfigYAMLUnchangedAndNoNewCommits confirms the central design shift
// invariant: no switch/restore/template-apply operation writes
// lifecycle/config.yaml or creates a git commit — declared intent is
// untouched and all live state lives in operations.yaml.
func assertConfigYAMLUnchangedAndNoNewCommits(t *testing.T, p *Project) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(p.Entry.Path, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != providerSwitchTestConfig {
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

func TestSwitchAgentProvider_ManualSwitch(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "manual test", false); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}

	// Declared config is untouched — it remains the source of primary intent.
	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Fatalf("lifecycle/config.yaml must not be mutated by a switch, got %+v", ag)
	}

	state, ok := p.Operations().AgentState("analyst-agent")
	if !ok {
		t.Fatal("expected operations.yaml to record the active override")
	}
	if state.Active.Provider != "gemini-cloud" || state.Active.Model != "gemini-2.5-flash" {
		t.Fatalf("expected switched active provider/model, got %+v", state)
	}
	if state.Primary.Provider != "anthropic-cloud" {
		t.Errorf("expected primary snapshotted from declared config, got %q", state.Primary.Provider)
	}

	assertConfigYAMLUnchangedAndNoNewCommits(t, p)
}

func TestSwitchAgentProvider_FailoverStashesPrimary(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "HTTP 529 Overloaded", true); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}

	state, ok := p.Operations().AgentState("analyst-agent")
	if !ok {
		t.Fatal("expected operations.yaml to record the active override")
	}
	if state.Active.Provider != "gemini-cloud" || state.Active.Model != "gemini-2.5-flash" {
		t.Fatalf("expected switched active provider/model, got %+v", state)
	}
	if state.Primary.Provider != "anthropic-cloud" || state.Primary.Model != "claude-3-7-sonnet" {
		t.Fatalf("expected primary stashed to original provider/model, got %+v", state)
	}
	if !state.IsFailedOver() {
		t.Error("expected IsFailedOver() true")
	}

	hist := p.Operations().HistorySnapshot()
	if len(hist) != 1 || hist[0].Action != "failover" {
		t.Fatalf("expected one failover history entry, got %+v", hist)
	}

	assertConfigYAMLUnchangedAndNoNewCommits(t, p)
}

func TestRestoreAgentProvider(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "test", true); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}
	if err := p.RestoreAgentProvider("analyst-agent"); err != nil {
		t.Fatalf("RestoreAgentProvider: %v", err)
	}

	if _, ok := p.Operations().AgentState("analyst-agent"); ok {
		t.Fatal("expected operations.yaml override cleared after restore")
	}

	// Declared config was never touched, so it already reads as the primary.
	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Fatalf("expected declared config to still read primary provider/model, got %+v", ag)
	}

	hist := p.Operations().HistorySnapshot()
	if len(hist) != 2 || hist[1].Action != "restore" {
		t.Fatalf("expected failover then restore history entries, got %+v", hist)
	}

	assertConfigYAMLUnchangedAndNoNewCommits(t, p)
}

func TestRestoreAgentProvider_NotInFailover(t *testing.T) {
	p := openTestProjectWithGit(t)

	err := p.RestoreAgentProvider("analyst-agent")
	if err == nil || !strings.Contains(err.Error(), "not currently in a failover state") {
		t.Fatalf("expected not-in-failover error, got: %v", err)
	}
}

func TestApplyProviderTemplate(t *testing.T) {
	p := openTestProjectWithGit(t)

	n, err := p.ApplyProviderTemplate("local-ai")
	if err != nil {
		t.Fatalf("ApplyProviderTemplate: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 agent updated, got %d", n)
	}

	state, ok := p.Operations().AgentState("analyst-agent")
	if !ok {
		t.Fatal("expected operations.yaml to record the template-applied override")
	}
	if state.Active.Provider != "gemini-cloud" || state.Active.Model != "gemini-2.5-flash" {
		t.Fatalf("expected template-applied provider/model, got %+v", state)
	}

	assertConfigYAMLUnchangedAndNoNewCommits(t, p)
}

func TestApplyProviderTemplate_UnknownTemplate(t *testing.T) {
	p := openTestProjectWithGit(t)

	_, err := p.ApplyProviderTemplate("does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected unknown-template error, got: %v", err)
	}
}
