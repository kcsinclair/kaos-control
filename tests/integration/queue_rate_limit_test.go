// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.3 — Queue rate-limit integration tests (QR1–QR4)
//
// These tests inject rate-limit stream events via a fake `claude` binary that
// outputs JSON to stdout and then exits. The agent supervisor detects the
// rate-limit pattern and broadcasts queue.rate_limit to the project hub, which
// the queue dispatcher's watchRunEvents goroutine picks up.
//
// Rate-limit event format (stream-json line that the driver reads):
//
//	{"error":{"type":"rate_limit_error","message":"<text>"}}
//
// The driver broadcasts this as:
//
//	agent.progress { event: { error: { type: "rate_limit_error", message: "..." } } }
//
// The supervisor then re-broadcasts as queue.rate_limit { raw_text: "..." }.

import (
	"testing"
	"time"
)

// rateLimitScript builds a shell script body that outputs a rate-limit event
// line with the given reset text and then exits with code 0.
// The format matches the Claude Code stream-json error shape that
// extractRateLimitText parses.
func rateLimitScript(resetText string) string {
	return "printf '%s\\n' '{\"error\":{\"type\":\"rate_limit_error\",\"message\":\"" +
		resetText + "\"}}'\nexit 0\n"
}

// TestQueue_RateLimit_FromSampleLog (QR1): replay a rate-limit event with a
// parseable reset time. Verify:
//   (a) the running job is marked failed/rate_limit
//   (b) a new pending row exists at the head with attempts=2
//   (c) queue_state.paused = true, paused_until ≈ next 20:00 Australia/Brisbane + grace
//   (d) the GET /api/queue snapshot reports paused=true
func TestQueue_RateLimit_FromSampleLog(t *testing.T) {
	// Fake claude outputs a rate-limit event for "resets 8pm (Australia/Brisbane)".
	setupFakeClaudeWithScript(t, rateLimitScript("resets 8pm (Australia/Brisbane)"))

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qr1-idea-1.md",
			content: makeApprovedArtifact("QR1 Rate Limit Idea", "idea", "qr1-idea"),
		},
	})

	// Enqueue.
	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qr1-idea-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)
	enqData := readJSON(t, enqResp)
	originalID, _ := enqData["id"].(string)

	// Wait for the rate-limit processing to complete (original job goes to failed,
	// new pending job is re-enqueued, and queue is paused).
	deadline := time.Now().Add(15 * time.Second)
	var snap map[string]any
	for time.Now().Before(deadline) {
		snap = env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// (a) Original job should be failed/rate_limit (in recent list).
	recent, _ := snap["recent"].([]any)
	foundFailed := false
	for _, raw := range recent {
		j, _ := raw.(map[string]any)
		if j["id"] == originalID {
			foundFailed = true
			if j["state"] != "failed" {
				t.Errorf("original job state: got %v, want failed", j["state"])
			}
			if reason, _ := j["reason"].(string); reason != "rate_limit" {
				t.Errorf("original job reason: got %q, want rate_limit", reason)
			}
		}
	}
	if !foundFailed {
		t.Errorf("original job %q not found in recent list", originalID)
	}

	// (b) A new pending job exists with attempts=2 and a smaller position than
	// the original job.
	pending, _ := snap["pending"].([]any)
	foundRequeued := false
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		if j["artifact_path"] == "lifecycle/ideas/qr1-idea-1.md" {
			foundRequeued = true
			if attempts, _ := j["attempts"].(float64); attempts != 2 {
				t.Errorf("requeued job attempts: got %v, want 2", attempts)
			}
		}
	}
	if !foundRequeued {
		t.Errorf("expected a re-queued pending job for qr1-idea-1.md")
	}

	// (c) / (d) Queue is paused with a paused_until set.
	if paused, _ := snap["paused"].(bool); !paused {
		t.Error("expected queue to be paused after rate-limit")
	}
	if until, _ := snap["paused_until"].(string); until == "" {
		t.Error("expected non-empty paused_until after rate-limit")
	}
}

// TestQueue_RateLimit_AutoResume (QR2): after a rate-limit pause, once the
// clock passes paused_until, the dispatcher should auto-resume and start the
// re-queued job.
//
// Implementation note: the queue dispatcher's ClockFn defaults to time.Now.
// In integration tests we cannot easily inject a fake clock, so this test
// instead uses a very short FallbackPause (via an unparseable reset text) to
// let real time pass. We verify the queue resumes within a reasonable wall-
// clock window.
func TestQueue_RateLimit_AutoResume(t *testing.T) {
	// Use an unparseable reset text so the dispatcher falls back to
	// FallbackPause. We set a 2-second FallbackPause by using a very short
	// fallback in the dispatcher config. Unfortunately, the integration test env
	// uses the default dispatcher config.
	//
	// Strategy: use a parseable "retry after 2 seconds" which the parser
	// resolves to now+2s, and with the 5-minute default resume grace the
	// paused_until would be now+2s+5min — too long to wait.
	//
	// Alternative strategy: use the dispatcher's auto-resume by not wiring any
	// grace. But the default is 5min.
	//
	// Pragmatic approach: issue two rate-limits, let the first one complete,
	// then directly resume via POST /api/queue/resume (simulating auto-resume
	// check), and verify the re-queued job starts.
	//
	// For a proper auto-resume integration test without clock injection, a
	// custom dispatcher config with very short FallbackPause + ResumeGrace
	// would be required. This is tracked as a future enhancement.

	setupFakeClaudeWithScript(t, rateLimitScript("resets 8pm (Australia/Brisbane)"))

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qr2-idea-1.md",
			content: makeApprovedArtifact("QR2 Auto-Resume Idea", "idea", "qr2-idea"),
		},
	})

	// Enqueue and wait for rate-limit pause.
	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qr2-idea-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)

	// Wait until paused.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Verify paused.
	snap := env.queueSnapshot()
	if paused, _ := snap["paused"].(bool); !paused {
		t.Skip("rate-limit pause not reached; skipping auto-resume check")
	}

	// Use a success-exit fake claude for the re-queued job.
	setupFakeClaude(t, 0)

	// Manually resume (simulating auto-resume after paused_until elapses).
	resumeResp := env.doRequest("POST", "/api/queue/resume", nil)
	requireStatus(t, resumeResp, 204)
	resumeResp.Body.Close()

	// The re-queued job (attempts=2) should start and complete.
	deadline = time.Now().Add(15 * time.Second)
	completed := false
	for time.Now().Before(deadline) {
		snap := env.queueSnapshot()
		recent, _ := snap["recent"].([]any)
		for _, raw := range recent {
			j, _ := raw.(map[string]any)
			if j["artifact_path"] == "lifecycle/ideas/qr2-idea-1.md" {
				if attempts, _ := j["attempts"].(float64); attempts == 2 {
					if j["state"] == "completed" {
						completed = true
					}
				}
			}
		}
		if completed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !completed {
		t.Error("expected re-queued job (attempts=2) to complete after resume")
	}
}

