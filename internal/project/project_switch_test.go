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

func TestSwitchAgentProvider_ManualSwitch(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "manual test", false); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}

	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "gemini-cloud" || ag.Model != "gemini-2.5-flash" {
		t.Fatalf("expected switched provider/model, got %+v", ag)
	}
	if ag.PrimaryProvider != "" {
		t.Errorf("manual switch (isFailover=false) should not stash a primary, got %q", ag.PrimaryProvider)
	}

	raw, err := os.ReadFile(filepath.Join(p.Entry.Path, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "provider: gemini-cloud") {
		t.Error("expected disk to reflect the switched provider")
	}

	commits, err := p.Git.Log("lifecycle/config.yaml", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 || !strings.HasPrefix(commits[0].Message, "switch(agent): analyst-agent -> gemini-cloud/gemini-2.5-flash") {
		t.Fatalf("expected a switch(agent) commit, got %+v", commits)
	}
}

func TestSwitchAgentProvider_FailoverStashesPrimary(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "HTTP 529 Overloaded", true); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}

	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "gemini-cloud" || ag.Model != "gemini-2.5-flash" {
		t.Fatalf("expected switched provider/model, got %+v", ag)
	}
	if ag.PrimaryProvider != "anthropic-cloud" || ag.PrimaryModel != "claude-3-7-sonnet" {
		t.Fatalf("expected primary stashed to original provider/model, got %+v", ag)
	}

	commits, err := p.Git.Log("lifecycle/config.yaml", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 || !strings.HasPrefix(commits[0].Message, "failover(agent): analyst-agent anthropic-cloud -> gemini-cloud") {
		t.Fatalf("expected a failover(agent) commit, got %+v", commits)
	}
}

func TestRestoreAgentProvider(t *testing.T) {
	p := openTestProjectWithGit(t)

	if err := p.SwitchAgentProvider("analyst-agent", "gemini-cloud", "gemini-2.5-flash", "test", true); err != nil {
		t.Fatalf("SwitchAgentProvider: %v", err)
	}
	if err := p.RestoreAgentProvider("analyst-agent"); err != nil {
		t.Fatalf("RestoreAgentProvider: %v", err)
	}

	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Fatalf("expected restored to primary provider/model, got %+v", ag)
	}
	if ag.PrimaryProvider != "" || ag.PrimaryModel != "" {
		t.Fatalf("expected primary fields cleared after restore, got %+v", ag)
	}

	commits, err := p.Git.Log("lifecycle/config.yaml", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 || !strings.HasPrefix(commits[0].Message, "restore(agent): analyst-agent restored to anthropic-cloud") {
		t.Fatalf("expected a restore(agent) commit, got %+v", commits)
	}
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

	ag := agentByName(t, p, "analyst-agent")
	if ag.Provider != "gemini-cloud" || ag.Model != "gemini-2.5-flash" {
		t.Fatalf("expected template-applied provider/model, got %+v", ag)
	}

	commits, err := p.Git.Log("lifecycle/config.yaml", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 || !strings.HasPrefix(commits[0].Message, "template(provider): applied local-ai") {
		t.Fatalf("expected a template(provider) commit, got %+v", commits)
	}
}

func TestApplyProviderTemplate_UnknownTemplate(t *testing.T) {
	p := openTestProjectWithGit(t)

	_, err := p.ApplyProviderTemplate("does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected unknown-template error, got: %v", err)
	}
}
