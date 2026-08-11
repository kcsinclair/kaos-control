// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestReleases_AddressableBySlugSurvivesIdChange verifies release-artefacts-9
// DR-5: releases are addressable by their durable slug, so a request survives an
// index rebuild that reassigns the cache-local autoincrement id. It reproduces
// the exact "release not found" scenario (a stale id) and shows the slug still
// resolves.
func TestReleases_AddressableBySlugSurvivesIdChange(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "Slug Key Rel", "status": "planned"})
	rel, _ := data["release"].(map[string]any)
	oldID := int64(rel["id"].(float64))
	slug, _ := rel["slug"].(string)
	if slug == "" {
		t.Fatal("create response missing slug")
	}

	slugPath := "/api/p/testproject/releases/" + slug

	// GET by slug works up front.
	resp := env.doRequest("GET", slugPath, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Simulate an index rebuild reassigning the autoincrement id (dropAndRecreate
	// + rehydrate does this — see releases_prune_test / the DR-5 discussion).
	newID := oldID + 1000
	if _, err := env.proj.Idx.DB().Exec(`UPDATE releases SET id=? WHERE id=?`, newID, oldID); err != nil {
		t.Fatalf("reassign release id: %v", err)
	}

	// The old id is gone — addressing by it now 404s (the reported bug).
	resp = env.doRequest("GET", releasePath(oldID), nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()

	// But the slug still resolves for GET …
	resp = env.doRequest("GET", slugPath, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// … and for PUT — the update succeeds despite the id change.
	resp = env.doRequest("PUT", slugPath, map[string]any{"name": "Slug Key Rel", "status": "active"})
	requireStatus(t, resp, http.StatusOK)
	got := readJSON(t, resp)
	gotRel, _ := got["release"].(map[string]any)
	if s, _ := gotRel["status"].(string); s != "active" {
		t.Errorf("status after slug PUT = %q, want active", s)
	}
}

// TestReleases_NumericIdStillWorks verifies the backward-compatibility path: a
// numeric ref that matches no slug is resolved as an autoincrement id, so
// existing id-keyed links keep working.
func TestReleases_NumericIdStillWorks(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "Id Compat Rel", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	rel, _ := body["release"].(map[string]any)
	if name, _ := rel["name"].(string); name != "Id Compat Rel" {
		t.Errorf("GET by id returned name %q, want %q", name, "Id Compat Rel")
	}
	resp.Body.Close()
}
