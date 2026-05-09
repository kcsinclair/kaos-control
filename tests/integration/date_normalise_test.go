// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/index"
)

// ---------------------------------------------------------------------------
// Milestone 4 — NormaliseDates: plain-date created backfill on startup
// ---------------------------------------------------------------------------

// TestNormaliseDates_RewritesPlainDate verifies that when the project index is
// opened with an artifact that has `created: "2026-04-27"` (plain-date format),
// the startup NormaliseDates routine rewrites the on-disk file to a valid
// RFC3339 value (midnight local time on that date).
func TestNormaliseDates_RewritesPlainDate(t *testing.T) {
	// Seed an artifact with a plain-date created field.
	plainDateContent := "---\ntitle: Plain Date Artifact\ntype: idea\nstatus: draft\nlineage: plain-date-artifact\ncreated: \"2026-04-27\"\n---\n\nBody.\n"
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/plain-date-artifact.md", content: plainDateContent},
	}
	env := newTestEnv(t, seeds)
	_ = env // env startup triggers Open → Scan → NormaliseDates

	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/plain-date-artifact.md")
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading rewritten file: %v", err)
	}
	content := string(raw)

	// The plain-date value must be gone.
	if strings.Contains(content, `"2026-04-27"`) || strings.Contains(content, `2026-04-27`) {
		// Check if the plain-date was not replaced (it's OK if it's part of an RFC3339 string).
		// The RFC3339 form includes the date followed by T..., so look for the bare form only.
		if strings.Contains(content, "created: \"2026-04-27\"") || strings.Contains(content, "created: 2026-04-27") {
			t.Errorf("plain-date created field was not rewritten; file content:\n%s", content)
		}
	}

	// Extract the created: line and parse it as RFC3339.
	createdVal := extractCreatedFromFM(content)
	if createdVal == "" {
		t.Fatalf("could not find created: field in rewritten file:\n%s", content)
	}
	parsed, err := time.Parse(time.RFC3339, createdVal)
	if err != nil {
		t.Errorf("rewritten created field %q is not valid RFC3339: %v", createdVal, err)
	}

	// Must be midnight on 2026-04-27 in local timezone.
	wantDate := time.Date(2026, 4, 27, 0, 0, 0, 0, time.Local)
	if parsed.Unix() != wantDate.Unix() {
		t.Errorf("rewritten created = %v (unix %d), want %v (unix %d)",
			parsed, parsed.Unix(), wantDate, wantDate.Unix())
	}
}

// TestNormaliseDates_LeavesRFC3339Untouched verifies that an artifact with an
// already-valid RFC3339 `created` field is not modified on disk during startup.
func TestNormaliseDates_LeavesRFC3339Untouched(t *testing.T) {
	const rfc3339Created = "2025-12-01T08:30:00Z"
	content := "---\ntitle: RFC3339 Artifact\ntype: idea\nstatus: draft\nlineage: rfc3339-artifact\ncreated: \"" + rfc3339Created + "\"\n---\n\nBody.\n"
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/rfc3339-artifact.md", content: content},
	}
	env := newTestEnv(t, seeds)

	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/rfc3339-artifact.md")
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	// The created value must still be exactly the original RFC3339 string.
	createdVal := extractCreatedFromFM(string(raw))
	if createdVal != rfc3339Created {
		t.Errorf("RFC3339 created was modified: want %q, got %q", rfc3339Created, createdVal)
	}
}

// TestNormaliseDates_PreservesOtherFields verifies that NormaliseDates does not
// alter any frontmatter fields other than `created`, and preserves the body.
func TestNormaliseDates_PreservesOtherFields(t *testing.T) {
	plainDateContent := "---\ntitle: Preserve Fields Test\ntype: ticket\nstatus: draft\nlineage: preserve-fields\ncreated: \"2026-01-15\"\n---\n\nThis is the body.\n"
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/preserve-fields.md", content: plainDateContent},
	}
	env := newTestEnv(t, seeds)

	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/preserve-fields.md")
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading rewritten file: %v", err)
	}
	content := string(raw)

	if !strings.Contains(content, "title: Preserve Fields Test") {
		t.Error("title field was modified or removed")
	}
	if !strings.Contains(content, "type: ticket") {
		t.Error("type field was modified or removed")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("status field was modified or removed")
	}
	if !strings.Contains(content, "lineage: preserve-fields") {
		t.Error("lineage field was modified or removed")
	}
	if !strings.Contains(content, "This is the body.") {
		t.Error("body was modified or removed")
	}

	// Created should now be RFC3339.
	createdVal := extractCreatedFromFM(content)
	if _, err := time.Parse(time.RFC3339, createdVal); err != nil {
		t.Errorf("created field %q is not valid RFC3339 after normalisation: %v", createdVal, err)
	}
}

