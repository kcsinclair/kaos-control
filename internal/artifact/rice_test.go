// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact_test

import (
	"math"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

func f64(v float64) *float64 { return &v }

func riceFM(reach, impact, confidence, effort *float64) artifact.Frontmatter {
	return artifact.Frontmatter{
		RiceReach:      reach,
		RiceImpact:     impact,
		RiceConfidence: confidence,
		RiceEffort:     effort,
	}
}

// TestRiceScore_AllValid verifies the formula is applied unrounded when all
// four components are present and satisfy their constraints.
func TestRiceScore_AllValid(t *testing.T) {
	fm := riceFM(f64(100), f64(2), f64(50), f64(4))
	score, ok := artifact.RiceScore(fm)
	if !ok {
		t.Fatal("expected ok=true for all-valid components")
	}
	want := (100.0 * 2.0 * (50.0 / 100.0)) / 4.0 // = 25
	if score != want {
		t.Errorf("RiceScore: want %v, got %v", want, score)
	}
}

// TestRiceScore_MissingComponent verifies any single missing component makes
// the score N/A (ok=false), regardless of which one.
func TestRiceScore_MissingComponent(t *testing.T) {
	valid := f64(1)
	cases := []struct {
		name                              string
		reach, impact, confidence, effort *float64
	}{
		{"missing reach", nil, valid, valid, valid},
		{"missing impact", valid, nil, valid, valid},
		{"missing confidence", valid, valid, nil, valid},
		{"missing effort", valid, valid, valid, nil},
		{"all missing", nil, nil, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ok := artifact.RiceScore(riceFM(c.reach, c.impact, c.confidence, c.effort))
			if ok {
				t.Errorf("expected ok=false for %s", c.name)
			}
		})
	}
}

// TestRiceScore_EffortNonPositive verifies effort <= 0 yields N/A with no
// panic and no Inf/NaN result.
func TestRiceScore_EffortNonPositive(t *testing.T) {
	for _, effort := range []float64{0, -1} {
		fm := riceFM(f64(1), f64(1), f64(1), f64(effort))
		score, ok := artifact.RiceScore(fm)
		if ok {
			t.Errorf("effort=%v: expected ok=false", effort)
		}
		if math.IsInf(score, 0) || math.IsNaN(score) {
			t.Errorf("effort=%v: got non-finite score %v", effort, score)
		}
	}
}

// TestRiceScore_ConfidenceBounds verifies confidence outside [0,100] is
// invalid while the boundary values 0 and 100 are valid.
func TestRiceScore_ConfidenceBounds(t *testing.T) {
	cases := []struct {
		confidence float64
		wantOK     bool
	}{
		{-1, false},
		{0, true},
		{100, true},
		{101, false},
	}
	for _, c := range cases {
		fm := riceFM(f64(1), f64(1), f64(c.confidence), f64(1))
		_, ok := artifact.RiceScore(fm)
		if ok != c.wantOK {
			t.Errorf("confidence=%v: want ok=%v, got %v", c.confidence, c.wantOK, ok)
		}
	}
}

// TestRiceScore_ZeroReachOrImpact verifies reach=0 or impact=0 with the other
// three components valid yields a real zero score (distinct from N/A).
func TestRiceScore_ZeroReachOrImpact(t *testing.T) {
	t.Run("reach zero", func(t *testing.T) {
		score, ok := artifact.RiceScore(riceFM(f64(0), f64(2), f64(50), f64(4)))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if score != 0 {
			t.Errorf("want score=0, got %v", score)
		}
	})
	t.Run("impact zero", func(t *testing.T) {
		score, ok := artifact.RiceScore(riceFM(f64(10), f64(0), f64(50), f64(4)))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if score != 0 {
			t.Errorf("want score=0, got %v", score)
		}
	})
}

// TestRiceScore_NegativeReachOrImpact verifies reach<0 or impact<0 is invalid.
func TestRiceScore_NegativeReachOrImpact(t *testing.T) {
	if _, ok := artifact.RiceScore(riceFM(f64(-1), f64(1), f64(1), f64(1))); ok {
		t.Error("negative reach: expected ok=false")
	}
	if _, ok := artifact.RiceScore(riceFM(f64(1), f64(-1), f64(1), f64(1))); ok {
		t.Error("negative impact: expected ok=false")
	}
}

// TestValidateRiceComponent verifies field-level constraint checks, including
// that a nil value is always valid (it represents "unset", not an error).
func TestValidateRiceComponent(t *testing.T) {
	if err := artifact.ValidateRiceComponent("rice_reach", nil); err != nil {
		t.Errorf("nil value: want nil error, got %v", err)
	}
	if err := artifact.ValidateRiceComponent("rice_reach", f64(-1)); err == nil {
		t.Error("rice_reach=-1: want error")
	}
	if err := artifact.ValidateRiceComponent("rice_impact", f64(-1)); err == nil {
		t.Error("rice_impact=-1: want error")
	}
	if err := artifact.ValidateRiceComponent("rice_confidence", f64(-1)); err == nil {
		t.Error("rice_confidence=-1: want error")
	}
	if err := artifact.ValidateRiceComponent("rice_confidence", f64(101)); err == nil {
		t.Error("rice_confidence=101: want error")
	}
	if err := artifact.ValidateRiceComponent("rice_effort", f64(0)); err == nil {
		t.Error("rice_effort=0: want error")
	}
	if err := artifact.ValidateRiceComponent("rice_effort", f64(-1)); err == nil {
		t.Error("rice_effort=-1: want error")
	}
}

// TestValidateRice_PartialComponents verifies ValidateRice only checks
// present components — a partially filled-in RICE set (as PATCH allows) is
// valid as long as what is present satisfies its constraint.
func TestValidateRice_PartialComponents(t *testing.T) {
	fm := riceFM(f64(1), nil, nil, nil)
	if err := artifact.ValidateRice(fm); err != nil {
		t.Errorf("partial valid components: want nil error, got %v", err)
	}
	fm = riceFM(f64(-1), nil, nil, nil)
	if err := artifact.ValidateRice(fm); err == nil {
		t.Error("partial invalid component: want error")
	}
}

// TestRoundRice verifies 2-dp rounding for display, distinct from the
// unrounded value RiceScore itself returns.
func TestRoundRice(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{25.0 / 3.0, 8.33},
		{1.005, 1.0}, // float64 representation of 1.005 rounds down
		{0, 0},
	}
	for _, c := range cases {
		if got := artifact.RoundRice(c.in); got != c.want {
			t.Errorf("RoundRice(%v): want %v, got %v", c.in, c.want, got)
		}
	}
}
