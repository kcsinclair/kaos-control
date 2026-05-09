// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestFullScanIndexing verifies that on startup the server indexes all seeded
// artifacts and GET /graph returns the expected node/edge counts.
// Test plan §7: "Full scan" scenario.
func TestFullScanIndexing(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/login.md",
			content: makeArtifact("Login Feature", "idea", "draft", "login", "", "User login via email/password."),
		},
		{
			relPath: "lifecycle/requirements/login-2.md",
			content: makeArtifact("Login Requirements", "ticket", "draft", "login",
				"lifecycle/ideas/login.md", "Detailed requirements.\n\nParent: [[ideas/login]]"),
		},
		{
			relPath: "lifecycle/ideas/signup.md",
			content: makeArtifact("Signup Feature", "idea", "draft", "signup", "", "User signup flow."),
		},
		{
			relPath: "lifecycle/requirements/signup-2.md",
			content: makeArtifact("Signup Requirements", "ticket", "planning", "signup",
				"lifecycle/ideas/signup.md", "Signup requirements.\n\nSee also: [[ideas/login]]"),
		},
		{
			relPath: "lifecycle/backend-plans/signup-3-be.md",
			content: makeArtifact("Signup Backend Plan", "plan-backend", "draft", "signup",
				"lifecycle/requirements/signup-2.md", "Backend plan for signup."),
		},
	}

	env := newTestEnv(t, seeds)

	// No auth needed for graph (read-only).
	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	data := readJSON(t, resp)

	nodes, ok := data["nodes"].([]any)
	if !ok {
		t.Fatal("expected nodes array in graph response")
	}
	if len(nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d", len(nodes))
	}

	edges, ok := data["edges"].([]any)
	if !ok {
		t.Fatal("expected edges array in graph response")
	}
	// Expected edges: login-2 → login (parent), signup-2 → signup (parent),
	// signup-3-be → signup-2 (parent), plus wiki links.
	if len(edges) < 3 {
		t.Errorf("expected at least 3 edges, got %d", len(edges))
	}

	// Verify list endpoint returns all 5 artifacts.
	resp2, err := http.Get(env.baseURL + "/api/p/testproject/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	list := readJSON(t, resp2)
	total, _ := list["total"].(float64)
	if int(total) != 5 {
		t.Errorf("expected total=5, got %v", total)
	}

	// Verify filtering by stage.
	resp3, err := http.Get(env.baseURL + "/api/p/testproject/artifacts?stage=ideas")
	if err != nil {
		t.Fatal(err)
	}
	filtered := readJSON(t, resp3)
	filteredTotal, _ := filtered["total"].(float64)
	if int(filteredTotal) != 2 {
		t.Errorf("expected 2 ideas, got %v", filteredTotal)
	}

	// Verify lineages endpoint.
	resp4, err := http.Get(env.baseURL + "/api/p/testproject/lineages")
	if err != nil {
		t.Fatal(err)
	}
	lineagesData := readJSON(t, resp4)
	lineages, ok := lineagesData["lineages"].([]any)
	if !ok {
		t.Fatal("expected lineages array")
	}
	if len(lineages) != 2 {
		t.Errorf("expected 2 lineages (login, signup), got %d", len(lineages))
	}
}

// TestScanWithFilterByStatus verifies filtering by status returns correct results.
func TestScanWithFilterByStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/alpha.md",
			content: makeArtifact("Alpha", "idea", "draft", "alpha", "", "Draft idea."),
		},
		{
			relPath: "lifecycle/ideas/beta.md",
			content: makeArtifact("Beta", "idea", "draft", "beta", "", "Another draft."),
		},
		{
			relPath: "lifecycle/requirements/alpha-2.md",
			content: makeArtifact("Alpha Req", "ticket", "planning", "alpha",
				"lifecycle/ideas/alpha.md", "Planning stage ticket."),
		},
	}

	env := newTestEnv(t, seeds)

	resp, err := http.Get(env.baseURL + "/api/p/testproject/artifacts?status=planning")
	if err != nil {
		t.Fatal(err)
	}
	data := readJSON(t, resp)
	total, _ := data["total"].(float64)
	if int(total) != 1 {
		t.Errorf("expected 1 planning artifact, got %v", total)
	}
}
