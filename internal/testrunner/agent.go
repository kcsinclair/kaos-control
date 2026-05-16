// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"context"
	"log/slog"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// SuiteCounts holds per-suite test result totals.
type SuiteCounts struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
}

// RunSummary summarises the outcome of a full test-runner agent execution.
type RunSummary struct {
	Go         SuiteCounts
	Vitest     SuiteCounts
	Playwright SuiteCounts
	// DefectsCreated is the number of new defect artifacts created.
	DefectsCreated int
	// DuplicatesFound is the number of failures that matched existing open defects.
	DuplicatesFound int
	// OrphanedFailures is the count of failures with no matching test artifact.
	OrphanedFailures int
	// CoverageGaps lists test artifact paths with no corresponding failure this run.
	CoverageGaps []string
	// Elapsed is the total wall-clock time for the run.
	Elapsed time.Duration
}

// Run executes the full test-runner flow:
//  1. Run all suites via Executor.
//  2. Map each failure to a test artifact.
//  3. Group failures by assertion location.
//  4. Deduplicate: append witness to existing defects, or file new ones.
//  5. Detect coverage gaps.
//  6. Broadcast the summary via hub.
func Run(ctx context.Context, projectDir string, idx *index.Index, h *hub.Hub) (*RunSummary, error) {
	start := time.Now()
	summary := &RunSummary{}

	// Step 1: run all suites.
	exec := &Executor{}
	suiteResults, err := exec.RunAll(ctx, projectDir)
	if err != nil {
		return nil, err
	}

	// Aggregate suite-level counts.
	for _, sr := range suiteResults {
		counts := SuiteCounts{
			Total:   sr.Total,
			Passed:  sr.Passed,
			Failed:  sr.Failed,
			Skipped: sr.Skipped,
		}
		switch sr.Suite {
		case "go":
			summary.Go = counts
		case "vitest":
			summary.Vitest = counts
		case "playwright":
			summary.Playwright = counts
		}
		if sr.RawError != "" {
			slog.Warn("suite produced non-JSON output", "suite", sr.Suite, "error", sr.RawError[:min(len(sr.RawError), 200)])
		}
	}

	// Build per-run components.
	mapper := NewArtifactMapper(idx)
	dedup := NewDeduplicator(idx)
	filer := NewDefectFiler(idx, projectDir)

	// Collect all test artifacts for orphan detection.
	testArtifacts, _, err := idx.List(index.Filter{Stage: "tests", Unlimited: true})
	if err != nil {
		slog.Warn("listing test artifacts for orphan detection", "err", err)
	}

	// Step 2–4: for each suite, map + group + file/deduplicate.
	for _, sr := range suiteResults {
		if len(sr.Failures) == 0 {
			continue
		}

		// Group failures by assertion location within this suite.
		clusters := dedup.GroupByAssertion(sr.Failures)

		for _, cluster := range clusters {
			primary := cluster[0]

			// Map primary failure to a test artifact.
			matched, err := mapper.MapFailure(primary)
			if err != nil {
				slog.Warn("MapFailure", "test", primary.TestName, "err", err)
			}
			if matched == nil {
				summary.OrphanedFailures += len(cluster)
			}

			// Determine lineage for dedup query.
			lineage := "tests-orphaned"
			if matched != nil {
				lineage = matched.Lineage
			}

			// Check for existing open defect.
			dup, err := dedup.FindDuplicate(primary, lineage)
			if err != nil {
				slog.Warn("FindDuplicate", "test", primary.TestName, "err", err)
			}

			if dup != nil {
				// Append witness to existing defect.
				if err := filer.AppendWitness(dup.Path, primary); err != nil {
					slog.Warn("AppendWitness", "defect", dup.Path, "err", err)
				}
				summary.DuplicatesFound++
			} else {
				// File a new defect.
				defectPath, err := filer.FileDefect(cluster, matched)
				if err != nil {
					slog.Warn("FileDefect", "test", primary.TestName, "err", err)
				} else {
					slog.Info("filed defect", "path", defectPath, "test", primary.TestName)
					summary.DefectsCreated++
				}
			}
		}
	}

	// Step 5: detect coverage gaps.
	summary.CoverageGaps = DetectOrphans(suiteResults, testArtifacts)

	// Step 6: record elapsed and broadcast.
	summary.Elapsed = time.Since(start)

	if h != nil {
		h.Broadcast(hub.Event{
			Type:    "testrunner.run.complete",
			Payload: summary,
		})
	}

	slog.Info("test-runner complete",
		"go_failed", summary.Go.Failed,
		"vitest_failed", summary.Vitest.Failed,
		"playwright_failed", summary.Playwright.Failed,
		"defects_created", summary.DefectsCreated,
		"duplicates", summary.DuplicatesFound,
		"orphans", summary.OrphanedFailures,
		"elapsed", summary.Elapsed,
	)

	return summary, nil
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
