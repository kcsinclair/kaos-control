// SPDX-License-Identifier: AGPL-3.0-or-later

// Package release defines the Release entity and its validation rules.
package release

import (
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned by Store methods when the requested release does not exist.
var ErrNotFound = errors.New("release not found")

// ValidStatuses is the set of allowed release status values.
var ValidStatuses = map[string]bool{
	"planned": true,
	"active":  true,
	"shipped": true,
}

// Release is a named, versioned bucket that groups artifacts.
type Release struct {
	ID        int64      `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Summary counts populated by Store.Get.
	IdeaCount   int `json:"idea_count,omitempty"`
	DefectCount int `json:"defect_count,omitempty"`
}

// Validate checks that r satisfies the Release invariants.
func (r *Release) Validate() error {
	var errs []error

	name := []rune(r.Name)
	if len(name) < 1 || len(name) > 120 {
		errs = append(errs, fmt.Errorf("name must be between 1 and 120 characters, got %d", len(name)))
	}

	if !ValidStatuses[r.Status] {
		errs = append(errs, fmt.Errorf("status %q is not valid; must be one of: planned, active, shipped", r.Status))
	}

	if r.StartDate != nil && r.EndDate != nil && r.EndDate.Before(*r.StartDate) {
		errs = append(errs, errors.New("end_date must be on or after start_date"))
	}

	return errors.Join(errs...)
}
