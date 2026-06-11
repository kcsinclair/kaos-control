// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// ── Milestone T5: Roadmap regression tests ───────────────────────────────────

// TestReleaseListOrdering verifies that GET /releases returns scheduled
// releases (those with a start_date) before unscheduled ones, and within
// each group items are ordered by start_date then name.
func TestReleaseListOrdering(t *testing.T) {
	env := newTestEnv(t, nil)

	// Create releases in deliberately unordered sequence.
	// Unscheduled (no dates).
	createRelease(t, env, map[string]any{"name": "Zzz Unscheduled", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "Aaa Unscheduled", "status": "planned"})
	// Scheduled.
	createRelease(t, env, map[string]any{
		"name": "Later Scheduled", "status": "planned",
		"start_date": "2026-07-01", "end_date": "2026-09-30",
	})
	createRelease(t, env, map[string]any{
		"name": "Earlier Scheduled", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	relList, _ := body["releases"].([]any)

	if len(relList) < 4 {
		t.Fatalf("want 4 releases, got %d", len(relList))
	}

	names := make([]string, len(relList))
	for i, r := range relList {
		rel, _ := r.(map[string]any)
		names[i], _ = rel["name"].(string)
	}

	// First two must be scheduled (have start_date), earlier one first.
	if names[0] != "Earlier Scheduled" {
		t.Errorf("names[0]: want %q, got %q", "Earlier Scheduled", names[0])
	}
	if names[1] != "Later Scheduled" {
		t.Errorf("names[1]: want %q, got %q", "Later Scheduled", names[1])
	}
	// Next two must be unscheduled, alphabetical by name.
	if names[2] != "Aaa Unscheduled" {
		t.Errorf("names[2]: want %q, got %q", "Aaa Unscheduled", names[2])
	}
	if names[3] != "Zzz Unscheduled" {
		t.Errorf("names[3]: want %q, got %q", "Zzz Unscheduled", names[3])
	}
}

// TestReleaseRenamePropagatesArtifactFrontmatter creates an idea with a
// release field, renames the release, and asserts the idea's on-disk
// frontmatter reflects the new release name.
func TestReleaseRenamePropagatesArtifactFrontmatter(t *testing.T) {
	const (
		oldRelease = "Roadmap Reg Old"
		newRelease = "Roadmap Reg New"
	)
	ideaRelPath := "lifecycle/ideas/roadmap-reg-rename-idea.md"
	seeds := []seedArtifact{
		{
			relPath: ideaRelPath,
			content: makeArtifactWithRelease("Roadmap Reg Idea", "idea", "draft",
				"roadmap-reg-rename", oldRelease, "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	data := createRelease(t, env, map[string]any{"name": oldRelease, "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   newRelease,
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Give the propagation watcher time to finish.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got := readArtifactRelease(t, env.projectRoot, ideaRelPath)
		if got == newRelease {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	got := readArtifactRelease(t, env.projectRoot, ideaRelPath)
	if got != newRelease {
		t.Errorf("idea frontmatter release: want %q, got %q", newRelease, got)
	}
}

// TestKanbanFilterByRelease creates a release, assigns two ideas to it,
// and verifies GET /artifacts?release=<name> returns only those two ideas.
func TestKanbanFilterByRelease(t *testing.T) {
	const releaseName = "Kanban Filter Release"
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kfr-idea-a.md",
			content: makeArtifactWithRelease("KFR Idea A", "idea", "draft",
				"kfr-idea-a", releaseName, "Body."),
		},
		{
			relPath: "lifecycle/ideas/kfr-idea-b.md",
			content: makeArtifactWithRelease("KFR Idea B", "idea", "draft",
				"kfr-idea-b", releaseName, "Body."),
		},
		{
			relPath: "lifecycle/ideas/kfr-idea-unrelated.md",
			content: makeArtifactWithRelease("KFR Unrelated", "idea", "draft",
				"kfr-idea-unrelated", "Other Release", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Wait for the startup index scan to finish (may be async on slow CI).
	time.Sleep(200 * time.Millisecond)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release="+urlEscape(releaseName), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	artifacts, _ := body["artifacts"].([]any)

	if len(artifacts) != 2 {
		// Collect titles for diagnosis.
		var titles []string
		for _, a := range artifacts {
			art, _ := a.(map[string]any)
			fm, _ := art["frontmatter"].(map[string]any)
			if title, _ := fm["title"].(string); title != "" {
				titles = append(titles, title)
			}
		}
		t.Errorf("filter by release %q: want 2 artifacts, got %d: %v",
			releaseName, len(artifacts), titles)
	}

	for _, a := range artifacts {
		art, _ := a.(map[string]any)
		p, _ := art["path"].(string)
		if strings.Contains(p, "kfr-idea-unrelated") {
			t.Error("unrelated idea should not appear in release filter results")
		}
	}
}

// urlEscape replaces spaces with %20 for use in query parameters.
// A proper url.QueryEscape would encode '+' which is equivalent in query strings,
// but this keeps the test file dependency-light.
func urlEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

