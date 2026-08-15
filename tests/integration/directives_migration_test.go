// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 6 (FR-16, FR-14 — HTTP half; the CLI half lives in
// tests/cli_directives_test.go): first-run legacy detection surfaced via
// the project summary, POST .../migrate-directives, idempotency, and the
// diff-before-overwrite gate on a user-edited AGENTS.md.
//
// FR-15 ("files written/updated by generation are... picked up by the
// existing indexing paths... with no special-casing") is deliberately NOT
// exercised as "retrievable via the artifacts index" here. As shipped,
// AGENTS.md/CLAUDE.md/GEMINI.md are written at the project root, and:
//   - internal/index/index.go's Scan restricts artifacts to
//     lifecycle/**/*.md ("Only .md files inside lifecycle/ are artifacts").
//   - internal/watcher/watcher.go's Watcher only recurses lifecycle/ and
//     docs/ (projectRoot/docs), never the project root itself.
//   - internal/directives/generate.go documents that Generate/Migrate hold
//     no index handle of their own ("a caller that does have one should
//     re-index each Created/Changed file").
//   - internal/http/projects.go's handleRefreshDirectives and
//     handleMigrateDirectives call p.ReloadConfig() (for the
//     lifecycle/config.yaml patch) but no re-index entry point for these
//     files.
// So there is no code path that makes AGENTS.md/CLAUDE.md/GEMINI.md
// retrievable via GET .../artifacts.
// TestDirectivesMigrate_WrittenFilesNotInArtifactsIndex documents that gap
// explicitly (asserting the current, verified behaviour) rather than
// silently assuming FR-15 holds for these files or silently dropping the
// check. See the companion lifecycle/tests/ artifact's Open Questions for
// the product-owner decision this needs (extend the index/watcher scope to
// cover project-root directive files, or revise FR-15 to exclude them, as
// FR-1/FR-3 were explicitly revised for the AGENTS.md-primary decision).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyClaudeMdBody = "# CLAUDE.md — legacy project\n\nSome hand-written guidance for Claude Code.\n"

func legacyDirectivesSeeds() []seedArtifact {
	return []seedArtifact{{relPath: "CLAUDE.md", content: legacyClaudeMdBody}}
}

func projectSummary(t *testing.T, env *testEnv) map[string]any {
	t.Helper()
	resp := env.doRequest("GET", "/api/projects/testproject", nil)
	requireStatus(t, resp, 200)
	return readJSON(t, resp)
}

func migrateDirectives(t *testing.T, env *testEnv, force bool) map[string]any {
	t.Helper()
	resp := env.doRequest("POST", "/api/projects/testproject/migrate-directives", map[string]any{"force": force})
	requireStatus(t, resp, 200)
	return readJSON(t, resp)
}

func TestDirectivesMigrate_LegacyLayout_SummaryReportsAvailable(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), directivesCfgYAML)
	summary := projectSummary(t, env)
	if avail, _ := summary["directivesMigrationAvailable"].(bool); !avail {
		t.Errorf("expected directivesMigrationAvailable=true for a legacy CLAUDE.md-only layout, got %v", summary)
	}
}

func TestDirectivesMigrate_NoLegacyClaudeMd_SummaryReportsUnavailable(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, directivesCfgYAML)
	summary := projectSummary(t, env)
	if avail, _ := summary["directivesMigrationAvailable"].(bool); avail {
		t.Errorf("expected directivesMigrationAvailable=false with no CLAUDE.md at all, got %v", summary)
	}
}

