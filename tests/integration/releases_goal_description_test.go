// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/release"
)

// ── Milestone 3: Release goal & description — REST API + round-trip-from-disk ──
//
// Covers DR-3, DR-5, DR-9 of requirements/release-goal-and-description-2.md
// per test-plans/release-goal-and-description-5-test.md Milestone 3.

// TestReleaseGoalDescription_CreateEchoesAndPersistsToDisk verifies that
// POST /releases with goal+description returns 201 with both fields echoed
// in the response body, and that the on-disk release file contains them
// (file-first write path, DR-3).
func TestReleaseGoalDescription_CreateEchoesAndPersistsToDisk(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	const goal = "Ship the roadmap goal subtitle"
	const description = "Adds a **goal** subtitle and a free-form description to every release."

	data := createRelease(t, env, map[string]any{
		"name":        "v-goal-desc-create",
		"status":      "planned",
		"goal":        goal,
		"description": description,
	})

	rel, _ := data["release"].(map[string]any)
	if got, _ := rel["goal"].(string); got != goal {
		t.Errorf("response goal: want %q, got %q", goal, got)
	}
	if got, _ := rel["description"].(string); got != description {
		t.Errorf("response description: want %q, got %q", description, got)
	}

	slug, _ := rel["slug"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md"))
	if err != nil {
		t.Fatalf("reading release file: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "goal: "+goal) {
		t.Errorf("release file missing goal line; content:\n%s", content)
	}
	if !strings.Contains(content, "description: "+description) {
		t.Errorf("release file missing description line; content:\n%s", content)
	}
}

// TestReleaseGoalDescription_GetAndListReflectValues verifies that both
// GET /releases (list) and GET /releases/{slug} return goal/description —
// populated when set, and "" (not absent) when never set.
func TestReleaseGoalDescription_GetAndListReflectValues(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	withData := createRelease(t, env, map[string]any{
		"name":        "v-goal-desc-with",
		"status":      "planned",
		"goal":        "With goal",
		"description": "With description",
	})
	withRel, _ := withData["release"].(map[string]any)
	withSlug, _ := withRel["slug"].(string)

	bareData := createRelease(t, env, map[string]any{
		"name":   "v-goal-desc-bare",
		"status": "planned",
	})
	bareRel, _ := bareData["release"].(map[string]any)
	bareSlug, _ := bareRel["slug"].(string)

	// GET /releases (list).
	listResp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, listResp, http.StatusOK)
	listBody := readJSON(t, listResp)
	releases, _ := listBody["releases"].([]any)

	found := map[string]map[string]any{}
	for _, raw := range releases {
		r, _ := raw.(map[string]any)
		name, _ := r["name"].(string)
		found[name] = r
	}

	if r, ok := found["v-goal-desc-with"]; !ok {
		t.Fatal("v-goal-desc-with missing from list")
	} else {
		if got, _ := r["goal"].(string); got != "With goal" {
			t.Errorf("list goal: want %q, got %q", "With goal", got)
		}
		if got, _ := r["description"].(string); got != "With description" {
			t.Errorf("list description: want %q, got %q", "With description", got)
		}
	}
	if r, ok := found["v-goal-desc-bare"]; !ok {
		t.Fatal("v-goal-desc-bare missing from list")
	} else {
		if got, ok := r["goal"]; !ok || got != "" {
			t.Errorf("list goal for unset release: want \"\", got %v (present=%v)", got, ok)
		}
		if got, ok := r["description"]; !ok || got != "" {
			t.Errorf("list description for unset release: want \"\", got %v (present=%v)", got, ok)
		}
	}

	// GET /releases/{slug} — direct slug lookup, both cases.
	withGet := env.doRequest("GET", "/api/p/testproject/releases/"+withSlug, nil)
	requireStatus(t, withGet, http.StatusOK)
	withBody := readJSON(t, withGet)
	withGot, _ := withBody["release"].(map[string]any)
	if got, _ := withGot["goal"].(string); got != "With goal" {
		t.Errorf("GET by slug goal: want %q, got %q", "With goal", got)
	}
	if got, _ := withGot["description"].(string); got != "With description" {
		t.Errorf("GET by slug description: want %q, got %q", "With description", got)
	}

	bareGet := env.doRequest("GET", "/api/p/testproject/releases/"+bareSlug, nil)
	requireStatus(t, bareGet, http.StatusOK)
	bareBody := readJSON(t, bareGet)
	bareGot, _ := bareBody["release"].(map[string]any)
	if got, _ := bareGot["goal"].(string); got != "" {
		t.Errorf("GET by slug goal for unset release: want \"\", got %q", got)
	}
	if got, _ := bareGot["description"].(string); got != "" {
		t.Errorf("GET by slug description for unset release: want \"\", got %q", got)
	}
}

// TestReleaseGoalDescription_UpdateOmittingFieldsPreservesValues verifies
// that PUT /releases/{id} with goal/description omitted from the JSON body
// leaves the stored values unchanged (merge-against-current semantics).
func TestReleaseGoalDescription_UpdateOmittingFieldsPreservesValues(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":        "v-goal-desc-omit",
		"status":      "planned",
		"goal":        "Original goal",
		"description": "Original description",
	})
	id := releaseID(t, data)

	// PUT with neither "goal" nor "description" present in the body.
	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-goal-desc-omit",
		"status": "active",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	if got, _ := rel["goal"].(string); got != "Original goal" {
		t.Errorf("goal after omit-update: want %q, got %q", "Original goal", got)
	}
	if got, _ := rel["description"].(string); got != "Original description" {
		t.Errorf("description after omit-update: want %q, got %q", "Original description", got)
	}

	// Disk file must still carry both original values.
	slug, _ := rel["slug"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md"))
	if err != nil {
		t.Fatalf("reading release file: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "goal: Original goal") {
		t.Errorf("release file lost goal on omit-update; content:\n%s", content)
	}
	if !strings.Contains(content, "description: Original description") {
		t.Errorf("release file lost description on omit-update; content:\n%s", content)
	}
}

// TestReleaseGoalDescription_UpdateEmptyStringClearsValues verifies that
// sending an explicit empty string for goal/description clears the stored
// value (distinct from omitting the field, which preserves it).
func TestReleaseGoalDescription_UpdateEmptyStringClearsValues(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":        "v-goal-desc-clear",
		"status":      "planned",
		"goal":        "Goal to clear",
		"description": "Description to clear",
	})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":        "v-goal-desc-clear",
		"status":      "planned",
		"goal":        "",
		"description": "",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	if got, _ := rel["goal"].(string); got != "" {
		t.Errorf("goal after clear: want \"\", got %q", got)
	}
	if got, _ := rel["description"].(string); got != "" {
		t.Errorf("description after clear: want \"\", got %q", got)
	}

	slug, _ := rel["slug"].(string)
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "releases", slug+".md"))
	if err != nil {
		t.Fatalf("reading release file: %v", err)
	}
	content := string(raw)
	if strings.Contains(content, "goal:") {
		t.Errorf("release file still has goal: line after clear; content:\n%s", content)
	}
	if strings.Contains(content, "description:") {
		t.Errorf("release file still has description: line after clear; content:\n%s", content)
	}
}

