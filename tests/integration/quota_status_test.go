// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/rate-limit-event-detection-5-test.md
// Requirement: lifecycle/requirements/rate-limit-event-detection-2.md
//
// Milestones 1-3, 5, 6 (Mode 1 — observability): a fake `claude` binary emits
// mid-stream `rate_limit_event` stream-json lines; the supervisor's broadcast
// closure (internal/agent/agent.go:801) parses them via extractRateLimitInfo
// and re-broadcasts a normalised `agent.quota_status` project-hub event with
// content-change debounce. These tests drive that behaviour end-to-end through
// the real HTTP + hub stack (no white-box access to extractRateLimitInfo /
// Manager.runQuota — those are internal/agent-package concerns).
//
// Coverage note: AC6 (the runQuota cache entry is removed by cleanupRunState)
// has no externally observable signal once a run is terminal — asserting it
// directly requires a package-internal test alongside the existing
// runPolicies/deniedCalls cleanup tests in internal/agent, which is out of
// scope for this integration suite. See the companion artifact at
// lifecycle/tests/rate-limit-event-detection-6-test.md.

import (
	"encoding/json"
	"testing"
	"time"
)

// rateLimitEventLine marshals a `rate_limit_event` stream-json line from raw
// rate_limit_info fields. Fields omitted from the map are absent from the
// JSON, so callers can exercise extractRateLimitInfo's defensive-parse paths
// (AC3) by simply leaving keys out.
func rateLimitEventLine(t *testing.T, fields map[string]any) string {
	t.Helper()
	ev := map[string]any{
		"type":            "rate_limit_event",
		"rate_limit_info": fields,
		"session_id":      "test-session",
		"uuid":            "test-uuid",
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// expectedQuotaPayload independently re-derives the normalised
// agent.quota_status payload from raw rate_limit_info fields, mirroring the
// FR1/FR3 rules (bucket/status normalisation, best-effort overage, RFC3339-UTC
// resets_at omitted when zero). It is a deliberately independent
// reimplementation — not a call into internal/agent — so a regression in the
// real normaliser is caught rather than mirrored.
func expectedQuotaPayload(fields map[string]any) map[string]any {
	bucket, _ := fields["rateLimitType"].(string)
	if bucket != "five_hour" && bucket != "weekly" {
		bucket = "unknown"
	}
	status, _ := fields["status"].(string)
	if status != "allowed" && status != "warning" && status != "rejected" {
		status = "unknown"
	}
	isUsingOverage, _ := fields["isUsingOverage"].(bool)
	overageStatus, _ := fields["overageStatus"].(string)
	overageDisabledReason, _ := fields["overageDisabledReason"].(string)

	want := map[string]any{
		"bucket":                  bucket,
		"status":                  status,
		"overage_available":       isUsingOverage || overageStatus != "rejected",
		"overage_disabled_reason": overageDisabledReason,
	}
	if raw, ok := fields["resetsAt"]; ok {
		var sec int64
		switch v := raw.(type) {
		case int:
			sec = int64(v)
		case int64:
			sec = v
		}
		if sec != 0 {
			want["resets_at"] = time.Unix(sec, 0).UTC().Format(time.RFC3339)
		}
	}
	return want
}

// assertQuotaPayload checks got against want, plus the run_id, plus that
// resets_at is present iff want says it should be (NFR2 — omitted, not
// null/empty, when ResetsAtUnix is 0).
func assertQuotaPayload(t *testing.T, got map[string]any, runID string, want map[string]any) {
	t.Helper()
	if rid, _ := got["run_id"].(string); rid != runID {
		t.Errorf("run_id: got %q, want %q", rid, runID)
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Errorf("%s: missing from payload (want %v)", k, wv)
			continue
		}
		if gv != wv {
			t.Errorf("%s: got %v, want %v", k, gv, wv)
		}
	}
	if _, wantResetsAt := want["resets_at"]; !wantResetsAt {
		if v, present := got["resets_at"]; present {
			t.Errorf("resets_at: got %v, want key omitted", v)
		}
	}
}

// collectQuotaEvents drains ch for agent.quota_status events scoped to runID
// until the run's terminal event (agent.finished/agent.failed) arrives, then
// returns everything collected. Since the broadcast closure emits
// agent.quota_status synchronously for each stream line, all quota events for
// a run are on the hub strictly before its terminal event.
func collectQuotaEvents(t *testing.T, ch chan []byte, runID string) []map[string]any {
	t.Helper()
	var quota []map[string]any
	timeout := time.After(10 * time.Second)
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			rid, _ := evt.Payload["run_id"].(string)
			if rid != runID {
				continue
			}
			switch evt.Type {
			case "agent.quota_status":
				quota = append(quota, evt.Payload)
			case "agent.finished", "agent.failed":
				return quota
			}
		case <-timeout:
			t.Fatalf("timed out waiting for terminal event on run %s", runID)
			return quota
		}
	}
}