func TestDirectivesMigrate_LegacyLayout_ProducesAgentsClaudeGemini(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), directivesCfgYAML)

	data := migrateDirectives(t, env, false)
	files, _ := data["files"].([]any)
	if len(files) != 3 {
		t.Fatalf("expected 3 FileWrite entries (AGENTS.md, CLAUDE.md, GEMINI.md), got %d: %v", len(files), files)
	}

	agentsBody := string(readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md")))
	if !strings.Contains(agentsBody, "Some hand-written guidance for Claude Code.") {
		t.Errorf("AGENTS.md missing legacy body:\n%s", agentsBody)
	}
	if !strings.HasPrefix(agentsBody, "<!-- kaos-control:generated:start -->") {
		t.Error("AGENTS.md legacy body was not wrapped in managed markers")
	}

	claudeBody := string(readFileMust(t, filepath.Join(env.projectRoot, "CLAUDE.md")))
	if claudeBody != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q", claudeBody)
	}
	geminiBody := string(readFileMust(t, filepath.Join(env.projectRoot, "GEMINI.md")))
	if geminiBody != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md: got %q", geminiBody)
	}

	summary := projectSummary(t, env)
	if avail, _ := summary["directivesMigrationAvailable"].(bool); avail {
		t.Errorf("expected directivesMigrationAvailable=false after a successful migration, got %v", summary)
	}
}

func TestDirectivesMigrate_RepostAfterMigration_IsIdempotentNoOp(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), directivesCfgYAML)
	migrateDirectives(t, env, false)

	data2 := migrateDirectives(t, env, false)
	if files, _ := data2["files"].([]any); len(files) != 0 {
		t.Errorf("expected an empty file report on re-post to an already-migrated project, got %v", files)
	}
}

func TestDirectivesMigrate_UserEditedAgentsMd_RequiresForce(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), directivesCfgYAML)
	userContent := "# Hand-written AGENTS.md that predates migration\n"
	if err := os.WriteFile(filepath.Join(env.projectRoot, "AGENTS.md"), []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	data := migrateDirectives(t, env, false)
	files, _ := data["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("expected a single FileWrite (AGENTS.md, withheld), got %d: %v", len(files), files)
	}
	fw, _ := files[0].(map[string]any)
	if diff, _ := fw["diff"].(string); diff == "" {
		t.Fatal("expected a non-empty diff")
	}

	agentsAfter := string(readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md")))
	if agentsAfter != userContent {
		t.Error("AGENTS.md should not have been written without force")
	}
	claudeAfter := string(readFileMust(t, filepath.Join(env.projectRoot, "CLAUDE.md")))
	if claudeAfter != legacyClaudeMdBody {
		t.Error("CLAUDE.md should not have been touched when the AGENTS.md write was withheld")
	}

	data2 := migrateDirectives(t, env, true)
	files2, _ := data2["files"].([]any)
	for _, f := range files2 {
		fw2, _ := f.(map[string]any)
		if fw2["path"] == "AGENTS.md" {
			if diff, _ := fw2["diff"].(string); diff != "" {
				t.Errorf("expected no diff gate with force=true, got %+v", fw2)
			}
		}
	}
	agentsAfterForce := string(readFileMust(t, filepath.Join(env.projectRoot, "AGENTS.md")))
	if !strings.Contains(agentsAfterForce, "Some hand-written guidance for Claude Code.") {
		t.Errorf("expected AGENTS.md replaced with the migrated legacy content, got:\n%s", agentsAfterForce)
	}
}

func TestDirectivesMigrate_ReadOnlyUser_Returns403(t *testing.T) {
	// architectureRoleGateCfgYAML (from architecture_promote_test.go) demotes
	// qa@test.local to "reviewer" only, outside RolesAdminOnly.
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), architectureRoleGateCfgYAML)
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest("POST", "/api/projects/testproject/migrate-directives", map[string]any{"force": false})
	requireStatus(t, resp, 403)
}

// TestDirectivesMigrate_WrittenFilesNotInArtifactsIndex documents the FR-15
// gap explained in this file's header comment.
func TestDirectivesMigrate_WrittenFilesNotInArtifactsIndex(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, legacyDirectivesSeeds(), directivesCfgYAML)
	migrateDirectives(t, env, false)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	for _, path := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		if findArtifactRow(t, data, path) != nil {
			t.Errorf("%s unexpectedly appeared in the artifacts index — if this now passes, FR-15 has likely been implemented for project-root directive files; update this test and the companion lifecycle/tests/ artifact's open question accordingly", path)
		}
	}
}
