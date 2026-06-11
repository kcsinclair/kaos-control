// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/triage"
)

// captureHandler is a slog.Handler that records log records for test assertions.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func newCaptureHandler() *captureHandler { return &captureHandler{} }

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Clone the record so attrs are accessible after the call returns.
	clone := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(a)
		return true
	})
	h.records = append(h.records, clone)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	return h
}

// findRecord returns the first warn/error record whose message equals msg,
// or nil if not found.
func (h *captureHandler) findRecord(msg string) *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.records {
		if h.records[i].Message == msg {
			return &h.records[i]
		}
	}
	return nil
}

// attrValue returns the string value of the named attribute in r, or "".
func attrValue(r *slog.Record, key string) string {
	var val string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			val = a.Value.String()
			return false
		}
		return true
	})
	return val
}

// TestTriageFailure_MalformedJSON verifies that a malformed LLM response
// results in a failed run and leaves the artifact unchanged.
func TestTriageFailure_MalformedJSON(t *testing.T) {
	installLLMFake(t, []string{"not valid json at all"})

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/malformed.md",
		content: makeArtifact("Malformed", "idea", "raw", "malformed", "",
			"Raw idea for malformed json test with enough words."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	// Record original bytes.
	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/malformed.md")
	origBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading original: %v", err)
	}

	// Trigger via API.
	resp := env.doRequest("POST", triageURL("malformed"), nil)
	requireStatus(t, resp, 202)

	// Wait for failure.
	run := pollForRunStatus(t, env, "lifecycle/ideas/malformed.md", "failed", 5*time.Second)
	if run == nil {
		t.Fatal("run did not fail within 5s")
	}

	// Artifact bytes must be unchanged.
	afterBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading after: %v", err)
	}
	if !bytes.Equal(origBytes, afterBytes) {
		t.Error("artifact bytes changed after failure (should be unchanged)")
	}

	// Artifact status must still be raw.
	fm := readArtifactFM(t, env.projectRoot, "lifecycle/ideas/malformed.md")
	if status, _ := fm["status"].(string); status != "raw" {
		t.Errorf("expected artifact status 'raw' after failure, got %q", status)
	}

	// run stderr should mention parse failure.
	stderr, _ := run["stderr_tail"].(string)
	if !strings.Contains(stderr, "JSON") && !strings.Contains(stderr, "json") &&
		!strings.Contains(stderr, "parse") && !strings.Contains(stderr, "unmarshal") {
		t.Errorf("expected stderr to mention JSON parse error; got: %q", stderr)
	}
}

// TestTriageFailure_ActionClarify verifies that an LLM response with
// action=clarify results in a failed run (only propose is accepted).
func TestTriageFailure_ActionClarify(t *testing.T) {
	clarifyResp := `{"action":"clarify","reply":"What is this idea about?","slug":"","title":"","labels":[],"body":""}`
	installLLMFake(t, []string{clarifyResp})

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/clarify-test.md",
		content: makeArtifact("Clarify Test", "idea", "raw", "clarify-test", "",
			"Raw idea for clarify action test with enough words."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("clarify-test"), nil)
	requireStatus(t, resp, 202)

	run := pollForRunStatus(t, env, "lifecycle/ideas/clarify-test.md", "failed", 5*time.Second)
	if run == nil {
		t.Fatal("run did not fail within 5s for clarify action")
	}
}

// TestTriageFailure_EmptyBody verifies that an LLM response with empty body
// results in a failed run and leaves the artifact unchanged.
func TestTriageFailure_EmptyBody(t *testing.T) {
	emptyBodyResp := `{"action":"propose","reply":"ok","slug":"empty-body","title":"Empty","labels":[],"body":""}`
	installLLMFake(t, []string{emptyBodyResp})

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/empty-body.md",
		content: makeArtifact("Empty Body", "idea", "raw", "empty-body", "",
			"Raw idea for empty body test with enough words."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/empty-body.md")
	origBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading original: %v", err)
	}

	resp := env.doRequest("POST", triageURL("empty-body"), nil)
	requireStatus(t, resp, 202)

	run := pollForRunStatus(t, env, "lifecycle/ideas/empty-body.md", "failed", 5*time.Second)
	if run == nil {
		t.Fatal("run did not fail within 5s for empty body")
	}

	afterBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading after: %v", err)
	}
	if !bytes.Equal(origBytes, afterBytes) {
		t.Error("artifact bytes changed after empty-body failure")
	}
}

