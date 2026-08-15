// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact

import (
	"fmt"
	"math"
)

// RiceValidationError reports a field-level RICE validation failure.
type RiceValidationError struct {
	Field   string
	Message string
}

func (e *RiceValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// RiceScore derives the RICE score from fm's four components:
// (reach × impact × (confidence/100)) / effort.
// ok is true only when all four components are set and each satisfies its
// constraint (see ValidateRiceComponent); otherwise ok is false ("N/A") and
// score is 0. The returned score is never Inf or NaN.
func RiceScore(fm Frontmatter) (score float64, ok bool) {
	if fm.RiceReach == nil || fm.RiceImpact == nil || fm.RiceConfidence == nil || fm.RiceEffort == nil {
		return 0, false
	}
	if ValidateRice(fm) != nil {
		return 0, false
	}
	return (*fm.RiceReach * *fm.RiceImpact * (*fm.RiceConfidence / 100)) / *fm.RiceEffort, true
}

// ValidateRiceComponent validates a single RICE component value against its
// field-specific constraint. A nil value is always valid — it represents the
// unset state, not an error. field must be one of "rice_reach", "rice_impact",
// "rice_confidence", "rice_effort".
func ValidateRiceComponent(field string, v *float64) error {
	if v == nil {
		return nil
	}
	switch field {
	case "rice_reach":
		if *v < 0 {
			return &RiceValidationError{Field: field, Message: "must be >= 0"}
		}
	case "rice_impact":
		if *v < 0 {
			return &RiceValidationError{Field: field, Message: "must be >= 0"}
		}
	case "rice_confidence":
		if *v < 0 || *v > 100 {
			return &RiceValidationError{Field: field, Message: "must be between 0 and 100"}
		}
	case "rice_effort":
		if *v <= 0 {
			return &RiceValidationError{Field: field, Message: "must be > 0"}
		}
	default:
		return &RiceValidationError{Field: field, Message: "unknown RICE field"}
	}
	return nil
}

// ValidateRice validates every RICE component present on fm, returning the
// first field-level error encountered, or nil when all present components
// satisfy their constraints. Components left unset (nil) are not required.
func ValidateRice(fm Frontmatter) error {
	if err := ValidateRiceComponent("rice_reach", fm.RiceReach); err != nil {
		return err
	}
	if err := ValidateRiceComponent("rice_impact", fm.RiceImpact); err != nil {
		return err
	}
	if err := ValidateRiceComponent("rice_confidence", fm.RiceConfidence); err != nil {
		return err
	}
	if err := ValidateRiceComponent("rice_effort", fm.RiceEffort); err != nil {
		return err
	}
	return nil
}

// RoundRice rounds score to 2 decimal places for display. Sorting and storage
// use the unrounded value returned by RiceScore.
func RoundRice(score float64) float64 {
	return math.Round(score*100) / 100
}
