//go:build integration

package integration

import (
	"testing"
)

// TestKanbanGrouping_ArtifactsHaveStatus verifies that the artifact list
// response includes a correct status field for each artifact — the data the
// frontend needs to group artifacts into kanban columns.
// Covers Milestone 2, scenario 1.
func TestKanbanGrouping_ArtifactsHaveStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kgrp-draft.md",
			content: makeArtifact("Draft Idea", "idea", "draft", "kgrp-draft", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/kgrp-approved-2.md",
			content: makeArtifact("Approved Ticket", "ticket", "approved", "kgrp-approved", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/kgrp-in-dev-2.md",
			content: makeArtifact("In Dev Ticket", "ticket", "in-development", "kgrp-in-dev", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/kgrp-done-2.md",
			content: makeArtifact("Done Ticket", "ticket", "done", "kgrp-done", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/kgrp-unknown.md",
			content: makeArtifact("Unknown Status", "idea", "custom-unknown-status", "kgrp-unknown", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected at least one artifact in the response")
	}

	// Build a map of lineage → returned status.
	returned := make(map[string]string)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		lineage, _ := item["lineage"].(string)
		status, _ := item["status"].(string)
		returned[lineage] = status
	}

	expected := map[string]string{
		"kgrp-draft":    "draft",
		"kgrp-approved": "approved",
		"kgrp-in-dev":   "in-development",
		"kgrp-done":     "done",
		"kgrp-unknown":  "custom-unknown-status",
	}
	for lineage, want := range expected {
		got, ok := returned[lineage]
		if !ok {
			t.Errorf("artifact with lineage %q not found in response", lineage)
			continue
		}
		if got != want {
			t.Errorf("lineage %q: expected status %q, got %q", lineage, want, got)
		}
	}
}

// TestKanbanGrouping_ArtifactsHaveCreated verifies that the artifact list
// response includes a non-empty created field for each artifact, which the
// frontend uses for age computation on kanban cards.
// Covers Milestone 2, scenario 2.
func TestKanbanGrouping_ArtifactsHaveCreated(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kgrp-created-1.md",
			content: makeArtifact("Created Test 1", "idea", "draft", "kgrp-created-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/kgrp-created-2.md",
			content: makeArtifact("Created Test 2", "idea", "draft", "kgrp-created-2", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected artifacts in response")
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		path, _ := item["path"].(string)
		created, _ := item["created"].(string)
		if created == "" {
			t.Errorf("artifact %q: expected non-empty created field", path)
		}
	}
}

// TestKanbanGrouping_ArtifactsHaveCardFields verifies that the artifact list
// response includes all fields required for kanban card rendering: title, type,
// priority, labels, lineage, and the frontmatter sub-object.
// Covers Milestone 2, scenario 3.
func TestKanbanGrouping_ArtifactsHaveCardFields(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kgrp-card.md",
			content: makeArtifactWithPriority(
				"Card Fields Idea", "idea", "draft", "kgrp-card",
				"high", "Body.",
				"feature", "ui",
			),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected at least one artifact in response")
	}

	// Find the seeded artifact.
	var artifact map[string]any
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if lineage, _ := item["lineage"].(string); lineage == "kgrp-card" {
			artifact = item
			break
		}
	}
	if artifact == nil {
		t.Fatal("seeded artifact kgrp-card not found in list response")
	}

	// Top-level fields required for kanban card rendering.
	if title, _ := artifact["title"].(string); title != "Card Fields Idea" {
		t.Errorf("expected title %q, got %q", "Card Fields Idea", title)
	}
	if typ, _ := artifact["type"].(string); typ != "idea" {
		t.Errorf("expected type %q, got %q", "idea", typ)
	}
	if lineage, _ := artifact["lineage"].(string); lineage != "kgrp-card" {
		t.Errorf("expected lineage %q, got %q", "kgrp-card", lineage)
	}

	// Frontmatter sub-object must be present.
	fm, ok := artifact["frontmatter"].(map[string]any)
	if !ok || fm == nil {
		t.Fatal("expected frontmatter sub-object in response item")
	}

	// Check frontmatter fields.
	if fmTitle, _ := fm["title"].(string); fmTitle != "Card Fields Idea" {
		t.Errorf("frontmatter.title: expected %q, got %q", "Card Fields Idea", fmTitle)
	}
	if fmPriority, _ := fm["priority"].(string); fmPriority != "high" {
		t.Errorf("frontmatter.priority: expected %q, got %q", "high", fmPriority)
	}
	fmLabels, _ := fm["labels"].([]any)
	if len(fmLabels) != 2 {
		t.Errorf("frontmatter.labels: expected 2, got %d", len(fmLabels))
	}
}

// TestKanbanGrouping_FilterByStatus verifies that the status filter on the
// artifact list endpoint works correctly for kanban column filtering.
// Covers Milestone 2, scenario 4.
func TestKanbanGrouping_FilterByStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kgrp-flt-draft-1.md",
			content: makeArtifact("Draft 1", "idea", "draft", "kgrp-flt-draft-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/kgrp-flt-draft-2.md",
			content: makeArtifact("Draft 2", "idea", "draft", "kgrp-flt-draft-2", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/kgrp-flt-approved-2.md",
			content: makeArtifact("Approved", "ticket", "approved", "kgrp-flt-approved", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=draft", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != 2 {
		t.Errorf("expected total=2 for status=draft filter, got %d", int(total))
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items for status=draft filter, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("filter returned artifact with status %q, expected %q", status, "draft")
		}
	}
}