// TestTriageFailure_NoRetry verifies that after a failure for a path, no
// second agent_runs row is inserted automatically (no internal retry).
func TestTriageFailure_NoRetry(t *testing.T) {
	installLLMFakeError(t, "deliberate failure for no-retry test")

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/no-retry.md",
		content: makeArtifact("No Retry", "idea", "raw", "no-retry", "",
			"Raw idea for no-retry test with enough words."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("no-retry"), nil)
	requireStatus(t, resp, 202)

	// Wait for failure.
	if run := pollForRunStatus(t, env, "lifecycle/ideas/no-retry.md", "failed", 5*time.Second); run == nil {
		t.Fatal("initial run did not fail within 5s")
	}

	// Wait an additional 3s to confirm no retry.
	time.Sleep(3 * time.Second)

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath("lifecycle/ideas/no-retry.md")
	if err != nil {
		t.Fatalf("ListAgentRunsByTargetPath: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected exactly 1 run (no retry), got %d", len(runs))
	}
}

// TestTriageFailure_LogLineContents verifies that a triage failure emits a
// slog warn line containing the required structured fields.
func TestTriageFailure_LogLineContents(t *testing.T) {
	handler := newCaptureHandler()
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(orig) })

	installLLMFakeError(t, "test-log-failure")

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/log-test.md",
		content: makeArtifact("Log Test", "idea", "raw", "log-test", "",
			"Raw idea for log line test with enough words."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("log-test"), nil)
	requireStatus(t, resp, 202)

	// Wait for failure.
	if run := pollForRunStatus(t, env, "lifecycle/ideas/log-test.md", "failed", 5*time.Second); run == nil {
		t.Fatal("run did not fail within 5s")
	}
	time.Sleep(100 * time.Millisecond) // let log flush

	// Find the warn record.
	rec := handler.findRecord("triage failed")
	if rec == nil {
		t.Fatal("expected slog warn 'triage failed' not found")
	}

	path := attrValue(rec, "path")
	if path != "lifecycle/ideas/log-test.md" {
		t.Errorf("log field 'path': want 'lifecycle/ideas/log-test.md', got %q", path)
	}
	lineage := attrValue(rec, "lineage")
	if lineage != "log-test" {
		t.Errorf("log field 'lineage': want 'log-test', got %q", lineage)
	}
	reason := attrValue(rec, "reason")
	if reason == "" {
		t.Error("log field 'reason' missing")
	}
	_ = attrValue(rec, "run_id") // present but not asserted on exact value
}

// TestTriageFailure_SandboxViolation_UnitCheck verifies that path traversal is
// rejected at the eligibility layer (not_in_ideas_dir) before reaching the
// sandbox. The public Trigger API is the first line of defence.
func TestTriageFailure_SandboxViolation_UnitCheck(t *testing.T) {
	// A traversal path: path.Dir returns "lifecycle/ideas/../requirements"
	// which != "lifecycle/ideas", so eligibility rejects it immediately.
	traversalPath := "lifecycle/ideas/../requirements/foo.md"

	env := newTriageTestEnv(t)

	_, err := env.proj.TriageMgr.Trigger(context.Background(), traversalPath, triage.TriggerAPI)
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}

	var ie triage.ErrIneligible
	if !errors.As(err, &ie) {
		t.Fatalf("expected ErrIneligible, got %T: %v", err, err)
	}
	if ie.Reason != "not_in_ideas_dir" {
		t.Errorf("expected reason 'not_in_ideas_dir', got %q", ie.Reason)
	}
}
