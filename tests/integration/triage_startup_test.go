// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
	"time"
)

// TestTriageStartup_SingleRawIdea verifies that a pre-existing raw idea
// (written before project.Open) is picked up by the startup re-scan and
// triaged within 5 s.
func TestTriageStartup_SingleRawIdea(t *testing.T) {
	installLLMFake(t, []string{defaultProposeJSON("foo", "Foo Idea", nil)})

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/foo.md",
			content: makeArtifact("Foo Idea", "idea", "raw", "foo", "",
				"This is the foo idea with enough words to qualify for automatic triage."),
		},
	}
	env := newTriageTestEnvWithSeeds(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	if !pollForArtifactStatus(t, env, "lifecycle/ideas/foo.md", "draft", 5*time.Second) {
		fm := readArtifactFM(t, env.projectRoot, "lifecycle/ideas/foo.md")
		t.Fatalf("startup-triaged idea not draft within 5s; current fm: %v", fm)
	}
}

// TestTriageStartup_EmptyIdeasDir verifies that an empty lifecycle/ideas/
// directory at startup does not insert any agent_runs rows.
func TestTriageStartup_EmptyIdeasDir(t *testing.T) {
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	// Let startup re-scan complete.
	time.Sleep(500 * time.Millisecond)

	runs, err := env.proj.Idx.ListAgentRuns("", 100)
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(runs) > 0 {
		t.Errorf("expected no agent runs for empty ideas dir, got %d", len(runs))
	}
}

// TestTriageStartup_MultipleRawWithCap verifies that multiple pre-existing raw
// ideas are all eventually triaged even when MaxConcurrent caps concurrency.
func TestTriageStartup_MultipleRawWithCap(t *testing.T) {
	installLLMFake(t, []string{
		defaultProposeJSON("bar1", "Bar One", nil),
		defaultProposeJSON("bar2", "Bar Two", nil),
		defaultProposeJSON("bar3", "Bar Three", nil),
	})

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bar1.md",
			content: makeArtifact("Bar One", "idea", "raw", "bar1", "",
				"This is bar one idea with enough words to qualify for triage."),
		},
		{
			relPath: "lifecycle/ideas/bar2.md",
			content: makeArtifact("Bar Two", "idea", "raw", "bar2", "",
				"This is bar two idea with enough words to qualify for triage."),
		},
		{
			relPath: "lifecycle/ideas/bar3.md",
			content: makeArtifact("Bar Three", "idea", "raw", "bar3", "",
				"This is bar three idea with enough words to qualify for triage."),
		},
	}
	env := newTriageTestEnvWithSeeds(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// All three must become draft within the timeout.
	deadline := time.Now().Add(10 * time.Second)
	paths := []string{
		"lifecycle/ideas/bar1.md",
		"lifecycle/ideas/bar2.md",
		"lifecycle/ideas/bar3.md",
	}
	for _, p := range paths {
		timeout := time.Until(deadline)
		if timeout <= 0 {
			t.Fatalf("timeout waiting for triage of %s", p)
		}
		if !pollForArtifactStatus(t, env, p, "draft", timeout) {
			fm := readArtifactFM(t, env.projectRoot, p)
			t.Errorf("idea %s not triaged to draft within timeout; current fm: %v", p, fm)
		}
	}
}

// TestTriageStartup_AlreadyDraftNoRuns verifies that pre-existing draft
// artifacts do not trigger triage on startup.
func TestTriageStartup_AlreadyDraftNoRuns(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/already1.md",
			content: makeArtifact("Already Draft One", "idea", "draft", "already1", "",
				"This is already-draft idea one with enough words for testing."),
		},
		{
			relPath: "lifecycle/ideas/already2.md",
			content: makeArtifact("Already Draft Two", "idea", "draft", "already2", "",
				"This is already-draft idea two with enough words for testing."),
		},
	}
	env := newTriageTestEnvWithSeeds(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Let startup re-scan complete.
	time.Sleep(1 * time.Second)

	for _, path := range []string{
		"lifecycle/ideas/already1.md",
		"lifecycle/ideas/already2.md",
	} {
		runs, err := env.proj.Idx.ListAgentRunsByTargetPath(path)
		if err != nil {
			t.Fatalf("ListAgentRunsByTargetPath(%s): %v", path, err)
		}
		if len(runs) > 0 {
			t.Errorf("expected no runs for already-draft idea %s, got %d", path, len(runs))
		}
	}
}