// runAgentAndCollectQuota starts a fake claude emitting scriptLines, runs the
// requirements-analyst agent against a fresh idea artifact, and returns the
// run_id plus every agent.quota_status payload observed before the run's
// terminal event.
func runAgentAndCollectQuota(t *testing.T, scriptLines []string, exitCode int) (runID string, quota []map[string]any) {
	t.Helper()
	setupFakeClaudeWithLines(t, scriptLines, exitCode)

	const artifactPath = "lifecycle/ideas/quota-status-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Quota Status Test", "idea", "draft", "quota-status-test", "", "Body."),
	}})

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID = startAgentRun(t, env, "requirements-analyst", artifactPath)
	quota = collectQuotaEvents(t, ch, runID)
	return runID, quota
}

// TestQuotaStatus_AC1_ExactSamplePayload replays the requirement's exact
// sample rate_limit_event (lifecycle/requirements/rate-limit-event-detection-2.md,
// Background section) and asserts the resulting agent.quota_status matches
// AC1's expected fields exactly, broadcast exactly once (AC4).
func TestQuotaStatus_AC1_ExactSamplePayload(t *testing.T) {
	const sampleLine = `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour","resetsAt":1778911200,"isUsingOverage":false,"overageStatus":"rejected","overageDisabledReason":"out_of_credits"},"session_id":"sess-ac1","uuid":"11111111-1111-1111-1111-111111111111"}`

	runID, quota := runAgentAndCollectQuota(t, []string{sampleLine}, 0)

	if len(quota) != 1 {
		t.Fatalf("agent.quota_status count: got %d, want 1 (payloads: %v)", len(quota), quota)
	}
	got := quota[0]
	if bucket, _ := got["bucket"].(string); bucket != "five_hour" {
		t.Errorf("bucket: got %q, want five_hour", bucket)
	}
	if status, _ := got["status"].(string); status != "allowed" {
		t.Errorf("status: got %q, want allowed", status)
	}
	if resetsAt, _ := got["resets_at"].(string); resetsAt != "2026-05-16T06:00:00Z" {
		t.Errorf("resets_at: got %q, want 2026-05-16T06:00:00Z", resetsAt)
	}
	if overageAvailable, _ := got["overage_available"].(bool); overageAvailable != false {
		t.Errorf("overage_available: got %v, want false", overageAvailable)
	}
	if reason, _ := got["overage_disabled_reason"].(string); reason != "out_of_credits" {
		t.Errorf("overage_disabled_reason: got %q, want out_of_credits", reason)
	}
	if rid, _ := got["run_id"].(string); rid != runID {
		t.Errorf("run_id: got %q, want %q", rid, runID)
	}
}

// TestQuotaStatus_AC1_NonRateLimitEventPayload_NoBroadcast verifies a normal
// (non-rate_limit_event) stream event, and a rate_limit_event missing its
// rate_limit_info object, both produce zero agent.quota_status broadcasts —
// but do not prevent a later, well-formed rate_limit_event from broadcasting
// (proving the malformed lines are skipped, not fatal).
func TestQuotaStatus_AC1_NonRateLimitEventPayload_NoBroadcast(t *testing.T) {
	assistantLine := `{"type":"assistant","message":{"content":[{"type":"text","text":"working on it"}]}}`
	missingInfoLine := `{"type":"rate_limit_event","session_id":"s","uuid":"u"}`
	validLine := rateLimitEventLine(t, map[string]any{
		"status":        "allowed",
		"rateLimitType": "five_hour",
		"resetsAt":      1778911200,
	})

	_, quota := runAgentAndCollectQuota(t, []string{assistantLine, missingInfoLine, validLine}, 0)

	if len(quota) != 1 {
		t.Fatalf("agent.quota_status count: got %d, want 1 (only the well-formed line should broadcast; payloads: %v)", len(quota), quota)
	}
	if bucket, _ := quota[0]["bucket"].(string); bucket != "five_hour" {
		t.Errorf("bucket: got %q, want five_hour", bucket)
	}
}

