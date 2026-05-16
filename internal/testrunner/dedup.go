// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kaos-control/kaos-control/internal/index"
)

// Deduplicator checks for existing open defects before new ones are created
// and groups failures that share the same root assertion location.
type Deduplicator struct {
	idx *index.Index
}

// NewDeduplicator creates a Deduplicator backed by the given index.
func NewDeduplicator(idx *index.Index) *Deduplicator {
	return &Deduplicator{idx: idx}
}

// autoTestLabel builds the deduplication label for a failure.
// Format: "autotest:<suite>:<normalized-pkg>:<normalized-testname>"
func autoTestLabel(f TestFailure) string {
	pkg := normaliseKey(f.Package)
	name := normaliseKey(f.TestName)
	return fmt.Sprintf("autotest:%s:%s:%s", f.Suite, pkg, name)
}

// autoLocLabel builds the location-based deduplication label.
// Format: "autoloc:<file>:<line>"
func autoLocLabel(f TestFailure) string {
	return fmt.Sprintf("autoloc:%s:%d", normaliseKey(f.File), f.Line)
}

// FindDuplicate searches open defects in the given lineage for one that
// matches f by test identifier, assertion location, or similar error message.
// Returns nil if no duplicate exists.
func (d *Deduplicator) FindDuplicate(f TestFailure, lineage string) (*index.ArtifactRow, error) {
	// Tier 1: label-based lookup by test identifier (fast index path).
	testLabel := autoTestLabel(f)
	rows, _, err := d.idx.List(index.Filter{
		Stage:     "defects",
		Lineage:   lineage,
		Label:     testLabel,
		Unlimited: true,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if isOpenDefect(r) {
			return r, nil
		}
	}

	// Tier 2: label-based lookup by assertion location.
	if f.File != "" && f.Line > 0 {
		locLabel := autoLocLabel(f)
		rows, _, err = d.idx.List(index.Filter{
			Stage:     "defects",
			Lineage:   lineage,
			Label:     locLabel,
			Unlimited: true,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if isOpenDefect(r) {
				return r, nil
			}
		}
	}

	// Tier 3: scan all open defects in the lineage and compare normalised error
	// messages. This is a fallback for defects created before the label scheme
	// was introduced, or when label matching is insufficient.
	if f.ErrorMsg != "" {
		normMsg := NormaliseError(f.ErrorMsg)
		allRows, _, err := d.idx.List(index.Filter{
			Stage:     "defects",
			Lineage:   lineage,
			Unlimited: true,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range allRows {
			if !isOpenDefect(r) {
				continue
			}
			// Check if the defect body contains the normalised error message.
			// We look in the title as an approximation since body isn't indexed.
			if strings.Contains(NormaliseError(r.Title), normMsg) {
				return r, nil
			}
		}
	}

	return nil, nil
}

// GroupByAssertion groups failures that share the same file:line into clusters.
// Each cluster will produce one defect with all test names as witnesses.
// Failures with unknown location (Line == 0) are each placed in their own group.
func (d *Deduplicator) GroupByAssertion(failures []TestFailure) [][]TestFailure {
	type key struct{ file string; line int }

	grouped := make(map[key][]TestFailure)
	var order []key
	var singletons []TestFailure

	for _, f := range failures {
		if f.Line == 0 {
			singletons = append(singletons, f)
			continue
		}
		k := key{file: f.File, line: f.Line}
		if _, exists := grouped[k]; !exists {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], f)
	}

	var result [][]TestFailure
	for _, k := range order {
		result = append(result, grouped[k])
	}
	for _, f := range singletons {
		result = append(result, []TestFailure{f})
	}
	return result
}

// ----- error normalisation -----

var (
	// timestampRe matches ISO dates and common date formats.
	timestampRe = regexp.MustCompile(`\d{4}[-/]\d{2}[-/]\d{2}`)
	// hexPointerRe matches hexadecimal pointer addresses.
	hexPointerRe = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	// uuidRe matches UUID-formatted strings.
	uuidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// NormaliseError strips variable content from an error message (timestamps, hex
// pointers, UUIDs) and returns the first 100 characters. Two errors that differ
// only in these variable parts will produce the same normalised string.
func NormaliseError(msg string) string {
	msg = timestampRe.ReplaceAllString(msg, "<date>")
	msg = hexPointerRe.ReplaceAllString(msg, "<ptr>")
	msg = uuidRe.ReplaceAllString(msg, "<uuid>")
	msg = strings.TrimSpace(msg)
	if len(msg) > 100 {
		msg = msg[:100]
	}
	return msg
}

// ----- helpers -----

// isOpenDefect returns true when the defect status is neither done nor abandoned.
func isOpenDefect(r *index.ArtifactRow) bool {
	return r.Status != "done" && r.Status != "abandoned"
}

// normaliseKey converts a string into a label-safe key by lowercasing and
// replacing sequences of non-alphanumeric characters with a single hyphen.
func normaliseKey(s string) string {
	s = strings.ToLower(s)
	s = nonAlnumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Truncate to keep labels manageable.
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}
