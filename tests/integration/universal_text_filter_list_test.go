//go:build integration

package integration

// Universal Text Filter — Milestone 2: Artifact List view UI tests.
//
// These tests verify TextFilter behaviour on the Artifact List view:
// input presence, real-time filtering, title highlighting, clear button,
// dropdown composition, pagination reset, and empty state.
//
// IMPORTANT: These tests require browser-level automation (DOM inspection,
// simulated user interaction). The project currently has no browser
// automation framework. Each test is skipped until a framework such as
// Playwright, chromedp, or rod is integrated. The test bodies document
// the exact scenario each test must cover so that implementation is
// unambiguous once the infrastructure is in place.
//
// Run with:
//   go test ./tests/integration/ -tags integration -run TestUniversalTextFilterList

import "testing"

// TestUniversalTextFilterList_InputPresent asserts that the Artifact List view
// contains a text input with aria-label="Filter artifacts by text".
func TestUniversalTextFilterList_InputPresent(t *testing.T) {
	t.Skip("requires browser automation: assert input[aria-label='Filter artifacts by text'] is present on /artifacts")
}

// TestUniversalTextFilterList_RealtimeFiltering types a known artifact title
// substring into the filter input and asserts that the table updates to show
// only matching rows without a page reload.
func TestUniversalTextFilterList_RealtimeFiltering(t *testing.T) {
	t.Skip("requires browser automation: type substring, assert table rows update to matching artifacts only")
}

// TestUniversalTextFilterList_TitleHighlighting types a search term and asserts
// that the title column of matching rows contains a <mark> element wrapping the
// matched substring.
func TestUniversalTextFilterList_TitleHighlighting(t *testing.T) {
	t.Skip("requires browser automation: assert <mark> wraps matched substring in title column cells")
}

// TestUniversalTextFilterList_ClearButton types text, clicks the clear (×)
// button, and asserts the filter input becomes empty and the full list is
// restored.
func TestUniversalTextFilterList_ClearButton(t *testing.T) {
	t.Skip("requires browser automation: click clear button, assert input empty and full list visible")
}

// TestUniversalTextFilterList_CompositionWithDropdowns selects a status
// dropdown value AND types a search string; asserts only artifacts matching
// both conditions appear (AND logic).
func TestUniversalTextFilterList_CompositionWithDropdowns(t *testing.T) {
	t.Skip("requires browser automation: combine status dropdown + text filter, assert AND intersection")
}

// TestUniversalTextFilterList_PaginationReset navigates to page 2, types a
// search term, and asserts the view resets to page 1.
func TestUniversalTextFilterList_PaginationReset(t *testing.T) {
	t.Skip("requires browser automation: navigate to page 2, type search term, assert page resets to 1")
}

// TestUniversalTextFilterList_EmptyResults types a string matching no
// artifacts and asserts the table shows an appropriate empty-state message.
func TestUniversalTextFilterList_EmptyResults(t *testing.T) {
	t.Skip("requires browser automation: type non-matching string, assert empty-state element visible")
}