// TestQuotaStatus_AC2_BucketDiscrimination covers FR1/AC2: rateLimitType
// "weekly" maps to Bucket "weekly"; an unrecognised value maps to "unknown".
func TestQuotaStatus_AC2_BucketDiscrimination(t *testing.T) {
	cases := []struct {
		name          string
		rateLimitType string
		wantBucket    string
	}{
		{"weekly", "weekly", "weekly"},
		{"unrecognised", "lunar", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{
				"status":        "allowed",
				"rateLimitType": tc.rateLimitType,
				"resetsAt":      1778911200,
			}
			line := rateLimitEventLine(t, fields)
			runID, quota := runAgentAndCollectQuota(t, []string{line}, 0)
			if len(quota) != 1 {
				t.Fatalf("agent.quota_status count: got %d, want 1", len(quota))
			}
			assertQuotaPayload(t, quota[0], runID, expectedQuotaPayload(fields))
		})
	}
}

// TestQuotaStatus_AC3_DefensiveParse_MissingFields covers AC3/NFR4: a
// rate_limit_event missing resetsAt/status/overageStatus/isUsingOverage parses
// without panic (the run completes normally and still broadcasts), with
// ResetsAtUnix=0 (resets_at key omitted) and Status defaulting to "unknown".
func TestQuotaStatus_AC3_DefensiveParse_MissingFields(t *testing.T) {
	fields := map[string]any{
		"rateLimitType": "five_hour",
		// status, resetsAt, isUsingOverage, overageStatus, overageDisabledReason
		// are all deliberately absent.
	}
	line := rateLimitEventLine(t, fields)
	runID, quota := runAgentAndCollectQuota(t, []string{line}, 0)
	if len(quota) != 1 {
		t.Fatalf("agent.quota_status count: got %d, want 1", len(quota))
	}
	assertQuotaPayload(t, quota[0], runID, expectedQuotaPayload(fields))
}

// TestQuotaStatus_NFR4_UnknownValuesSurfaceAsUnknown covers NFR4: a novel
// status ("throttled") and a novel overageStatus surface as best-effort
// "unknown"/computed values without dropping the event (ok=true still
// broadcasts).
func TestQuotaStatus_NFR4_UnknownValuesSurfaceAsUnknown(t *testing.T) {
	fields := map[string]any{
		"status":        "throttled",
		"rateLimitType": "five_hour",
		"resetsAt":      1778911200,
		"overageStatus": "future_overage_state",
	}
	line := rateLimitEventLine(t, fields)
	runID, quota := runAgentAndCollectQuota(t, []string{line}, 0)
	if len(quota) != 1 {
		t.Fatalf("agent.quota_status count: got %d, want 1 (unknown values must not drop the event)", len(quota))
	}
	assertQuotaPayload(t, quota[0], runID, expectedQuotaPayload(fields))
}

// TestQuotaStatus_OverageAvailableTruthTable covers the FR1 OverageAvailable
// truth table: isUsingOverage:true always wins; otherwise OverageAvailable is
// true unless overageStatus is exactly "rejected".
func TestQuotaStatus_OverageAvailableTruthTable(t *testing.T) {
	cases := []struct {
		name           string
		isUsingOverage bool
		overageStatus  string
		want           bool
	}{
		{"using_overage_true", true, "rejected", true},
		{"not_using_overage_rejected", false, "rejected", false},
		{"not_using_overage_available", false, "available", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{
				"status":         "allowed",
				"rateLimitType":  "five_hour",
				"resetsAt":       1778911200,
				"isUsingOverage": tc.isUsingOverage,
				"overageStatus":  tc.overageStatus,
			}
			line := rateLimitEventLine(t, fields)
			runID, quota := runAgentAndCollectQuota(t, []string{line}, 0)
			if len(quota) != 1 {
				t.Fatalf("agent.quota_status count: got %d, want 1", len(quota))
			}
			assertQuotaPayload(t, quota[0], runID, expectedQuotaPayload(fields))
			if got, _ := quota[0]["overage_available"].(bool); got != tc.want {
				t.Errorf("overage_available: got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestQuotaStatus_AC5_ContentChangeDebounce covers AC5/FR4: two identical
// consecutive rate_limit_events broadcast once; a third event differing in
// exactly one debounce-tuple field broadcasts a second time. Each subtest
// exercises a different field of the tuple (bucket, status, resets_at,
// overage_available, overage_disabled_reason).
func TestQuotaStatus_AC5_ContentChangeDebounce(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{
			"status":                "allowed",
			"rateLimitType":         "five_hour",
			"resetsAt":              1778911200,
			"isUsingOverage":        false,
			"overageStatus":         "rejected",
			"overageDisabledReason": "out_of_credits",
		}
	}

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"bucket", func(f map[string]any) { f["rateLimitType"] = "weekly" }},
		{"status", func(f map[string]any) { f["status"] = "warning" }},
		{"resets_at", func(f map[string]any) { f["resetsAt"] = 1778911200 + 3600 }},
		{"overage_available", func(f map[string]any) { f["isUsingOverage"] = true }},
		{"overage_disabled_reason", func(f map[string]any) { f["overageDisabledReason"] = "different_reason" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseFields := base()
			variantFields := base()
			tc.mutate(variantFields)

			lines := []string{
				rateLimitEventLine(t, baseFields),
				rateLimitEventLine(t, baseFields), // identical consecutive — suppressed
				rateLimitEventLine(t, variantFields),
			}
			runID, quota := runAgentAndCollectQuota(t, lines, 0)
			if len(quota) != 2 {
				t.Fatalf("agent.quota_status count: got %d, want 2 (identical pair suppressed, variant broadcast; payloads: %v)", len(quota), quota)
			}
			assertQuotaPayload(t, quota[0], runID, expectedQuotaPayload(baseFields))
			assertQuotaPayload(t, quota[1], runID, expectedQuotaPayload(variantFields))
		})
	}
}

// TestQuotaStatus_AC8_NoRateLimitEvent_ZeroBroadcasts covers the Mode-1 half
// of AC8: a run whose stream contains no rate_limit_event emits zero
// agent.quota_status events.
func TestQuotaStatus_AC8_NoRateLimitEvent_ZeroBroadcasts(t *testing.T) {
	const artifactPath = "lifecycle/ideas/quota-status-degradation.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Quota Status Degradation Test", "idea", "draft", "quota-status-degradation", "", "Body."),
	}})
	setupFakeClaude(t, 0) // fakeClaudeSuccessEvents: init + result, no rate_limit_event

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	quota := collectQuotaEvents(t, ch, runID)

	if len(quota) != 0 {
		t.Errorf("agent.quota_status count: got %d, want 0 for a run with no rate_limit_event (payloads: %v)", len(quota), quota)
	}
}

