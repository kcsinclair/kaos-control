// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApprovedToDoneByApprover verifies that:
//   - a user with the 'approver' role can transition approved → done
//   - a user without that role (qa, dev) gets 403
//   - the status is persisted to disk after the successful transition
//
// Directly covers the regression scenario from
// lifecycle/defects/product-owner-cannot-transition.md.
func TestApprovedToDoneByApprover(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/approve-done.md",
			content: makeArtifact("Approve Done Test", "ticket", "approved", "approve-done", "", "Testing approved → done."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/approve-done.md"

	// QA user does not hold the 'approver' role — must get 403.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition", map[string]any{
		"to": "done",
	})
	requireStatus(t, resp, 403)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden' for qa user, got %q", code)
	}

	// Dev user does not hold the 'approver' role — must get 403.
	env.login("dev@test.local", "dev-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition", map[string]any{
		"to": "done",
	})
	requireStatus(t, resp, 403)
	data = readJSON(t, resp)
	errData, _ = data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden' for dev user, got %q", code)
	}

	// Admin holds [product-owner, analyst, reviewer, approver] — must succeed.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition", map[string]any{
		"to": "done",
	})
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if status, _ := artifact["status"].(string); status != "done" {
		t.Errorf("expected status 'done', got %q", status)
	}

	// Verify the status is written to disk.
	content, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "requirements", "approve-done.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(content), "status: done") {
		t.Error("status field not updated to 'done' on disk")
	}
}

// TestFullLifecyclePlanningToDone exercises the complete terminal path:
//
//	planning → in-development → in-qa → approved → done
//
// Each step is performed by a user holding the required role:
//   - planning → in-development: approver (admin) — gate satisfied by three approved plans
//   - in-development → in-qa:    backend-developer/test-developer (dev)
//   - in-qa → approved:          qa (qa)
//   - approved → done:           approver (admin)
//
// The test asserts the final 'done' status is persisted to disk and that at
// least two git commits were recorded against the artifact (initial + transitions).
func TestFullLifecyclePlanningToDone(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/full-lifecycle.md",
			content: makeArtifact("Full Lifecycle Idea", "idea", "draft", "full-lifecycle", "", "Originating idea."),
		},
		{
			relPath: "lifecycle/requirements/full-lifecycle-2.md",
			content: makeArtifact("Full Lifecycle Ticket", "ticket", "planning", "full-lifecycle",
				"lifecycle/ideas/full-lifecycle.md", "Ticket ready for development."),
		},
		// All three plan types approved — satisfies the required_plans gate.
		{
			relPath: "lifecycle/backend-plans/full-lifecycle-3-be.md",
			content: makeArtifact("Full Lifecycle BE Plan", "plan-backend", "approved", "full-lifecycle",
				"lifecycle/requirements/full-lifecycle-2.md", "Backend plan."),
		},
		{
			relPath: "lifecycle/frontend-plans/full-lifecycle-4-fe.md",
			content: makeArtifact("Full Lifecycle FE Plan", "plan-frontend", "approved", "full-lifecycle",
				"lifecycle/requirements/full-lifecycle-2.md", "Frontend plan."),
		},
		{
			relPath: "lifecycle/test-plans/full-lifecycle-5-test.md",
			content: makeArtifact("Full Lifecycle Test Plan", "plan-test", "approved", "full-lifecycle",
				"lifecycle/requirements/full-lifecycle-2.md", "Test plan."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/requirements/full-lifecycle-2.md"

	type step struct {
		email    string
		pass     string
		to       string
		wantCode int
	}
	steps := []step{
		// planning → in-development: approver role; gate passes (all plans approved).
		{"admin@test.local", "admin-pass-123", "in-development", 200},
		// in-development → in-qa: backend-developer / test-developer role.
		{"dev@test.local", "dev-pass-123", "in-qa", 200},
		// in-qa → approved: qa role.
		{"qa@test.local", "qa-pass-123", "approved", 200},
		// approved → done: approver role.
		{"admin@test.local", "admin-pass-123", "done", 200},
	}

	for _, s := range steps {
		env.login(s.email, s.pass)
		resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition", map[string]any{
			"to": s.to,
		})
		requireStatus(t, resp, s.wantCode)
		data := readJSON(t, resp)
		artifact, _ := data["artifact"].(map[string]any)
		if status, _ := artifact["status"].(string); status != s.to {
			t.Fatalf("after transition to %q (user %s): got status %q", s.to, s.email, status)
		}
	}

	// Verify final 'done' status is persisted to disk.
	content, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "requirements", "full-lifecycle-2.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(content), "status: done") {
		t.Error("final status 'done' not found on disk")
	}

	// Verify git commits were recorded.
	commits, err := env.proj.Git.Log(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 {
		t.Errorf("expected at least 2 commits (initial + transitions), got %d", len(commits))
	}
}
