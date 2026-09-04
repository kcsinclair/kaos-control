// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import "testing"

func TestSwitchoverReasons_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(SwitchoverReasons))
	for _, r := range SwitchoverReasons {
		if seen[r] {
			t.Errorf("duplicate reason %q", r)
		}
		seen[r] = true
		if r == "" {
			t.Error("empty reason in SwitchoverReasons")
		}
	}
	if len(SwitchoverReasons) != 12 {
		t.Errorf("expected 12 canonical reasons (FR-2.3 table), got %d: %v", len(SwitchoverReasons), SwitchoverReasons)
	}
}

func TestDefaultSwitchoverAction(t *testing.T) {
	cases := []struct {
		reason  string
		enabled bool
		want    string
	}{
		{string(RateLimitKindRateLimit), false, SwitchoverActionPauseQueue},
		{string(RateLimitKindRateLimit), true, SwitchoverActionFailover},
		{string(RateLimitKindOverloaded), true, SwitchoverActionFailover},
		{string(RateLimitKindUnreachable), true, SwitchoverActionFailover},
		{FailureReasonAuthError, true, SwitchoverActionFailover},
		{FailureReasonAuthError, false, SwitchoverActionPauseQueue},
		{FailureReasonProviderDisconnected, true, SwitchoverActionRetryInPlace},
		{FailureReasonProviderDisconnected, false, SwitchoverActionRetryInPlace},
		{FailureReasonModelNotFound, true, SwitchoverActionFailRun},
		{FailureReasonTimeout, true, SwitchoverActionFailRun},
	}
	for _, c := range cases {
		if got := DefaultSwitchoverAction(c.reason, c.enabled); got != c.want {
			t.Errorf("DefaultSwitchoverAction(%q, %v) = %q, want %q", c.reason, c.enabled, got, c.want)
		}
	}
}
