// SPDX-License-Identifier: AGPL-3.0-or-later

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
		if !e.CanTransition(c.from, c.to, []string{"product-owner"}, "") {
			t.Errorf("product-owner should be allowed %s → %s", c.from, c.to)
		}
	}
}

func TestNonProductOwnerStillRestricted(t *testing.T) {
	e := New(nil)

	if e.CanTransition("in-progress", "done", []string{"reviewer"}, "") {
		t.Error("reviewer should not be allowed in-progress → done")
	}
	if e.CanTransition("draft", "done", []string{"analyst"}, "") {
		t.Error("analyst should not be allowed draft → done")
	}
}

func TestExistingRulesStillApply(t *testing.T) {
	e := New(nil)

	if !e.CanTransition("draft", "clarifying", []string{"analyst"}, "") {
		t.Error("analyst should be allowed draft → clarifying")
	}
	if !e.CanTransition("approved", "done", []string{"approver"}, "") {
		t.Error("approver should be allowed approved → done")
	}
	if !e.CanTransition("in-development", "in-qa", []string{"backend-developer"}, "") {
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
		// draft → clarifying: analyst (active_status for requirements-analyst agent)
		{"draft", "clarifying", "analyst"},
		// clarifying → planning: analyst (active_status for planning-analyst agent)
		{"clarifying", "planning", "analyst"},
		// planning → in-development: approver (gates the move to development)
		{"planning", "in-development", "approver"},
		// in-development → in-qa: backend-developer (active_status for qa handoff)
		{"in-development", "in-qa", "backend-developer"},
	}

	for _, c := range cases {
		if !e.CanTransition(c.from, c.to, []string{c.role}, "") {
			t.Errorf("role %q should be allowed to transition %s → %s",
				c.role, c.from, c.to)
		}
	}
}

// TestSystemRoleCanBlockFromAnyStatus asserts that the "system" actor may
// transition an artifact to "blocked" from every non-blocked known status.
func TestSystemRoleCanBlockFromAnyStatus(t *testing.T) {
	e := New(nil)

	statuses := []string{
		"raw", "draft", "clarifying", "planning",
		"in-development", "in-qa", "approved",
		"rejected", "abandoned", "done",
	}
	for _, from := range statuses {
		if !e.CanTransition(from, "blocked", []string{"system"}, "") {
			t.Errorf("system should be allowed to transition %s → blocked", from)
		}
	}
}

// TestSystemRoleCanUnblockToDraft asserts that the "system" actor may
// transition a blocked artifact back to draft.
func TestSystemRoleCanUnblockToDraft(t *testing.T) {
	e := New(nil)

	if !e.CanTransition("blocked", "draft", []string{"system"}, "") {
		t.Error("system should be allowed to transition blocked → draft")
	}
}

// TestSystemRoleCannotDoOtherTransitions asserts that the "system" actor is
// not permitted to make transitions other than any→blocked and blocked→draft.
func TestSystemRoleCannotDoOtherTransitions(t *testing.T) {
	e := New(nil)

	cases := []struct{ from, to string }{
		{"draft", "clarifying"},
		{"clarifying", "planning"},
		{"planning", "in-development"},
		{"in-development", "in-qa"},
		{"in-qa", "approved"},
		{"approved", "done"},
		{"draft", "rejected"},
		{"draft", "abandoned"},
		{"draft", "done"},
	}
	for _, c := range cases {
		if e.CanTransition(c.from, c.to, []string{"system"}, "") {
			t.Errorf("system should NOT be allowed to transition %s → %s", c.from, c.to)
		}
	}
}

func TestAllowedTargetsForProductOwnerCoversAllStatuses(t *testing.T) {
	e := New(nil)
	got := e.AllowedTargets("anything", []string{"product-owner"}, "")

	want := []string{
		"raw", "clarifying", "planning", "in-development", "in-qa", "approved",
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

// TestRawToDraftTransitions verifies the new raw → draft transition rules.
func TestRawToDraftTransitions(t *testing.T) {
	e := New(nil)

	allowed := []string{"product-owner", "analyst", "system"}
	for _, role := range allowed {
		if !e.CanTransition("raw", "draft", []string{role}, "idea") {
			t.Errorf("role %q should be allowed to transition raw → draft", role)
		}
	}

	denied := []string{"backend-developer", "qa", "reviewer"}
	for _, role := range denied {
		if e.CanTransition("raw", "draft", []string{role}, "idea") {
			t.Errorf("role %q should NOT be allowed to transition raw → draft", role)
		}
	}
}

// TestDraftToRawTransition verifies that only product-owner may demote draft → raw.
func TestDraftToRawTransition(t *testing.T) {
	e := New(nil)

	if !e.CanTransition("draft", "raw", []string{"product-owner"}, "idea") {
		t.Error("product-owner should be allowed to transition draft → raw")
	}

	denied := []string{"analyst", "system", "backend-developer", "reviewer"}
	for _, role := range denied {
		if e.CanTransition("draft", "raw", []string{role}, "idea") {
			t.Errorf("role %q should NOT be allowed to transition draft → raw", role)
		}
	}
}

// TestRawEscapeHatches verifies that universal escape hatch transitions apply
// from raw status without any new explicit rules.
func TestRawEscapeHatches(t *testing.T) {
	e := New(nil)

	// reviewer can reject from raw
	if !e.CanTransition("raw", "rejected", []string{"reviewer"}, "idea") {
		t.Error("reviewer should be allowed raw → rejected")
	}
	// product-owner can abandon from raw
	if !e.CanTransition("raw", "abandoned", []string{"product-owner"}, "idea") {
		t.Error("product-owner should be allowed raw → abandoned")
	}
	// system can block from raw
	if !e.CanTransition("raw", "blocked", []string{"system"}, "idea") {
		t.Error("system should be allowed raw → blocked")
	}
	// analyst can block from raw
	if !e.CanTransition("raw", "blocked", []string{"analyst"}, "idea") {
		t.Error("analyst should be allowed raw → blocked")
	}
}

// TestAllowedTargetsFromRawForAnalyst verifies AllowedTargets from raw for analyst.
func TestAllowedTargetsFromRawForAnalyst(t *testing.T) {
	e := New(nil)
	targets := e.AllowedTargets("raw", []string{"analyst"}, "idea")

	set := map[string]bool{}
	for _, t := range targets {
		set[t] = true
	}
	if !set["draft"] {
		t.Error("AllowedTargets from raw for analyst should include 'draft'")
	}
	if !set["blocked"] {
		t.Error("AllowedTargets from raw for analyst should include 'blocked'")
	}
	if set["raw"] {
		t.Error("AllowedTargets from raw for analyst should NOT include 'raw' (self-transition guard)")
	}
}
