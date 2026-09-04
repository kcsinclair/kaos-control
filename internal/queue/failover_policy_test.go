// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import "testing"

func TestFailoverPolicy_ActionFor(t *testing.T) {
	fp := FailoverPolicy{Actions: map[string]string{
		"rate_limit": "failover",
		"timeout":    "fail_run",
	}}

	if got := fp.ActionFor("rate_limit"); got != "failover" {
		t.Errorf("rate_limit: got %q", got)
	}
	if got := fp.ActionFor("timeout"); got != "fail_run" {
		t.Errorf("timeout: got %q", got)
	}
	// Absent reason defaults to the safest fail-closed action.
	if got := fp.ActionFor("unknown_reason"); got != "pause_queue" {
		t.Errorf("unknown reason: got %q, want pause_queue", got)
	}
}

func TestFailoverPolicy_ActionFor_NilActions(t *testing.T) {
	var fp FailoverPolicy
	if got := fp.ActionFor("rate_limit"); got != "pause_queue" {
		t.Errorf("nil Actions map: got %q, want pause_queue", got)
	}
}
