// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTriageWatcher_CreateRawIdea_TriageRuns verifies that creating a raw idea
// file causes the watcher to trigger triage, which rewrites the artifact and
// transitions its status to draft.
func TestTriageWatcher_CreateRawIdea_TriageRuns(t *testing.T) {
	installLLMFake(t, []string{defaultProposeJSON("alpha", "Alpha Idea", nil)})
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	writeRawIdea(t, env.projectRoot, "alpha",
		"Alpha Idea", "This is the alpha idea body with enough words to qualify.")

	// Wait for triage to complete: poll for status=draft.
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/alpha.md", "draft", 5*time.Second) {
		fm := readArtifactFM(t, env.projectRoot, "lifecycle/ideas/alpha.md")
		t.Fatalf("artifact not triaged to draft within 5s; current fm: %v", fm)
	}

	// Verify body contains expected sections.
	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/alpha.md")
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading artifact: %v", err)
	}
	body := string(content)
	if !strings.Contains(body, "## Raw Idea") {
		t.Error("triaged artifact missing ## Raw Idea section")
	}
	if !strings.Contains(body, "## Idea") {
		t.Error("triaged artifact missing ## Idea section")
	}
}

// TestTriageWatcher_CreateDraftIdea_NoTriage verifies that a draft idea is
// not triaged (only raw ideas are eligible).
func TestTriageWatcher_CreateDraftIdea_NoTriage(t *testing.T) {
	// No LLM fake needed — triage should never run.
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	content := makeArtifact("Draft Idea", "idea", "draft", "draft-idea", "",
		"Draft body with enough words to be considered substantial content for testing.")
	relPath := "lifecycle/ideas/draft-idea.md"
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing draft idea: %v", err)
	}

	// Wait 2s and verify no agent_runs row for this path.
	time.Sleep(2 * time.Second)

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath(relPath)
	if err != nil {
		t.Fatalf("listing runs: %v", err)
	}
	if len(runs) > 0 {
		t.Errorf("expected no agent runs for draft idea, got %d", len(runs))
	}
}

// TestTriageWatcher_CreateRawDefect_NoTriage verifies that a raw file with
// type=defect under lifecycle/ideas/ is not triaged (wrong type).
func TestTriageWatcher_CreateRawDefect_NoTriage(t *testing.T) {
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	content := makeArtifact("Bug Report", "defect", "raw", "bug-report", "",
		"Bug body with enough words to qualify if it were an idea.")
	relPath := "lifecycle/ideas/bug-report.md"
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing raw defect: %v", err)
	}

	time.Sleep(2 * time.Second)

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath(relPath)
	if err != nil {
		t.Fatalf("listing runs: %v", err)
	}
	if len(runs) > 0 {
		t.Errorf("expected no agent runs for raw defect, got %d", len(runs))
	}
}

// TestTriageWatcher_ModifyDraftIdea_NoTriage verifies that modifying a draft
// idea does not trigger triage.
func TestTriageWatcher_ModifyDraftIdea_NoTriage(t *testing.T) {
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	// Pre-create a draft idea.
	relPath := "lifecycle/ideas/existing-draft.md"
	absPath := filepath.Join(env.projectRoot, relPath)
	content := makeArtifact("Existing Draft", "idea", "draft", "existing-draft", "",
		"Pre-existing draft idea body with enough words for testing purposes.")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing draft: %v", err)
	}
	// Let the watcher process the create event first.
	time.Sleep(500 * time.Millisecond)

	// Modify the body.
	updated := makeArtifact("Existing Draft", "idea", "draft", "existing-draft", "",
		"Updated draft idea body with modified content still having enough words.")
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("updating draft: %v", err)
	}

	time.Sleep(2 * time.Second)

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath(relPath)
	if err != nil {
		t.Fatalf("listing runs: %v", err)
	}
	if len(runs) > 0 {
		t.Errorf("expected no agent runs after modifying draft idea, got %d", len(runs))
	}
}

