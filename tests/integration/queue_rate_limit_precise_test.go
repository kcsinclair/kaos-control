// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/rate-limit-event-detection-5-test.md
// (Milestone 4 — dispatcher prefers precise reset).
// Requirement: lifecycle/requirements/rate-limit-event-detection-2.md
// (FR6-FR8, AC7).
//
// Mode 2: when a run has seen a mid-stream `rate_limit_event` before a Format
// 1-3 denial, the supervisor attaches the cached ResetsAtUnix to the
// `queue.rate_limit` hub event (agent.go:809-824), and the dispatcher's
// handleRateLimit (dispatcher.go:439) uses it directly instead of calling
// ParseResetTime on the denial's raw text.
//
// AC7's fallback half ("resets_at_unix absent/zero → dispatcher pauses using
// the text-parsed value, byte-identical to current behaviour") is already
// covered by the pre-existing tests in queue_rate_limit_test.go — those
// fixtures never emit a rate_limit_event, so they continue to exercise the
// unmodified ParseResetTime path unchanged by this feature. This file adds
// only the new precise-reset behaviour.

import (
	"fmt"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/queue"
)

// queueResumeGraceDefault mirrors internal/queue/dispatcher.go's
// Config.resumeGrace() default (5 minutes), applied when newQueueTestEnv does
// not override it.
const queueResumeGraceDefault = 5 * time.Minute

// rateLimitEventThenTextErrorScript builds a fake-claude script body that
// emits a mid-stream `rate_limit_event` carrying resetsAtUnix, followed by a
// Format-2 rate_limit_error denial whose raw text is independently
// parseable to a different time. The rate_limit_event line must precede the
// denial so the supervisor's per-run quota cache is populated before the
// denial is forwarded (agent.go:818).
func rateLimitEventThenTextErrorScript(resetsAtUnix int64, errorText string) string {
	eventLine := fmt.Sprintf(
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour","resetsAt":%d,"isUsingOverage":false,"overageStatus":"rejected","overageDisabledReason":""}}`,
		resetsAtUnix)
	return "printf '%s\\n' '" + eventLine + "'\n" +
		"printf '%s\\n' '{\"error\":{\"type\":\"rate_limit_error\",\"message\":\"" + errorText + "\"}}'\n" +
		"exit 0\n"
}

// TestQueue_RateLimit_PrecisePreferred (AC7 precise): a rate_limit_event with
// a precise resetsAt is observed before a Format-2 denial whose raw text
// ParseResetTime would resolve to a materially different time. The queue must
// pause using the precise value (+ resume grace), proving it bypasses
// ParseResetTime rather than merely happening to agree with it.
func TestQueue_RateLimit_PrecisePreferred(t *testing.T) {
	now := time.Now()
	resetsAtUnix := now.Add(90 * time.Minute).Unix()
	const errorText = "resets 8pm (Australia/Brisbane)"

	// Sanity check: the two candidate reset times must actually differ, or
	// this test wouldn't distinguish "precise wins" from "coincidence".
	parsed, ok := queue.ParseResetTime(errorText, now)
	if !ok {
		t.Fatalf("test setup: ParseResetTime(%q) unexpectedly failed to parse", errorText)
	}
	if parsed.Unix() == resetsAtUnix {
		t.Fatalf("test setup: chosen resetsAtUnix (%d) coincides with the text-parsed time (%d); pick a different offset", resetsAtUnix, parsed.Unix())
	}

	setupFakeClaudeWithScript(t, rateLimitEventThenTextErrorScript(resetsAtUnix, errorText))

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/precise-reset-idea.md",
			content: makeApprovedArtifact("Precise Reset Idea", "idea", "precise-reset-idea"),
		},
	})

	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/precise-reset-idea.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)

	deadline := time.Now().Add(15 * time.Second)
	var snap map[string]any
	for time.Now().Before(deadline) {
		snap = env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if paused, _ := snap["paused"].(bool); !paused {
		t.Fatal("expected queue to be paused after rate-limit denial")
	}

	wantPausedUntil := time.Unix(resetsAtUnix, 0).UTC().Add(queueResumeGraceDefault).Format(time.RFC3339)
	gotPausedUntil, _ := snap["paused_until"].(string)
	if gotPausedUntil != wantPausedUntil {
		t.Errorf("paused_until: got %q, want %q (precise resetsAtUnix=%d + resume grace)", gotPausedUntil, wantPausedUntil, resetsAtUnix)
	}

	// Confirm it did NOT fall back to the text-parsed value.
	notWantPausedUntil := parsed.UTC().Add(queueResumeGraceDefault).Format(time.RFC3339)
	if gotPausedUntil == notWantPausedUntil {
		t.Errorf("paused_until: got %q, matches the text-parsed fallback — precise resetsAtUnix was not preferred", gotPausedUntil)
	}
}

// TestQueue_RateLimit_OverloadedFallback_NoResetsAtUnix (AC7 fallback
// sub-case): an "overloaded" denial with no prior rate_limit_event (so
// resets_at_unix is absent) and unparseable raw text must still fall back to
// OverloadPause (~5 min), not the longer default FallbackPause (~30 min) used
// for plain rate-limit denials — proving the kind-based fallback selection is
// unaffected by this feature when no precise reset is available.
func TestQueue_RateLimit_OverloadedFallback_NoResetsAtUnix(t *testing.T) {
	setupFakeClaudeWithScript(t, "printf '%s\\n' '{\"error\":{\"type\":\"overloaded_error\",\"message\":\"529 Overloaded, please retry\"}}'\nexit 0\n")

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/overloaded-fallback-idea.md",
			content: makeApprovedArtifact("Overloaded Fallback Idea", "idea", "overloaded-fallback-idea"),
		},
	})

	before := time.Now()
	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/overloaded-fallback-idea.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)

	deadline := time.Now().Add(15 * time.Second)
	var snap map[string]any
	for time.Now().Before(deadline) {
		snap = env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if paused, _ := snap["paused"].(bool); !paused {
		t.Fatal("expected queue to be paused after overloaded denial")
	}

	pausedUntilStr, _ := snap["paused_until"].(string)
	pausedUntil, err := time.Parse(time.RFC3339, pausedUntilStr)
	if err != nil {
		t.Fatalf("paused_until %q did not parse as RFC3339: %v", pausedUntilStr, err)
	}

	// OverloadPause (5min) + resume grace (5min) ≈ 10min from "before". The
	// rate-limit FallbackPause (30min) + resume grace (5min) ≈ 35min would be
	// far outside this window, so a generous [7min, 20min] band unambiguously
	// distinguishes the two without depending on exact scheduling latency.
	got := pausedUntil.Sub(before)
	if got < 7*time.Minute || got > 20*time.Minute {
		t.Errorf("paused_until - enqueue time: got %s, want within [7m,20m] (OverloadPause+grace ≈ 10m; a rate-limit FallbackPause+grace ≈ 35m would land far outside this band)", got)
	}
}