// TestQuotaStatus_NFR3_OllamaDriver_NoQuotaEvents covers NFR3: the
// rate_limit_event shape is Claude-Code-specific. A non-Claude driver
// (Ollama) run never produces a rate_limit_event-shaped stream event, so it
// must emit zero agent.quota_status events.
func TestQuotaStatus_NFR3_OllamaDriver_NoQuotaEvents(t *testing.T) {
	const artifactPath = "lifecycle/ideas/quota-status-ollama.md"
	env := newOllamaAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Quota Status Ollama Test", "idea", "draft", "quota-status-ollama", "", "Body."),
	}}, 2)

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env.testEnv, "ollama-analyst", artifactPath)
	quota := collectQuotaEvents(t, ch, runID)

	if len(quota) != 0 {
		t.Errorf("agent.quota_status count: got %d, want 0 for an Ollama-driven run (payloads: %v)", len(quota), quota)
	}
}

// TestQuotaStatus_NFR1_ExistingEventShapesUnchanged covers NFR1: the presence
// of agent.quota_status events does not alter the shape of an ordinary
// agent.progress event delivered in the same stream.
func TestQuotaStatus_NFR1_ExistingEventShapesUnchanged(t *testing.T) {
	assistantLine := `{"type":"assistant","message":{"content":[{"type":"text","text":"still working"}]}}`
	quotaLine := rateLimitEventLine(t, map[string]any{
		"status":        "allowed",
		"rateLimitType": "five_hour",
		"resetsAt":      1778911200,
	})
	setupFakeClaudeWithLines(t, []string{assistantLine, quotaLine}, 0)

	const artifactPath = "lifecycle/ideas/quota-status-nfr1.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Quota Status NFR1 Test", "idea", "draft", "quota-status-nfr1", "", "Body."),
	}})

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)

	var progressPayload map[string]any
	var quotaSeen bool
	timeout := time.After(10 * time.Second)
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			rid, _ := evt.Payload["run_id"].(string)
			if rid != runID {
				continue
			}
			switch evt.Type {
			case "agent.progress":
				if progressPayload == nil {
					progressPayload = evt.Payload
				}
			case "agent.quota_status":
				quotaSeen = true
			case "agent.finished", "agent.failed":
				break COLLECT
			}
		case <-timeout:
			t.Fatal("timed out waiting for terminal event")
		}
	}

	if !quotaSeen {
		t.Fatal("expected at least one agent.quota_status event")
	}
	if progressPayload == nil {
		t.Fatal("expected at least one agent.progress event")
	}
	// Unchanged shape: run_id/line/raw/event keys, same as before this feature.
	for _, key := range []string{"run_id", "line", "raw", "event"} {
		if _, ok := progressPayload[key]; !ok {
			t.Errorf("agent.progress payload missing pre-existing key %q: %v", key, progressPayload)
		}
	}
}