// TestNormaliseDates_IndexReflectsNormalisedValue verifies that after
// NormaliseDates runs, the index stores the correct createdUnix value
// derived from the rewritten RFC3339 timestamp.
func TestNormaliseDates_IndexReflectsNormalisedValue(t *testing.T) {
	plainDateContent := "---\ntitle: Index Value Test\ntype: idea\nstatus: draft\nlineage: index-value-test\ncreated: \"2026-03-10\"\n---\n\nBody.\n"
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/index-value-test.md", content: plainDateContent},
	}
	env := newTestEnv(t, seeds)

	// The index was rebuilt on Open; re-index the rewritten file so the index
	// reflects the RFC3339 value (NormaliseDates rewrites disk after Scan).
	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/index-value-test.md")
	if err := env.proj.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile after normalisation: %v", err)
	}

	row, err := env.proj.Idx.Get("lifecycle/ideas/index-value-test.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("artifact not found in index")
	}
	if row.Created.IsZero() {
		t.Fatal("index Created is zero after normalisation")
	}

	wantDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local)
	if row.Created.Unix() != wantDate.Unix() {
		t.Errorf("index Created = %v (unix %d), want %v (unix %d)",
			row.Created, row.Created.Unix(), wantDate, wantDate.Unix())
	}
}

// ---------------------------------------------------------------------------
// Milestone 1 — Index Upsert: date parsing edge cases
// (These exercise the internal index package directly via the project's index.)
// ---------------------------------------------------------------------------

// TestIndexUpsert_PlainDateCreated verifies that upserting an artifact whose
// `created` frontmatter field is a plain date (YYYY-MM-DD) stores the correct
// createdUnix value — midnight local time on that date.
func TestIndexUpsert_PlainDateCreated(t *testing.T) {
	env := newTestEnv(t, nil)

	a := &artifact.Artifact{
		Path:  "lifecycle/ideas/plain-date-upsert.md",
		Slug:  "plain-date-upsert",
		Stage: "ideas",
		Index: 0,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Plain Date Upsert",
			Type:    "idea",
			Status:  "draft",
			Lineage: "plain-date-upsert",
			Created: "2026-04-27",
		},
	}
	if err := env.proj.Idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	row, err := env.proj.Idx.Get("lifecycle/ideas/plain-date-upsert.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}

	wantMidnight := time.Date(2026, 4, 27, 0, 0, 0, 0, time.Local)
	if row.Created.IsZero() {
		t.Fatal("expected non-zero Created for plain-date input")
	}
	if row.Created.Unix() != wantMidnight.Unix() {
		t.Errorf("Created mismatch: want %v (unix %d), got %v (unix %d)",
			wantMidnight, wantMidnight.Unix(), row.Created, row.Created.Unix())
	}
}

