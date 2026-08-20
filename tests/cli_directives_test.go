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

// legacyClaudeMdFixture is a genuine pre-refactor CLAUDE.md body — the last
// rendered form of internal/initcmd/templates/CLAUDE.md.tmpl before commit
// 0a6956c2 ("feat(init): emit AGENTS.md-primary directive set on project
// init") replaced it with the AGENTS.md-primary pointer scheme. `runInit`
// now always produces the already-migrated layout (CLAUDE.md as a bare
// `@AGENTS.md` pointer), so it can no longer stand in for a legacy fixture;
// this constant is written directly to disk instead so migrate-directives
// has real legacy content to migrate.
const legacyClaudeMdFixture = `# CLAUDE.md — legacy-project

Guidance for Claude Code (claude.ai/code) when working in this repository.

## Repository Layout

` + "```" + `
<project-root>/
├── lifecycle/              Artifact store (ideas → requirements → plans → releases)
│   ├── config.yaml         Per-project configuration (roles, agents, stages)
│   ├── architecture/       Chosen architecture/stack, ADRs (decisions/), standards
│   ├── ideas/
│   ├── requirements/
│   ├── backend-plans/
│   ├── frontend-plans/
│   ├── test-plans/
│   ├── tests/
│   ├── prototypes/
│   ├── releases/
│   ├── defects/
│   └── docs/
└── tests/                  Integration test code
` + "```" + `

## Lineage Filename Convention

Artifacts for a single idea share a **slug** and carry a **monotonic index** across stages:

` + "```" + `
lifecycle/ideas/login.md             (originating idea — no index suffix)
lifecycle/requirements/login-2.md
lifecycle/backend-plans/login-3-be.md
lifecycle/frontend-plans/login-4-fe.md
lifecycle/test-plans/login-5-test.md
` + "```" + `

- The first file in a lineage has **no index suffix**. Subsequent indices start at ` + "`-2`" + `.
- The index is monotonic **per lineage, across all stages** — never reused.
- Rejected-and-replanned artifacts get the **next** index; superseded files stay in place.
- Every non-originating artifact has ` + "`parent:`" + ` in its YAML frontmatter pointing to the previous file.

## Frontmatter Requirements

Required fields on every artifact: ` + "`title`, `type`, `status`, `lineage`" + `.

## Commit Conventions

- Commits should be small and focused. Don't amend published commits; create new ones.
- When a plan drove a change, reference the plan's milestone heading in the commit message.
- Do not skip pre-commit hooks or signing.

## Agent Roles

Roles split by lifecycle phase:

- **Think:** ` + "`analyst`" + ` — reads ideas → writes requirements; reads requirements → writes plans.
- **Make:** ` + "`backend-developer`, `frontend-developer`, `test-developer`" + ` — implement plans in code.
- **Verify:** ` + "`qa`" + ` — runs tests, raises defects in ` + "`lifecycle/defects/`" + `.
- **Cross-cutting:** ` + "`product-owner`, `reviewer`" + `.
`

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

	// init produces the already-migrated AGENTS.md-primary layout (including
	// a GEMINI.md pointer — init always establishes the full multi-agent set
	// via IncludeGemini: true, unlike migrate-directives); rewind this
	// project to a genuine pre-refactor legacy layout — no AGENTS.md or
	// GEMINI.md, CLAUDE.md holding the real (non-pointer) content — so the
	// migration path this test names is actually exercised.
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "GEMINI.md")); err != nil {
		t.Fatal(err)
	}
	legacyClaudeMd := []byte(legacyClaudeMdFixture)
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), legacyClaudeMd, 0o644); err != nil {
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

	// Rewind to a genuine legacy layout, as above: without a legacy
	// (non-pointer) CLAUDE.md, migrate-directives no-ops before it ever
	// looks at AGENTS.md, and the pending-diff path below is never reached.
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(legacyClaudeMdFixture), 0o644); err != nil {
		t.Fatal(err)
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
