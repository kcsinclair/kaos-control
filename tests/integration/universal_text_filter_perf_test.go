// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Universal Text Filter — Milestone 7: Performance validation tests.
//
// These tests verify that client-side filtering meets the < 16 ms budget for
// datasets of 500 artifacts and that rapid typing is properly debounced.
//
// IMPORTANT: Measuring time-from-input-event-to-render-completion and
// verifying debounce behaviour require browser-level automation with
// performance.mark/measure or similar instrumentation. Each test is skipped
// until a browser automation framework with timing API access is integrated.
//
// Run with:
//   go test ./tests/integration/ -tags integration -run TestUniversalTextFilterPerf

import "testing"

// TestUniversalTextFilterPerf_500ArtifactDataset seeds 500 artifacts,
// navigates to the Kanban view (heaviest client-side filter), types a search
// term, and asserts that the time from input event to render completion is
// less than 16 ms (one animation frame).
func TestUniversalTextFilterPerf_500ArtifactDataset(t *testing.T) {
	t.Skip("requires browser automation with timing instrumentation: measure filter render time <= 16ms for 500 artifacts")
}

// TestUniversalTextFilterPerf_DebouncePreventJank simulates typing 10
// characters in under 100 ms and asserts that filtering is invoked at most
// once after the debounce period, not once per keystroke.
func TestUniversalTextFilterPerf_DebouncePreventJank(t *testing.T) {
	t.Skip("requires browser automation: rapid keystrokes, assert filter invoked <= 1 time after debounce period")
}
