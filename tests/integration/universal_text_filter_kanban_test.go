//go:build integration

package integration

// Universal Text Filter — Milestone 3: Kanban Board view UI tests.
//
// These tests verify client-side TextFilter behaviour on the Kanban view:
// input presence, card hiding, empty-column indicator, dropdown composition,
// and clear restoring cards.
//
// IMPORTANT: These tests require browser-level automation (DOM inspection,
// computed CSS style checks). Each test is skipped until a browser
// automation framework is integrated.
//
// Run with:
//   go test ./tests/integration/ -tags integration -run TestUniversalTextFilterKanban

import "testing"

// TestUniversalTextFilterKanban_InputPresent asserts that the Kanban view
// contains a TextFilter input.
func TestUniversalTextFilterKanban_InputPresent(t *testing.T) {
	t.Skip("requires browser automation: assert TextFilter input present on /kanban")
}

// TestUniversalTextFilterKanban_CardsHiddenOnFilter types a search term and
// asserts cards not matching are hidden (absent from DOM or display:none).
func TestUniversalTextFilterKanban_CardsHiddenOnFilter(t *testing.T) {
	t.Skip("requires browser automation: type term, assert non-matching cards are not displayed")
}

// TestUniversalTextFilterKanban_EmptyColumnIndicator types a term that removes
// all cards from at least one column and asserts the column remains visible
// with a 'No matching items' message.
func TestUniversalTextFilterKanban_EmptyColumnIndicator(t *testing.T) {
	t.Skip("requires browser automation: filter to empty column, assert 'No matching items' message visible")
}

// TestUniversalTextFilterKanban_CompositionWithDropdowns combines the text
// filter with a dropdown filter and asserts AND logic is applied.
func TestUniversalTextFilterKanban_CompositionWithDropdowns(t *testing.T) {
	t.Skip("requires browser automation: combine dropdown + text filter, assert AND intersection on cards")
}

// TestUniversalTextFilterKanban_ClearRestoresCards types text, then clears,
// and asserts all cards reappear (subject to dropdown filters).
func TestUniversalTextFilterKanban_ClearRestoresCards(t *testing.T) {
	t.Skip("requires browser automation: clear filter, assert all cards visible again")
}