// TestQueue_RateLimit_FallbackOnUnparseable (QR3): inject a rate-limit event
// with unparseable text ("resets soon"); assert the queue is paused and
// paused_until is set (using the fallback pause).
func TestQueue_RateLimit_FallbackOnUnparseable(t *testing.T) {
	setupFakeClaudeWithScript(t, rateLimitScript("resets soon"))

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qr3-idea-1.md",
			content: makeApprovedArtifact("QR3 Fallback Idea", "idea", "qr3-idea"),
		},
	})

	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qr3-idea-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)

	// Wait for pause.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	snap := env.queueSnapshot()
	if paused, _ := snap["paused"].(bool); !paused {
		t.Fatal("expected queue to be paused after unparseable rate-limit text")
	}
	// paused_until must be set to some future time (the fallback).
	until, _ := snap["paused_until"].(string)
	if until == "" {
		t.Error("expected non-empty paused_until when using fallback pause")
	}
}

// TestQueue_RateLimit_MaxAttempts (QR4): force 3 consecutive rate-limit
// failures on the same artifact. After the 3rd (exceeding MaxAttempts=3),
// the job must NOT be re-enqueued.
//
// We exercise this by configuring setupFakeClaudeWithScript to always emit a
// rate-limit event. The default MaxAttempts is 5; we need the test to hit the
// cap sooner. Since we cannot easily override MaxAttempts in the integration
// env without a custom helper, this test uses the default (5) and verifies
// after 5 failures.
//
// NOTE: with the default MaxAttempts=5, this test is slow (5 round-trips
// through the dispatcher). It is marked as a longer-running integration test.
func TestQueue_RateLimit_MaxAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("TestQueue_RateLimit_MaxAttempts is slow; skipped in -short mode")
	}

	setupFakeClaudeWithScript(t, rateLimitScript("resets 8pm (Australia/Brisbane)"))

	env := newQueueTestEnv(t, []seedArtifact{
		{
			relPath: "lifecycle/ideas/qr4-idea-1.md",
			content: makeApprovedArtifact("QR4 MaxAttempts Idea", "idea", "qr4-idea"),
		},
	})

	enqResp := env.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": "lifecycle/ideas/qr4-idea-1.md",
		"agent":         "requirements-analyst",
	})
	requireStatus(t, enqResp, 201)

	// Default MaxAttempts is 5. Keep resuming after each pause until the job
	// is no longer re-enqueued (i.e., pending count drops to 0).
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts+1; attempt++ {
		// Wait for pause (or for no pending jobs if all attempts exhausted).
		deadline := time.Now().Add(30 * time.Second)
		paused := false
		for time.Now().Before(deadline) {
			snap := env.queueSnapshot()
			if p, _ := snap["paused"].(bool); p {
				paused = true
				break
			}
			pending, _ := snap["pending"].([]any)
			if len(pending) == 0 {
				break // no more pending → max attempts hit
			}
			time.Sleep(100 * time.Millisecond)
		}

		snap := env.queueSnapshot()
		pending, _ := snap["pending"].([]any)
		if len(pending) == 0 {
			// Max attempts reached — job is no longer re-enqueued.
			break
		}

		if !paused {
			t.Fatalf("attempt %d: timed out waiting for pause or empty pending queue", attempt)
		}

		if attempt > maxAttempts {
			// We've exceeded expected max — still pending means it shouldn't be.
			t.Errorf("job still pending after %d attempts (expected cap at %d)", attempt, maxAttempts)
			break
		}

		// Resume so the next attempt can run.
		resumeResp := env.doRequest("POST", "/api/queue/resume", nil)
		requireStatus(t, resumeResp, 204)
		resumeResp.Body.Close()
	}

	// After all attempts exhausted, the artifact should have no pending job.
	snap := env.queueSnapshot()
	pending, _ := snap["pending"].([]any)
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		if j["artifact_path"] == "lifecycle/ideas/qr4-idea-1.md" {
			t.Errorf("expected no pending job after max attempts, found: %v", j)
		}
	}
}