// TestIndexUpsert_GarbageCreated verifies that upserting an artifact with an
// unrecognised `created` value falls back to the artifact's CreatedAt field
// and does not error.
func TestIndexUpsert_GarbageCreated(t *testing.T) {
	env := newTestEnv(t, nil)

	fallbackTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	a := &artifact.Artifact{
		Path:      "lifecycle/ideas/garbage-created.md",
		Slug:      "garbage-created",
		Stage:     "ideas",
		Index:     0,
		Mtime:     time.Now(),
		CreatedAt: fallbackTime,
		FM: artifact.Frontmatter{
			Title:   "Garbage Created",
			Type:    "idea",
			Status:  "draft",
			Lineage: "garbage-created",
			Created: "not-a-date-at-all",
		},
	}
	if err := env.proj.Idx.Upsert(a); err != nil {
		t.Fatalf("Upsert with garbage created: %v", err)
	}

	row, err := env.proj.Idx.Get("lifecycle/ideas/garbage-created.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}

	// The garbage value should not produce a valid createdUnix — it should
	// fall back to CreatedAt. The index stores createdUnix=0 for unrecognised
	// values when there's no valid parse, so row.Created may be zero OR equal
	// to fallbackTime depending on Upsert implementation. Verify no panic and
	// that the stored value equals the fallback (or zero if no fallback path taken).
	// Per the implementation: garbage → no parse → createdUnix=0 → fallback to CreatedAt.
	if row.Created.IsZero() {
		// Zero is acceptable only if Upsert couldn't parse and ignored CreatedAt.
		// But the implementation does: if createdUnix == 0 && !a.CreatedAt.IsZero() → use CreatedAt.
		// So non-zero CreatedAt should propagate.
		t.Errorf("expected Created to fall back to CreatedAt %v, got zero", fallbackTime)
	} else if row.Created.Unix() != fallbackTime.Unix() {
		t.Errorf("Created = %v (unix %d), want fallback %v (unix %d)",
			row.Created, row.Created.Unix(), fallbackTime, fallbackTime.Unix())
	}
}

// TestIndexUpsert_EmptyCreatedUsesCreatedAt verifies that an artifact with an
// empty `created` frontmatter field uses the CreatedAt backfill value when present.
func TestIndexUpsert_EmptyCreatedUsesCreatedAt(t *testing.T) {
	env := newTestEnv(t, nil)

	backfill := time.Date(2024, 11, 20, 9, 30, 0, 0, time.UTC)
	a := &artifact.Artifact{
		Path:      "lifecycle/ideas/empty-created.md",
		Slug:      "empty-created",
		Stage:     "ideas",
		Index:     0,
		Mtime:     time.Now(),
		CreatedAt: backfill,
		FM: artifact.Frontmatter{
			Title:   "Empty Created",
			Type:    "idea",
			Status:  "draft",
			Lineage: "empty-created",
			Created: "",
		},
	}
	if err := env.proj.Idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	row, err := env.proj.Idx.Get("lifecycle/ideas/empty-created.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}
	if row.Created.IsZero() {
		t.Errorf("expected Created to be backfilled from CreatedAt %v, got zero", backfill)
	} else if row.Created.Unix() != backfill.Unix() {
		t.Errorf("Created = %v, want backfill %v", row.Created, backfill)
	}
}

// ---------------------------------------------------------------------------
// Milestone 3 (supplemental) — Upsert with RFC3339 input (direct path)
// ---------------------------------------------------------------------------

// TestIndexUpsert_RFC3339Created verifies that an artifact with an RFC3339
// `created` field stores exactly the right Unix timestamp in the index.
func TestIndexUpsert_RFC3339Created(t *testing.T) {
	env := newTestEnv(t, nil)

	const rfc3339 = "2026-05-01T14:00:00+10:00"
	want, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		t.Fatalf("bad test input: %v", err)
	}

	a := &artifact.Artifact{
		Path:  "lifecycle/ideas/rfc3339-upsert.md",
		Slug:  "rfc3339-upsert",
		Stage: "ideas",
		Index: 0,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "RFC3339 Upsert",
			Type:    "idea",
			Status:  "draft",
			Lineage: "rfc3339-upsert",
			Created: rfc3339,
		},
	}
	if err := env.proj.Idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	row, err := env.proj.Idx.Get("lifecycle/ideas/rfc3339-upsert.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("Get returned nil")
	}
	if row.Created.Unix() != want.Unix() {
		t.Errorf("Created = %v (unix %d), want %v (unix %d)",
			row.Created, row.Created.Unix(), want, want.Unix())
	}
}

// ---------------------------------------------------------------------------
// Helper — open a standalone index for the supplemental Milestone 1 tests
// ---------------------------------------------------------------------------

// openStandaloneIndex opens a fresh in-memory-ish index for unit-level testing
// without needing a full HTTP server.
func openStandaloneIndex(t *testing.T) *index.Index {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	if err := os.MkdirAll(dir+"/lifecycle", 0o755); err != nil {
		t.Fatal(err)
	}
	stages := []config.Stage{{Name: "ideas", Dir: "ideas"}}
	idx, err := index.Open(dbPath, dir, stages)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

// ---------------------------------------------------------------------------
// Helper — extract the created field value from a frontmatter block
// ---------------------------------------------------------------------------

// extractCreatedFromFM parses the created: field from YAML frontmatter content.
// Returns the unquoted value or empty string if not found.
func extractCreatedFromFM(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "created:") {
			continue
		}
		val := strings.TrimPrefix(trimmed, "created:")
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		return val
	}
	return ""
}
