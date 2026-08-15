// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyClaudeMd = "# CLAUDE.md — my-project\n\nSome hand-written guidance for Claude Code.\n"

func legacyProjectFixture(t *testing.T) string {
	t.Helper()
	root := writeConfigFixture(t) // has a qa agent on driver: gemini-cli
	if err := os.WriteFile(filepath.Join(root, claudeFile), []byte(legacyClaudeMd), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestNeedsMigration_LegacyLayout(t *testing.T) {
	root := legacyProjectFixture(t)
	needed, err := NeedsMigration(root)
	if err != nil {
		t.Fatalf("NeedsMigration: %v", err)
	}
	if !needed {
		t.Error("expected NeedsMigration=true for a legacy CLAUDE.md-only layout")
	}
}

func TestNeedsMigration_NoClaudeMd(t *testing.T) {
	root := writeConfigFixture(t)
	needed, err := NeedsMigration(root)
	if err != nil {
		t.Fatalf("NeedsMigration: %v", err)
	}
	if needed {
		t.Error("expected NeedsMigration=false with no CLAUDE.md at all")
	}
}

func TestMigrate_LegacyLayout_ProducesAgentsClaudeGemini(t *testing.T) {
	root := legacyProjectFixture(t)

	res, err := Migrate(root, MigrateOptions{})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	agentsBody, err := os.ReadFile(filepath.Join(root, agentsFile))
	if err != nil {
		t.Fatalf("AGENTS.md not written: %v", err)
	}
	if !strings.Contains(string(agentsBody), "Some hand-written guidance for Claude Code.") {
		t.Errorf("AGENTS.md missing legacy body:\n%s", agentsBody)
	}
	if !strings.HasPrefix(string(agentsBody), genStart) {
		t.Error("AGENTS.md legacy body was not wrapped in managed markers")
	}

	claudeBody, err := os.ReadFile(filepath.Join(root, claudeFile))
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	if string(claudeBody) != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q", claudeBody)
	}

	geminiBody, err := os.ReadFile(filepath.Join(root, geminiFile))
	if err != nil {
		t.Fatalf("GEMINI.md not written: %v", err)
	}
	if string(geminiBody) != "@AGENTS.md\n" {
		t.Errorf("GEMINI.md: got %q", geminiBody)
	}

	if len(res.Files) != 3 {
		t.Errorf("expected 3 FileWrite entries, got %d: %+v", len(res.Files), res.Files)
	}

	needed, err := NeedsMigration(root)
	if err != nil {
		t.Fatalf("NeedsMigration: %v", err)
	}
	if needed {
		t.Error("expected NeedsMigration=false after a successful migration")
	}
}

func TestMigrate_AlreadyMigrated_NoOp(t *testing.T) {
	root := legacyProjectFixture(t)
	if _, err := Migrate(root, MigrateOptions{}); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	res, err := Migrate(root, MigrateOptions{})
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(res.Files) != 0 {
		t.Errorf("expected empty Files on an already-migrated project, got %+v", res.Files)
	}
}

func TestMigrate_NoClaudeMd_NoOp(t *testing.T) {
	root := writeConfigFixture(t)
	res, err := Migrate(root, MigrateOptions{})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Files) != 0 {
		t.Errorf("expected empty Files with no CLAUDE.md, got %+v", res.Files)
	}
}

func TestMigrate_UserEditedAgentsMd_RequiresForce(t *testing.T) {
	root := legacyProjectFixture(t)
	userContent := "# Hand-written AGENTS.md that predates migration\n"
	if err := os.WriteFile(filepath.Join(root, agentsFile), []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(root, MigrateOptions{})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Files) != 1 || res.Files[0].Diff == "" {
		t.Fatalf("expected a single FileWrite with a Diff, got %+v", res.Files)
	}

	agentsAfter, _ := os.ReadFile(filepath.Join(root, agentsFile))
	if string(agentsAfter) != userContent {
		t.Error("AGENTS.md should not have been written without force")
	}
	claudeAfter, _ := os.ReadFile(filepath.Join(root, claudeFile))
	if string(claudeAfter) != legacyClaudeMd {
		t.Error("CLAUDE.md should not have been touched when AGENTS.md write was withheld")
	}

	res2, err := Migrate(root, MigrateOptions{Force: true})
	if err != nil {
		t.Fatalf("Migrate with force: %v", err)
	}
	for _, f := range res2.Files {
		if f.Path == agentsFile && f.Diff != "" {
			t.Errorf("expected no diff gate with force=true, got %+v", f)
		}
	}
	agentsAfterForce, _ := os.ReadFile(filepath.Join(root, agentsFile))
	if !strings.Contains(string(agentsAfterForce), "Some hand-written guidance for Claude Code.") {
		t.Errorf("expected AGENTS.md replaced with migrated content, got:\n%s", agentsAfterForce)
	}
}
