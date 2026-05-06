//go:build integration

package integration

// Universal Text Filter — Milestone 5: Project Feed view UI tests.
//
// These tests verify TextFilter behaviour on the Project Feed view including
// entry filtering, composition with event-type toggles, and clear restoring
// all entries.
//
// IMPORTANT: These tests require browser-level automation. Each test is
// skipped until a browser automation framework is integrated.
//
// Run with:
//   go test ./tests/integration/ -tags integration -run TestUniversalTextFilterFeed

import "testing"

// TestUniversalTextFilterFeed_InputPresent asserts that the Project Feed view
// contains a TextFilter input.
func TestUniversalTextFilterFeed_InputPresent(t *testing.T) {
	t.Skip("requires browser automation: assert TextFilter input present on /feed")
}

// TestUniversalTextFilterFeed_EntriesHiddenOnFilter types a substring of a
// known event summary and asserts only matching feed entries are visible.
func TestUniversalTextFilterFeed_EntriesHiddenOnFilter(t *testing.T) {
	t.Skip("requires browser automation: type event summary substring, assert only matching entries visible")
}

// TestUniversalTextFilterFeed_CompositionWithTypeToggles enables specific feed
// event types AND types a search string; asserts both filters are applied.
func TestUniversalTextFilterFeed_CompositionWithTypeToggles(t *testing.T) {
	t.Skip("requires browser automation: enable event-type toggles + type text, assert AND intersection")
}

// TestUniversalTextFilterFeed_ClearRestoresEntries types text, then clears,
// and asserts all entries reappear (subject to active type toggles).
func TestUniversalTextFilterFeed_ClearRestoresEntries(t *testing.T) {
	t.Skip("requires browser automation: clear filter, assert all entries visible again")
}