// TestTriageWatcher_RapidWrites_OneRun verifies that two rapid writes within
// the watcher debounce window result in exactly one triage run.
func TestTriageWatcher_RapidWrites_OneRun(t *testing.T) {
	installLLMFake(t, []string{defaultProposeJSON("rapid", "Rapid Idea", nil)})
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	relPath := "lifecycle/ideas/rapid.md"
	absPath := filepath.Join(env.projectRoot, relPath)
	body := "This is a rapid idea with enough words to qualify for triage processing."
	content1 := fmt.Sprintf("---\ntitle: Rapid Idea\ntype: idea\nstatus: raw\nlineage: rapid\n---\n\n%s\n", body)
	content2 := fmt.Sprintf("---\ntitle: Rapid Idea\ntype: idea\nstatus: raw\nlineage: rapid\n---\n\n%s Updated.\n", body)

	if err := os.WriteFile(absPath, []byte(content1), 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	// Second write within 100ms (well inside the 150ms debounce).
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(absPath, []byte(content2), 0o644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	// Wait for triage to complete.
	if !pollForArtifactStatus(t, env, relPath, "draft", 5*time.Second) {
		t.Fatal("artifact not triaged to draft within 5s")
	}

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath(relPath)
	if err != nil {
		t.Fatalf("listing runs: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected exactly 1 agent run, got %d", len(runs))
	}
}

// TestTriageWatcher_ReRunAfterStatusReset verifies that resetting a triaged
// artifact's status back to raw triggers triage again. The ## Raw Idea block
// is preserved; ## Idea is replaced.
func TestTriageWatcher_ReRunAfterStatusReset(t *testing.T) {
	firstBody := defaultProposeJSON("rerun", "Rerun Idea", nil)
	secondBody := defaultProposeJSON("rerun", "Rerun Idea", nil)
	installLLMFake(t, []string{firstBody, secondBody})
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	// Step 1: initial triage.
	writeRawIdea(t, env.projectRoot, "rerun", "Rerun Idea",
		"This is the rerun idea with enough words for the initial triage pass.")
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/rerun.md", "draft", 5*time.Second) {
		t.Fatal("initial triage did not produce draft within 5s")
	}

	// Read the ## Raw Idea block from the triaged artifact.
	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/rerun.md")
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading triaged artifact: %v", err)
	}
	triaged := string(content)
	rawIdeaStart := strings.Index(triaged, "## Raw Idea")
	ideaStart := strings.Index(triaged, "## Idea")
	if rawIdeaStart < 0 || ideaStart < 0 {
		t.Fatalf("triaged artifact missing sections; content:\n%s", triaged)
	}
	originalRawBlock := triaged[rawIdeaStart:ideaStart]

	// Step 2: reset status to raw (write with existing sections still present).
	resetContent := strings.Replace(triaged, "status: draft", "status: raw", 1)
	if err := os.WriteFile(absPath, []byte(resetContent), 0o644); err != nil {
		t.Fatalf("resetting status: %v", err)
	}

	// Wait for re-triage.
	if !pollForArtifactStatus(t, env, "lifecycle/ideas/rerun.md", "draft", 5*time.Second) {
		t.Fatal("re-triage did not produce draft within 5s")
	}

	// Read post-re-triage content.
	reContent, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading re-triaged artifact: %v", err)
	}
	reTriaged := string(reContent)

	// ## Raw Idea block must be byte-identical.
	reRawStart := strings.Index(reTriaged, "## Raw Idea")
	reIdeaStart := strings.Index(reTriaged, "## Idea")
	if reRawStart < 0 || reIdeaStart < 0 {
		t.Fatalf("re-triaged artifact missing sections; content:\n%s", reTriaged)
	}
	newRawBlock := reTriaged[reRawStart:reIdeaStart]
	if originalRawBlock != newRawBlock {
		t.Errorf("## Raw Idea block changed on re-run:\nbefore: %q\nafter:  %q",
			originalRawBlock, newRawBlock)
	}
}