// TestReleaseGoalDescription_UpdateOnlyGoalNoSpuriousRename verifies that a
// PUT which only changes goal/description (name unchanged) rewrites the file
// with the new values, updates the cache row, and does not trigger the
// rename/propagation path (artifacts_renamed stays 0, slug is unchanged).
func TestReleaseGoalDescription_UpdateOnlyGoalNoSpuriousRename(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":        "v-goal-desc-no-rename",
		"status":      "planned",
		"goal":        "Old goal",
		"description": "Old description",
	})
	id := releaseID(t, data)
	origRel, _ := data["release"].(map[string]any)
	origSlug, _ := origRel["slug"].(string)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-goal-desc-no-rename",
		"status": "planned",
		"goal":   "New goal",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	if renamed, _ := body["artifacts_renamed"].(float64); renamed != 0 {
		t.Errorf("artifacts_renamed: want 0, got %v", renamed)
	}

	rel, _ := body["release"].(map[string]any)
	newSlug, _ := rel["slug"].(string)
	if newSlug != origSlug {
		t.Errorf("slug changed on a goal-only update: want %q, got %q", origSlug, newSlug)
	}
	if got, _ := rel["goal"].(string); got != "New goal" {
		t.Errorf("goal after update: want %q, got %q", "New goal", got)
	}
	if got, _ := rel["description"].(string); got != "Old description" {
		t.Errorf("description should be preserved: want %q, got %q", "Old description", got)
	}

	// Exactly one release file should exist for this slug — no rename artefact.
	entries, err := os.ReadDir(filepath.Join(env.projectRoot, "lifecycle", "releases"))
	if err != nil {
		t.Fatal(err)
	}
	var matches []string
	for _, e := range entries {
		if strings.Contains(e.Name(), "no-rename") {
			matches = append(matches, e.Name())
		}
	}
	if len(matches) != 1 {
		t.Errorf("expected exactly 1 release file for this lineage, got %v", matches)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "releases", newSlug+".md"))
	if err != nil {
		t.Fatalf("reading release file: %v", err)
	}
	if !strings.Contains(string(raw), "goal: New goal") {
		t.Errorf("release file not rewritten with new goal; content:\n%s", string(raw))
	}
}

