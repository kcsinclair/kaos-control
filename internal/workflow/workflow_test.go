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
