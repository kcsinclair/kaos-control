package workflow

import "testing"

func TestProductOwnerBypassesAnyTransition(t *testing.T) {
	e := New(nil)

	cases := []struct{ from, to string }{
		{"in-progress", "done"},
		{"draft", "done"},
		{"in-qa", "draft"},
		{"approved", "rejected"},
	}
	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{"product-owner"}) {
			t.Errorf("product-owner should be allowed %s → %s", c.from, c.to)
		}
	}
}

func TestNonProductOwnerStillRestricted(t *testing.T) {
	e := New(nil)

	if e.CanTransition("in-progress", "done", []string{"reviewer"}) {
		t.Error("reviewer should not be allowed in-progress → done")
	}
	if e.CanTransition("draft", "done", []string{"analyst"}) {
		t.Error("analyst should not be allowed draft → done")
	}
}

func TestExistingRulesStillApply(t *testing.T) {
	e := New(nil)

	if !e.CanTransition("draft", "clarifying", []string{"analyst"}) {
		t.Error("analyst should be allowed draft → clarifying")
	}
	if !e.CanTransition("approved", "done", []string{"approver"}) {
		t.Error("approver should be allowed approved → done")
	}
	if !e.CanTransition("in-development", "in-qa", []string{"backend-developer"}) {
		t.Error("backend-developer should be allowed in-development → in-qa")
	}
}

// TestWorkflowPredecessors verifies that each predecessor → active_status pair
// used by the agent-launcher-panels feature is a valid workflow transition for
// the expected role. This ensures eligibility logic in the frontend is grounded
// in the actual workflow engine rather than hardcoded assumptions.
// Covers test plan Milestone 4.
func TestWorkflowPredecessors(t *testing.T) {
	e := New(nil)

	cases := []struct {
		from string
		to   string
		role string
	}{
		// draft → clarifying: analyst (active_status for analyst-requirements agent)
		{"draft", "clarifying", "analyst"},
		// clarifying → planning: analyst (active_status for analyst-planner agent)
		{"clarifying", "planning", "analyst"},
		// planning → in-development: approver (gates the move to development)
		{"planning", "in-development", "approver"},
		// in-development → in-qa: backend-developer (active_status for qa handoff)
		{"in-development", "in-qa", "backend-developer"},
	}

	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{c.role}) {
			t.Errorf("role %q should be allowed to transition %s → %s",
				c.role, c.from, c.to)
		}
	}
}

func TestAllowedTargetsForProductOwnerCoversAllStatuses(t *testing.T) {
	e := New(nil)
	got := e.AllowedTargets("anything", []string{"product-owner"})

	want := []string{
		"clarifying", "planning", "in-development", "in-qa", "approved",
		"done", "draft", "rejected", "abandoned", "blocked",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d allowed targets, got %d (%v)", len(want), len(got), got)
	}
	set := map[string]bool{}
	for _, s := range got {
		set[s] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("expected %q to be in allowed targets, got %v", w, got)
		}
	}
}
