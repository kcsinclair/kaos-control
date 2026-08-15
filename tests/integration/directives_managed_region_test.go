// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 3 (FR-11, OQ-6): refresh replaces only the generated region and
// never clobbers user prose written outside it; a file whose managed-region
// markers are missing (deleted, or hand-authored) is diff-gated behind
// Force before it's replaced.
//
// Note on "edit content inside the markers to simulate a user-edited
// generated block; refresh without force → response carries a diff": OQ-6's
// resolution ("directives are generated once when the stack is picked and
// frozen until an explicit refresh... Pairs with managed-region markers so
// a refresh updates only the generated block") settles this in favour of
// *always* refreshing the managed region on a successful refresh — the
// region is machine territory by definition, so there is nothing to diff
// there; diff-gating (FR-11) applies only when the markers themselves are
// missing. internal/directives/write_test.go's
// TestWriteFile_SurgicalRefresh_NoForceNeeded locks in the same behaviour
// against the underlying writeFile primitive.
// TestDirectivesManagedRegion_EditsInsideMarkers_AlwaysRefreshedNoForceNeeded
// below verifies that resolution holds through the HTTP endpoint too,
// rather than silently assuming the test plan's literal wording.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectivesManagedRegion_ProseOutsideMarkersSurvivesRefresh(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)

	agentsPath := filepath.Join(env.projectRoot, "AGENTS.md")
	original := string(readFileMust(t, agentsPath))
	withProse := "Team note: talk to #platform before touching this file.\n\n" + original +
		"\nAnother note appended after the managed region.\n"
	if err := os.WriteFile(agentsPath, []byte(withProse), 0o644); err != nil {
		t.Fatal(err)
	}

	// Force a genuine change to the generated region itself by editing the
	// promoted stack's `run:` command on disk, so this refresh isn't a no-op.
	stackPath := filepath.Join(env.projectRoot, "lifecycle", "architecture", "go-vue.md")
	stackRaw := string(readFileMust(t, stackPath))
	updatedStackRaw := strings.Replace(stackRaw, "go run ./cmd/<app>", "go run ./cmd/myapp", 1)
	if updatedStackRaw == stackRaw {
		t.Fatal("fixture setup: expected to find the go-vue run command to replace")
	}
	if err := os.WriteFile(stackPath, []byte(updatedStackRaw), 0o644); err != nil {
		t.Fatal(err)
	}

	refreshDirectives(t, env, false)

	after := string(readFileMust(t, agentsPath))
	if !strings.Contains(after, "Team note: talk to #platform before touching this file.") {
		t.Error("prose above the managed region did not survive refresh")
	}
	if !strings.Contains(after, "Another note appended after the managed region.") {
		t.Error("prose below the managed region did not survive refresh")
	}
	if !strings.Contains(after, "go run ./cmd/myapp") {
		t.Errorf("expected the refreshed managed region to reflect the updated run command:\n%s", after)
	}
}

func TestDirectivesManagedRegion_MarkersDeleted_RequiresForce(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)

	agentsPath := filepath.Join(env.projectRoot, "AGENTS.md")
	tampered := "# AGENTS.md\n\nHand-rewritten, markers removed entirely.\n"
	if err := os.WriteFile(agentsPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}

	data := refreshDirectives(t, env, false)
	fw := fileWriteByPath(t, data, "AGENTS.md")
	diff, _ := fw["diff"].(string)
	if diff == "" {
		t.Fatal("expected a non-empty diff when markers are missing and force=false")
	}
	unchanged := string(readFileMust(t, agentsPath))
	if unchanged != tampered {
		t.Error("AGENTS.md should not have been written without force")
	}

	data2 := refreshDirectives(t, env, true)
	fw2 := fileWriteByPath(t, data2, "AGENTS.md")
	if d, _ := fw2["diff"].(string); d != "" {
		t.Errorf("expected no diff with force=true, got: %s", d)
	}
	afterForce := string(readFileMust(t, agentsPath))
	if strings.Contains(afterForce, "Hand-rewritten, markers removed entirely.") {
		t.Error("expected AGENTS.md to be replaced when force=true")
	}
	if !strings.HasPrefix(afterForce, "<!-- kaos-control:generated:start -->") {
		t.Error("expected managed-region markers restored after the forced overwrite")
	}
}

func TestDirectivesManagedRegion_EditsInsideMarkers_AlwaysRefreshedNoForceNeeded(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, []seedArtifact{promotedStackSeed(t, "go-vue.md")}, directivesCfgYAML)
	refreshDirectives(t, env, false)

	agentsPath := filepath.Join(env.projectRoot, "AGENTS.md")
	original := string(readFileMust(t, agentsPath))
	handEdited := strings.Replace(original, "## Commit Conventions", "## Commit Conventions (hand-edited by a user)", 1)
	if handEdited == original {
		t.Fatal("fixture setup: expected to find \"## Commit Conventions\" inside the generated region")
	}
	if err := os.WriteFile(agentsPath, []byte(handEdited), 0o644); err != nil {
		t.Fatal(err)
	}

	data := refreshDirectives(t, env, false)
	fw := fileWriteByPath(t, data, "AGENTS.md")
	if diff, _ := fw["diff"].(string); diff != "" {
		t.Errorf("expected no diff gate for an edit inside the managed markers (see file header), got: %s", diff)
	}

	after := string(readFileMust(t, agentsPath))
	if strings.Contains(after, "hand-edited by a user") {
		t.Error("expected the hand-edited generated region to be overwritten by refresh")
	}
}
