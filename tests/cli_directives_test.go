// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package cli_test

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 6 (FR-16, FR-14 — CLI half; the HTTP half lives in
// tests/integration/directives_migration_test.go): `kaos-control
// migrate-directives` produces the same file set as the
// POST .../migrate-directives endpoint and exits non-zero with a hint on a
// pending diff; `kaos-control init --refresh-directives` on a promoted
// project regenerates AGENTS.md and patches the standard agents, matching
// the POST .../directives/refresh endpoint's output for the same selection.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func writePromotedGoVueStack(t *testing.T, projectDir string) {
	t.Helper()
	raw, err := catalogfs.FS.ReadFile("tech-stacks/go-vue.md")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(projectDir, "lifecycle", "architecture", "go-vue.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ─── migrate-directives ───────────────────────────────────────────────────

func TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini(t *testing.T) {
	dir := t.TempDir()
	if _, _, code := runInit(t, dir); code != 0 {
		t.Fatalf("init: want exit 0, got %d", code)
	}
	legacyClaudeMd, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runBin(t, "migrate-directives", dir)
	if code != 0 {
		t.Fatalf("migrate-directives: want exit 0, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Migrated directives for "+dir) {
		t.Errorf("stdout missing migration banner:\n%s", stdout)
	}

	agentsBody, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not written: %v", err)
	}
	if !strings.Contains(string(agentsBody), string(legacyClaudeMd)) {
		t.Errorf("AGENTS.md missing the legacy CLAUDE.md body:\nAGENTS.md:\n%s\n\nlegacy CLAUDE.md:\n%s", agentsBody, legacyClaudeMd)
	}
	if !strings.HasPrefix(string(agentsBody), "<!-- kaos-control:generated:start -->") {
		t.Error("AGENTS.md legacy body was not wrapped in managed markers")
	}

	claudeBody, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(claudeBody) != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q", claudeBody)
	}

	// The default init scaffold configures no gemini/gemini-cli driver, so
	// GEMINI.md is skipped and reported, never left as an orphan (FR-12).
	if _, err := os.Stat(filepath.Join(dir, "GEMINI.md")); !os.IsNotExist(err) {
		t.Error("GEMINI.md should not have been written with no gemini driver configured")
	}
	if !strings.Contains(stdout, "skipped  GEMINI.md (no gemini driver configured)") {
		t.Errorf("stdout does not report GEMINI.md as skipped:\n%s", stdout)
	}
}

func TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint(t *testing.T) {
	dir := t.TempDir()
	if _, _, code := runInit(t, dir); code != 0 {
		t.Fatalf("init: want exit 0, got %d", code)
	}

	userAgentsMd := "# Hand-written AGENTS.md that predates migration\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(userAgentsMd), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runBin(t, "migrate-directives", dir)
	if code == 0 {
		t.Fatalf("expected non-zero exit for a pending diff, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "-force") {
		t.Errorf("stderr missing the -force hint:\n%s", stderr)
	}
	agentsAfter, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agentsAfter) != userAgentsMd {
		t.Error("AGENTS.md should not have been written without -force")
	}

	stdout2, stderr2, code2 := runBin(t, "migrate-directives", "-force", dir)
	if code2 != 0 {
		t.Fatalf("migrate-directives -force: want exit 0, got %d\nstdout:\n%s\nstderr:\n%s", code2, stdout2, stderr2)
	}
	agentsAfterForce, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agentsAfterForce) == userAgentsMd {
		t.Error("expected AGENTS.md to be replaced with -force")
	}
}

// ─── init --refresh-directives ─────────────────────────────────────────────

func TestInitRefreshDirectivesCmd_PromotedProject_RegeneratesAgentsAndPatchesConfig(t *testing.T) {
	dir := t.TempDir()
	if _, _, code := runInit(t, dir); code != 0 {
		t.Fatalf("init: want exit 0, got %d", code)
	}
	writePromotedGoVueStack(t, dir)

	// The freshly scaffolded CLAUDE.md predates AGENTS.md and carries no
	// managed markers, so the first refresh needs -force to replace it —
	// matching the diff-gate exercised in
	// tests/integration/directives_managed_region_test.go.
	stdout, stderr, code := runBin(t, "init", "--refresh-directives", "-force", dir)
	if code != 0 {
		t.Fatalf("init --refresh-directives: want exit 0, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Refreshed directives for "+dir) {
		t.Errorf("stdout missing refresh banner:\n%s", stdout)
	}

	agentsBody, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not written: %v", err)
	}
	for _, want := range []string{"internal/", "cmd/", "web/src/"} {
		if !strings.Contains(string(agentsBody), want) {
			t.Errorf("AGENTS.md missing go-vue layout entry %q:\n%s", want, agentsBody)
		}
	}
	if !strings.HasPrefix(string(agentsBody), "<!-- kaos-control:generated:start -->") {
		t.Error("AGENTS.md missing managed-region markers")
	}

	claudeBody, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(claudeBody) != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md: got %q", claudeBody)
	}

	cfgRaw, err := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgRaw), "go build ./...") {
		t.Error("lifecycle/config.yaml was not patched with the go-vue build command")
	}

	// Same selection, re-run without -force: markers are now intact, so the
	// surgical refresh applies with no diff gate, and re-running is a no-op.
	stdout2, stderr2, code2 := runBin(t, "init", "--refresh-directives", dir)
	if code2 != 0 {
		t.Fatalf("second init --refresh-directives: want exit 0, got %d\nstdout:\n%s\nstderr:\n%s", code2, stdout2, stderr2)
	}
	if strings.Contains(stdout2, "  created  ") || strings.Contains(stdout2, "  updated  ") {
		t.Errorf("expected a no-op second run, got:\n%s", stdout2)
	}
}