// TestReleaseGoalDescription_RoundTripFromDisk is the DR-9 key acceptance
// check: create/edit releases via the API, wipe the releases *table* (not the
// files), rehydrate, and verify identical goal/description reproduced from
// disk — the index-is-a-cache invariant.
func TestReleaseGoalDescription_RoundTripFromDisk(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{
		"name":        "v-roundtrip-with",
		"status":      "active",
		"goal":        "Roundtrip goal",
		"description": "Roundtrip description",
	})
	createRelease(t, env, map[string]any{
		"name":   "v-roundtrip-bare",
		"status": "planned",
	})

	store := release.NewStore(env.proj.Idx.DB())

	// Wipe the DB table only — files on disk are untouched.
	if _, err := store.PruneExcept("testproject", nil); err != nil {
		t.Fatalf("PruneExcept: %v", err)
	}
	if n, err := store.Count("testproject"); err != nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatalf("expected 0 rows after wipe, got %d", n)
	}

	result, err := release.Rehydrate(context.Background(), store, "testproject", env.projectRoot)
	if err != nil {
		t.Fatalf("Rehydrate: %v", err)
	}
	if result.Inserted != 2 {
		t.Fatalf("Rehydrate inserted: want 2, got %d (skipped=%d errors=%v)", result.Inserted, result.Skipped, result.Errors)
	}

	withRel, err := store.GetByName("testproject", "v-roundtrip-with")
	if err != nil {
		t.Fatal(err)
	}
	if withRel == nil {
		t.Fatal("v-roundtrip-with missing after rehydrate")
	}
	if withRel.Goal != "Roundtrip goal" {
		t.Errorf("rehydrated goal: want %q, got %q", "Roundtrip goal", withRel.Goal)
	}
	if withRel.Description != "Roundtrip description" {
		t.Errorf("rehydrated description: want %q, got %q", "Roundtrip description", withRel.Description)
	}

	bareRel, err := store.GetByName("testproject", "v-roundtrip-bare")
	if err != nil {
		t.Fatal(err)
	}
	if bareRel == nil {
		t.Fatal("v-roundtrip-bare missing after rehydrate")
	}
	if bareRel.Goal != "" {
		t.Errorf("rehydrated goal for bare release: want \"\", got %q", bareRel.Goal)
	}
	if bareRel.Description != "" {
		t.Errorf("rehydrated description for bare release: want \"\", got %q", bareRel.Description)
	}
}

// TestReleaseGoalDescription_RehydrateExistingFilesNoNewKeys verifies DR-8:
// pre-existing release files that predate this feature (no goal/description
// keys) load through rehydrate with empty fields, no error, and — crucially —
// without the indexer rewriting the file (index is read-only w.r.t. disk).
func TestReleaseGoalDescription_RehydrateExistingFilesNoNewKeys(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	dir := filepath.Join(env.projectRoot, "lifecycle", "releases")
	path := filepath.Join(dir, "legacy-release.md")
	if err := os.WriteFile(path, []byte(makeReleaseMD("Legacy Release", "planned")), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/p/testproject/releases/rehydrate", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	if inserted, _ := body["inserted"].(float64); int(inserted) != 1 {
		t.Errorf("inserted: want 1, got %v", body["inserted"])
	}

	listResp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, listResp, http.StatusOK)
	listBody := readJSON(t, listResp)
	releases, _ := listBody["releases"].([]any)

	var found map[string]any
	for _, raw := range releases {
		r, _ := raw.(map[string]any)
		if name, _ := r["name"].(string); name == "Legacy Release" {
			found = r
		}
	}
	if found == nil {
		t.Fatal("Legacy Release missing from list after rehydrate")
	}
	if got, _ := found["goal"].(string); got != "" {
		t.Errorf("goal for legacy release: want \"\", got %q", got)
	}
	if got, _ := found["description"].(string); got != "" {
		t.Errorf("description for legacy release: want \"\", got %q", got)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("rehydrate rewrote the legacy release file; before:\n%s\nafter:\n%s", before, after)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Errorf("rehydrate touched the legacy release file's mtime: before=%v after=%v", beforeInfo.ModTime(), afterInfo.ModTime())
	}
}
