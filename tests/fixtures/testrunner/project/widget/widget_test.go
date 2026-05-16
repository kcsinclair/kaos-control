// Package widget_test contains deliberately failing tests used as fixtures for
// the testrunner integration tests.
package widget_test

import "testing"

// TestWidgetPasses always passes (provides a passing result in fixture output).
func TestWidgetPasses(t *testing.T) {}

// TestWidgetFails always fails with a known, stable error message.
func TestWidgetFails(t *testing.T) {
	t.Error("widget not initialised: fixture failure")
}
