// Package statuscheck implements the lineage staleness detection algorithm.
// Given a set of artifacts sharing a lineage, it walks from parent to child
// and identifies artifacts whose status lags behind all their actively-progressing children.
package statuscheck

import (
	"fmt"

	"github.com/kaos-control/kaos-control/internal/index"
)

// statusOrder defines the canonical forward progression of workflow statuses.
var statusOrder = []string{
	"draft", "clarifying", "planning", "in-development", "in-qa", "approved", "done",
}

// terminalStatuses are excluded from staleness comparison.
var terminalStatuses = map[string]bool{
	"rejected":  true,
	"abandoned": true,
	"blocked":   true,
}

// statusRank returns the position of s in statusOrder, or -1 if not present.
func statusRank(s string) int {
	for i, v := range statusOrder {
		if v == s {
			return i
		}
	}
	return -1
}

// ChildInfo describes a direct child artifact in a lineage, including its
// current workflow status. This allows the frontend to display child statuses
// and perform WebSocket relevance matching.
type ChildInfo struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Result describes a single stale artifact and the recommended advancement.
// The CanAdvance and BlockedReason fields are filled in by the caller (they
// require the workflow engine and user context).
type Result struct {
	Path            string      `json:"path"`
	Lineage         string      `json:"lineage"`
	Type            string      `json:"type"`
	CurrentStatus   string      `json:"current_status"`
	SuggestedStatus string      `json:"suggested_status"`
	Reason          string      `json:"reason"`
	Children        []ChildInfo `json:"children"`
	CanAdvance      bool        `json:"can_advance"`
	BlockedReason   string      `json:"blocked_reason,omitempty"`
}

// Check runs the staleness algorithm over a flat slice of artifacts that share
// a lineage. It returns one Result per stale parent. The algorithm is pure:
// it does no I/O and accepts no workflow engine — callers fill in CanAdvance
// and BlockedReason after the fact.
//
// Rules:
//   - A single-artifact lineage (no children) never reports staleness.
//   - Terminal statuses (rejected, abandoned, blocked) are never considered stale.
//   - An artifact is stale when every non-terminal direct child has a status
//     strictly later in the order than the parent's current status.
//   - The suggested target status is the minimum status among all non-terminal
//     children (the furthest the parent can validly advance to in one step).
func Check(artifacts []*index.ArtifactRow) []Result {
	if len(artifacts) <= 1 {
		return nil
	}

	// Build path → row map for quick child lookups.
	byPath := make(map[string]*index.ArtifactRow, len(artifacts))
	for _, a := range artifacts {
		byPath[a.Path] = a
	}

	// Build parent-path → []child-path map using FM.Parent.
	children := make(map[string][]string, len(artifacts))
	for _, a := range artifacts {
		if a.FM.Parent != "" {
			children[a.FM.Parent] = append(children[a.FM.Parent], a.Path)
		}
	}

	var results []Result

	for _, a := range artifacts {
		// Skip artifacts in terminal statuses — they are never stale.
		if terminalStatuses[a.Status] {
			continue
		}

		childPaths, hasChildren := children[a.Path]
		if !hasChildren {
			// Leaf nodes are never stale.
			continue
		}

		// Collect non-terminal direct children.
		var activeChildren []*index.ArtifactRow
		for _, cp := range childPaths {
			if child, ok := byPath[cp]; ok && !terminalStatuses[child.Status] {
				activeChildren = append(activeChildren, child)
			}
		}

		if len(activeChildren) == 0 {
			// All children are terminal — parent is not stale.
			continue
		}

		parentRank := statusRank(a.Status)

		// Verify every active child is strictly ahead of the parent.
		allAhead := true
		minChildRank := -1
		var childInfos []ChildInfo
		for _, child := range activeChildren {
			cr := statusRank(child.Status)
			if cr == -1 || cr <= parentRank {
				allAhead = false
				break
			}
			if minChildRank == -1 || cr < minChildRank {
				minChildRank = cr
			}
			childInfos = append(childInfos, ChildInfo{Path: child.Path, Status: child.Status})
		}

		if !allAhead || minChildRank == -1 {
			continue
		}

		suggestedStatus := statusOrder[minChildRank]
		results = append(results, Result{
			Path:            a.Path,
			Lineage:         a.Lineage,
			Type:            a.Type,
			CurrentStatus:   a.Status,
			SuggestedStatus: suggestedStatus,
			Reason: fmt.Sprintf(
				"all active children have advanced to at least %q; parent is still %q",
				suggestedStatus, a.Status,
			),
			Children: childInfos,
		})
	}

	return results
}
