// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 4
// (FR-11, FR-12, FR-14, OQ-4): ADR numbering (monotonic, zero-padded, never
// reused), ADR-0001 authoring idempotency, and default status: draft for
// human/agent-created ADRs.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

// adrFilenameRe guards the created-ADR filename format: adr-NNNN-slug.md.
var adrFilenameRe = regexp.MustCompile(`^adr-\d{4}-.+\.md$`)

func createADR(t *testing.T, env *testEnv, slug, title, status, body string) map[string]any {
	t.Helper()
	req := map[string]string{"slug": slug, "title": title, "body": body}
	if status != "" {
		req["status"] = status
	}
	resp := env.doRequest("POST", "/api/p/testproject/architecture/adrs", req)
	requireStatus(t, resp, 201)
	return readJSON(t, resp)
}

// TestADR_NextNumber_EmptyDecisionsDir covers GET .../adrs/next on an empty
// decisions/ directory.
func TestADR_NextNumber_EmptyDecisionsDir(t *testing.T) {
	env := newTestEnv(t, nil)

	resp := env.doRequest("GET", "/api/p/testproject/architecture/adrs/next", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if n, ok := data["number"].(float64); !ok || n != 1 {
		t.Errorf("number = %v, want 1", data["number"])
	}
}

// TestADR_CreateAndNumbering covers: first POST creates adr-0001-<slug>.md
// with type: adr, status: draft by default; a second POST creates
// adr-0002-*.md; next then returns 3.
func TestADR_CreateAndNumbering(t *testing.T) {
	env := newTestEnv(t, nil)

	first := createADR(t, env, "adopt-postgres", "Adopt Postgres", "", "Because reasons.")
	if got, _ := first["path"].(string); got != "lifecycle/architecture/decisions/adr-0001-adopt-postgres.md" {
		t.Errorf("first ADR path = %q", got)
	}
	if n, _ := first["number"].(float64); n != 1 {
		t.Errorf("first ADR number = %v, want 1", first["number"])
	}

	content := string(readFileMust(t, filepath.Join(env.projectRoot, "lifecycle/architecture/decisions/adr-0001-adopt-postgres.md")))
	if !strings.Contains(content, "type: adr") {
		t.Errorf("expected type: adr in created ADR, got:\n%s", content)
	}
	if !strings.Contains(content, "status: draft") {
		t.Errorf("expected default status: draft, got:\n%s", content)
	}

	second := createADR(t, env, "adopt-vue", "Adopt Vue", "", "Because reasons.")
	if got, _ := second["path"].(string); got != "lifecycle/architecture/decisions/adr-0002-adopt-vue.md" {
		t.Errorf("second ADR path = %q", got)
	}
	if n, _ := second["number"].(float64); n != 2 {
		t.Errorf("second ADR number = %v, want 2", second["number"])
	}

	nextResp := env.doRequest("GET", "/api/p/testproject/architecture/adrs/next", nil)
	requireStatus(t, nextResp, 200)
	nextData := readJSON(t, nextResp)
	if n, _ := nextData["number"].(float64); n != 3 {
		t.Errorf("next number after two ADRs = %v, want 3", nextData["number"])
	}
}

// TestADR_SupersededStillCounts adds a status: superseded adr-0003 fixture
// directly on disk and asserts /adrs/next still returns 4 — superseded
// numbers are counted and never reused (FR-14).
func TestADR_SupersededStillCounts(t *testing.T) {
	env := newTestEnv(t, nil)

	createADR(t, env, "one", "One", "", "Body.")
	createADR(t, env, "two", "Two", "", "Body.")

	supersededPath := filepath.Join(env.projectRoot, "lifecycle/architecture/decisions/adr-0003-superseded.md")
	content := "---\ntitle: Superseded\ntype: adr\nstatus: superseded\n---\n\nBody.\n"
	if err := os.WriteFile(supersededPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("GET", "/api/p/testproject/architecture/adrs/next", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if n, _ := data["number"].(float64); n != 4 {
		t.Errorf("next number with superseded adr-0003 present = %v, want 4", data["number"])
	}
}

// TestADR_FilenameFormat guards that a created ADR's basename matches
// ^adr-\d{4}-.+\.md$.
func TestADR_FilenameFormat(t *testing.T) {
	env := newTestEnv(t, nil)

	created := createADR(t, env, "format-check", "Format Check", "", "Body.")
	path, _ := created["path"].(string)
	base := filepath.Base(path)
	if !adrFilenameRe.MatchString(base) {
		t.Errorf("created ADR filename %q does not match %s", base, adrFilenameRe.String())
	}
}

// TestADR0001_AuthoringIsIdempotent invokes the WriteADR0001 primitive twice
// with identical inputs directly (the wizard's ADR-0001 authoring path) and
// asserts exactly one adr-0001-*.md results, titled "Adopt <arch> with
// <stack>", with the Q&A trail and ranked rejected alternatives in the body,
// and no adr-0002 spawned (FR-12, NFR-3).
func TestADR0001_AuthoringIsIdempotent(t *testing.T) {
	env := newTestEnv(t, nil)

	qaTrail := "Q: Why Postgres? A: Because it fits our consistency needs."
	rejected := []string{"MongoDB", "MySQL"}

	path1, err := architecture.WriteADR0001(env.projectRoot, "Postgres Modular Monolith", "Go + Vue", qaTrail, rejected)
	if err != nil {
		t.Fatalf("first WriteADR0001: %v", err)
	}
	path2, err := architecture.WriteADR0001(env.projectRoot, "Postgres Modular Monolith", "Go + Vue", qaTrail, rejected)
	if err != nil {
		t.Fatalf("second WriteADR0001: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected the same path both times, got %q then %q", path1, path2)
	}

	entries, err := os.ReadDir(filepath.Join(env.projectRoot, "lifecycle/architecture/decisions"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 {
		t.Fatalf("expected exactly one adr-0001-*.md, got %v", names)
	}
	if !strings.HasPrefix(names[0], "adr-0001-") {
		t.Errorf("expected adr-0001-*, got %q", names[0])
	}
	for _, n := range names {
		if strings.HasPrefix(n, "adr-0002-") {
			t.Errorf("unexpected adr-0002 spawned: %v", names)
		}
	}

	content := string(readFileMust(t, filepath.Join(env.projectRoot, path1)))
	if !strings.Contains(content, "title: Adopt Postgres Modular Monolith with Go + Vue") {
		t.Errorf("expected title 'Adopt Postgres Modular Monolith with Go + Vue', got:\n%s", content)
	}
	if !strings.Contains(content, qaTrail) {
		t.Errorf("expected Q&A trail in body, got:\n%s", content)
	}
	if !strings.Contains(content, "## Rejected alternatives") ||
		!strings.Contains(content, "MongoDB") || !strings.Contains(content, "MySQL") {
		t.Errorf("expected ranked rejected-alternatives section, got:\n%s", content)
	}
}
