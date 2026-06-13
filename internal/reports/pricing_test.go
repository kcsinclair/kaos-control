// SPDX-License-Identifier: AGPL-3.0-or-later

package reports

import (
	"math"
	"testing"
)

// These fixtures are real modelUsage entries pulled from kaos-control's own run
// logs. They double as a guard that the price table reproduces the
// API-reported total cost from the token counts (token×price ≈ recorded
// costUSD) — if Anthropic changes pricing, these go red.
func TestSplitCost_ReconcilesToRecordedTotal(t *testing.T) {
	cases := []struct {
		name                              string
		model                             string
		inputTok, cacheWrite, cacheRead   int64
		outputTok                         int64
		recorded                          float64
		wantOutputCost                    float64 // outputTok × output price
	}{
		{"sonnet-4-6", "claude-sonnet-4-6", 9898, 372107, 13347973, 118446, 7.206177150000002, 118446 * 15.0 / 1e6},
		{"opus-4-6", "claude-opus-4-6", 3811, 56535, 516372, 14496, 0.9929847500000001, 14496 * 25.0 / 1e6},
		{"haiku-4-5 (dated id)", "claude-haiku-4-5-20251001", 741, 0, 0, 14, 0.000811, 14 * 5.0 / 1e6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, out := splitCost(tc.model, tc.inputTok, tc.cacheWrite, tc.cacheRead, tc.outputTok, tc.recorded)

			// Components must reconcile exactly to the authoritative total.
			if got := in + out; math.Abs(got-tc.recorded) > 1e-9 {
				t.Errorf("input+output = %.10f, want recorded total %.10f", got, tc.recorded)
			}
			// And the output share should match list pricing within a cent —
			// confirming the prices (not just the reconciliation) are correct.
			if math.Abs(out-tc.wantOutputCost) > 0.01 {
				t.Errorf("output cost = %.6f, want ~%.6f", out, tc.wantOutputCost)
			}
			if in <= 0 || out <= 0 {
				t.Errorf("expected positive split, got input=%.6f output=%.6f", in, out)
			}
		})
	}
}

func TestSplitCost_UnknownModelNoSplit(t *testing.T) {
	in, out := splitCost("gpt-4o", 1000, 0, 0, 500, 1.23)
	if in != 0 || out != 0 {
		t.Errorf("unknown model should yield no split, got input=%.6f output=%.6f", in, out)
	}
}

func TestSplitCost_NoTokensNoSplit(t *testing.T) {
	in, out := splitCost("claude-sonnet-4-6", 0, 0, 0, 0, 0)
	if in != 0 || out != 0 {
		t.Errorf("zero tokens should yield no split, got input=%.6f output=%.6f", in, out)
	}
}

func TestPriceFor_FamilyPrefixMatch(t *testing.T) {
	for _, m := range []string{"claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8"} {
		if p, ok := priceFor(m); !ok || p.OutputPerMTok != 25 {
			t.Errorf("%s: got ok=%v output=%v, want opus $25/MTok", m, ok, p.OutputPerMTok)
		}
	}
	if _, ok := priceFor("claude-3-5-sonnet"); ok {
		t.Error("claude-3-5-sonnet should not match a 4.x family prefix")
	}
}
