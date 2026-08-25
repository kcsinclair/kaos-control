// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
)

// ---------------------------------------------------------------------------
// Canonical agy NDJSON fixture (test plan "Fixture" section): one init, four
// step_update (one carries a non-empty text_delta, one does not), one result
// with status:"SUCCESS" and the recorded usage — matching the verified event
// schema in lifecycle/requirements/gemini-cli-stream-json-2.md. Every test
// below derives its input from these building blocks so the schema is
// defined in exactly one place.
// ---------------------------------------------------------------------------

const agyInitLine = `{"event":"init","conversation_id":"conv-1","init":{"cwd":"/work","tools":["read_file","write_file"]}}`

const agySuccessResultLine = `{"event":"result","result":{"conversation_id":"conv-1","status":"SUCCESS","response":"OK\n","num_turns":1,"duration_seconds":4.363532,"usage":{"input_tokens":19425,"output_tokens":25,"thinking_tokens":24,"cache_read_tokens":0,"total_tokens":19450}}}`

// agyStepUpdateLines are the fixture's four step_update lines. Index 1 carries
// the fixture's only non-empty text_delta ("Hello"); indices 0 and 3 do not.
var agyStepUpdateLines = []string{
	`{"event":"step_update","step_update":{"conversation_id":"conv-1","step_index":0,"step_type":"thought","state":"running"}}`,
	`{"event":"step_update","step_update":{"conversation_id":"conv-1","step_index":1,"step_type":"message","state":"running","text_delta":"Hello"}}`,
	`{"event":"step_update","step_update":{"conversation_id":"conv-1","step_index":1,"step_type":"message","state":"running","text_delta":" world"}}`,
	`{"event":"step_update","step_update":{"conversation_id":"conv-1","step_index":2,"step_type":"tool_call","state":"completed","duration_seconds":0.42}}`,
}

// agyFixtureLines returns the canonical stream (init + 4 step_update +
// resultLine). Pass "" for resultLine to simulate a truncated stream — a
// clean exit with no terminal result event (Milestone 4).
func agyFixtureLines(resultLine string) []string {
	lines := append([]string{agyInitLine}, agyStepUpdateLines...)
	if resultLine != "" {
		lines = append(lines, resultLine)
	}
	return lines
}

// agyFixtureLog joins agyFixtureLines into NDJSON log content, as ParseAgyResultLine expects it.
func agyFixtureLog(resultLine string) string {
	return strings.Join(agyFixtureLines(resultLine), "\n") + "\n"
}

// agyResultLine builds a `{"event":"result",...}` line with the given status
// and response text, mirroring the verified schema's result payload shape.
func agyResultLine(status, response string) string {
	return fmt.Sprintf(`{"event":"result","result":{"conversation_id":"conv-1","status":%q,"response":%q,"num_turns":2,"duration_seconds":1.5,"usage":{"input_tokens":100,"output_tokens":50,"thinking_tokens":10,"cache_read_tokens":5,"total_tokens":150}}}`, status, response)
}

// ---------------------------------------------------------------------------
// Milestone 2 (FR-3): ParseAgyResultLine happy path.
// ---------------------------------------------------------------------------

func TestParseAgyResultLine_HappyPath(t *testing.T) {
	result, err := agent.ParseAgyResultLine(agyFixtureLog(agySuccessResultLine))
	if err != nil {
		t.Fatalf("ParseAgyResultLine: %v", err)
	}
	if result.NumTurns != 1 {
		t.Errorf("NumTurns = %d, want 1", result.NumTurns)
	}
	if result.DurationMs != 4364 {
		t.Errorf("DurationMs = %d, want 4364 (round(4.363532s * 1000))", result.DurationMs)
	}
	if result.DurationApiMs != 4364 {
		t.Errorf("DurationApiMs = %d, want 4364", result.DurationApiMs)
	}
	if result.Usage.InputTokens != 19425 {
		t.Errorf("Usage.InputTokens = %d, want 19425", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 25 {
		t.Errorf("Usage.OutputTokens = %d, want 25", result.Usage.OutputTokens)
	}
	if result.Usage.CacheReadInputTokens != 0 {
		t.Errorf("Usage.CacheReadInputTokens = %d, want 0", result.Usage.CacheReadInputTokens)
	}
	if result.Usage.CacheCreationInputTokens != 0 {
		t.Errorf("Usage.CacheCreationInputTokens = %d, want 0 (no agy equivalent — FR-6, never fabricated)", result.Usage.CacheCreationInputTokens)
	}
	if result.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD = %v, want 0 (agy reports no cost — FR-6, never fabricated)", result.TotalCostUSD)
	}
	if result.Subtype != "success" {
		t.Errorf("Subtype = %q, want %q", result.Subtype, "success")
	}
	if result.IsError {
		t.Error("IsError = true, want false")
	}
	if result.Result != "OK\n" {
		t.Errorf("Result = %q, want %q", result.Result, "OK\n")
	}
}

// ---------------------------------------------------------------------------
// Milestone 3 (FR-4): success/failure mapping over result.status.
// ---------------------------------------------------------------------------

func TestParseAgyResultLine_StatusMapping(t *testing.T) {
	cases := []struct {
		name        string
		status      string
		response    string
		wantIsError bool
		wantSubtype string
	}{
		{"success", "SUCCESS", "All done.", false, "success"},
		{"error", "ERROR", "boom", true, "error"},
		{"cancelled", "CANCELLED", "", true, "cancelled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			log := agyFixtureLog(agyResultLine(tc.status, tc.response))
			result, err := agent.ParseAgyResultLine(log)
			if err != nil {
				t.Fatalf("ParseAgyResultLine: %v", err)
			}
			if result.IsError != tc.wantIsError {
				t.Errorf("IsError = %v, want %v", result.IsError, tc.wantIsError)
			}
			if result.Subtype != tc.wantSubtype {
				t.Errorf("Subtype = %q, want %q", result.Subtype, tc.wantSubtype)
			}
			if tc.wantIsError {
				if !strings.Contains(result.Result, tc.status) {
					t.Errorf("Result = %q, want it to surface status %q", result.Result, tc.status)
				}
			} else if result.Result != tc.response {
				t.Errorf("Result = %q, want response %q", result.Result, tc.response)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Milestone 4, layer 1 (FR-4): missing terminal result event.
// ---------------------------------------------------------------------------

func TestParseAgyResultLine_NoResultLine(t *testing.T) {
	log := agyFixtureLog("") // init + step_updates only, no terminal result
	result, err := agent.ParseAgyResultLine(log)
	if err == nil {
		t.Fatal("expected an error when no result line is present")
	}
	if result != nil {
		t.Errorf("expected a nil result, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// Milestone 7 (NFR-2), parser layer: malformed/plain-text lines around a
// valid result line never prevent that result from parsing.
// ---------------------------------------------------------------------------

func TestParseAgyResultLine_ToleratesMalformedAndPlainTextLines(t *testing.T) {
	fixture := agyFixtureLines(agySuccessResultLine)
	mixed := append([]string{}, fixture[:len(fixture)-1]...)
	mixed = append(mixed, `{"event": "step_update", not valid json`, "some plain stdout text", fixture[len(fixture)-1])
	log := strings.Join(mixed, "\n") + "\n"

	result, err := agent.ParseAgyResultLine(log)
	if err != nil {
		t.Fatalf("ParseAgyResultLine: %v", err)
	}
	if result.IsError {
		t.Error("expected the valid trailing result line to parse despite malformed/plain-text lines around it")
	}
}

// ---------------------------------------------------------------------------
// Driver-level tests (Milestones 5, 6, 7): drive GeminiCliDriver.Start
// directly against a shim `agy` binary, following the same shape as the
// existing internal/agent/gemini_cli_test.go re-exec tests but via a plain
// shell script (GeminiCliDriver.BinaryPath is exported for exactly this).
// ---------------------------------------------------------------------------

// writeAgyShim writes an executable script that prints each of lines to
// stdout verbatim (one per line) and exits 0. Returns the script's path.
func writeAgyShim(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	for _, l := range lines {
		sb.WriteString("printf '%s\\n' " + shellQuote(l) + "\n")
	}
	sb.WriteString("exit 0\n")
	path := filepath.Join(dir, "fake-agy")
	if err := os.WriteFile(path, []byte(sb.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// shellQuote wraps s in single quotes for embedding in a generated /bin/sh script.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Milestone 5 (FR-5): OnTTFT fires exactly once, on the first
// text_delta-bearing step_update — not on init, and not on the earlier
// delta-less step_update.
func TestGeminiCliDriver_TTFT(t *testing.T) {
	fixture := agyFixtureLines(agySuccessResultLine)
	shim := writeAgyShim(t, fixture)
	driver := &agent.GeminiCliDriver{BinaryPath: shim}

	var ttftCalls int
	var ttftMs int64
	run := agent.Run{
		RunID:       "run-ttft",
		PromptText:  "hello",
		ProjectRoot: t.TempDir(),
		LogPath:     filepath.Join(t.TempDir(), "run.log"),
		OnTTFT: func(ms int64) {
			ttftCalls++
			ttftMs = ms
		},
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if ttftCalls != 1 {
		t.Fatalf("OnTTFT called %d times, want exactly 1", ttftCalls)
	}
	if ttftMs < 0 {
		t.Errorf("OnTTFT elapsed = %dms, want >= 0", ttftMs)
	}

	// Confirm the fixture invariant this assertion relies on: an init event
	// and a delta-less step_update both precede the first text_delta, so a
	// single TTFT call could only have fired on that first delta-bearing step.
	var sawInit, sawDeltaLessStep bool
	for _, ev := range events {
		switch ev.Event["event"] {
		case "init":
			sawInit = true
		case "step_update":
			if !sawDeltaLessStep {
				if step, _ := ev.Event["step_update"].(map[string]any); step != nil && step["text_delta"] == nil {
					sawDeltaLessStep = true
				}
			}
		}
	}
	if !sawInit || !sawDeltaLessStep {
		t.Fatal("fixture invariant broken: expected an init event and a delta-less step_update before the first text_delta")
	}
}

// Milestone 6 (FR-5): proc.Progress() yields one ProgressEvent per agy line,
// step_update payloads expose text_delta, and raw NDJSON is teed to the log.
func TestGeminiCliDriver_ProgressEvents(t *testing.T) {
	fixture := agyFixtureLines(agySuccessResultLine)
	shim := writeAgyShim(t, fixture)
	driver := &agent.GeminiCliDriver{BinaryPath: shim}

	logPath := filepath.Join(t.TempDir(), "run.log")
	run := agent.Run{
		RunID:       "run-progress",
		PromptText:  "hello",
		ProjectRoot: t.TempDir(),
		LogPath:     logPath,
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	// events[0] is the synthetic "started" marker; one event per fixture line follows.
	if len(events) != len(fixture)+1 {
		t.Fatalf("got %d events, want %d (1 started + %d fixture lines)", len(events), len(fixture)+1, len(fixture))
	}
	for i, raw := range fixture {
		if ev := events[i+1]; ev.Raw != raw {
			t.Errorf("event %d Raw = %q, want %q", i+1, ev.Raw, raw)
		}
	}

	var sawTextDelta bool
	for _, ev := range events {
		if ev.Event["event"] != "step_update" {
			continue
		}
		if step, _ := ev.Event["step_update"].(map[string]any); step != nil && step["text_delta"] == "Hello" {
			sawTextDelta = true
		}
	}
	if !sawTextDelta {
		t.Error(`no step_update progress event exposed text_delta="Hello"`)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	for _, raw := range fixture {
		if !strings.Contains(string(logBytes), raw) {
			t.Errorf("log file missing raw NDJSON line: %s", raw)
		}
	}
}

// Milestone 7 (NFR-2), driver layer: malformed JSON and plain-text lines
// interleaved with valid agy events are never fatal — they degrade to raw
// "output" progress events — and a valid event after them still parses.
func TestGeminiCliDriver_MalformedAndUnknownLinesDegradeGracefully(t *testing.T) {
	lines := []string{
		agyInitLine,
		`{"event": "step_update", this is not valid json`,
		"plain text line with no JSON at all",
		agyStepUpdateLines[1], // carries a non-empty text_delta
	}
	shim := writeAgyShim(t, lines)
	driver := &agent.GeminiCliDriver{BinaryPath: shim}

	run := agent.Run{
		RunID:       "run-malformed",
		PromptText:  "hello",
		ProjectRoot: t.TempDir(),
		LogPath:     filepath.Join(t.TempDir(), "run.log"),
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("process should exit cleanly despite malformed/unknown lines: %v", err)
	}

	if len(events) != len(lines)+1 {
		t.Fatalf("got %d events, want %d", len(events), len(lines)+1)
	}
	if got := events[2].Event["type"]; got != "output" {
		t.Errorf("malformed JSON line did not degrade to type:output, got event=%v", events[2].Event)
	}
	if got := events[3].Event["type"]; got != "output" {
		t.Errorf("plain-text line did not degrade to type:output, got event=%v", events[3].Event)
	}
	if got := events[4].Event["event"]; got != "step_update" {
		t.Errorf("valid step_update after malformed lines did not parse natively, got event=%v", events[4].Event)
	}
}

// ---------------------------------------------------------------------------
// End-to-end tests (Milestone 4 layer 2, Milestone 9): drive a real
// gemini-cli run through the full HTTP + supervisor stack with a shim `agy`
// on PATH, mirroring setupFakeClaude's approach for claude-code-cli.
// GeminiCliDriver has no config-level binary-path override (see
// internal/agent/gemini_cli.go), so PATH-shadowing is the only way to
// substitute a fake binary when driven through the manager rather than
// called directly.
// ---------------------------------------------------------------------------

const geminiAgentCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: gemini-tester
    role: [backend-developer]
    driver: gemini-cli
    model: gemini-2.5-flash
    active_status: in-development
    done_on_success: true
    allowed_write_paths:
      - lifecycle/backend-plans
    git_identity:
      name: Gemini Tester Agent
      email: gemini-tester@test.local
    prompt_templates:
      backend-developer: "Test gemini-cli prompt for {target_path}"
`

// setupAgyShimOnPath writes an executable "agy" script emitting lines (one
// per stdout line) then exiting 0, and prepends its directory to PATH.
func setupAgyShimOnPath(t *testing.T, lines []string) {
	t.Helper()
	fakeDir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	for _, l := range lines {
		sb.WriteString("printf '%s\\n' " + shellQuote(l) + "\n")
	}
	sb.WriteString("exit 0\n")
	fakeScript := filepath.Join(fakeDir, "agy")
	if err := os.WriteFile(fakeScript, []byte(sb.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+":"+os.Getenv("PATH"))
}

// Milestone 9: a successful gemini-cli run streams step_update progress over
// the WebSocket hub and produces a RunResult (turns, duration, token usage,
// zero cost) via the run-result endpoint — the data the run summary card and
// Agents view render from.
func TestGeminiCliRun_EndToEnd_SuccessTelemetryAndProgress(t *testing.T) {
	setupAgyShimOnPath(t, agyFixtureLines(agySuccessResultLine))

	const targetPath = "lifecycle/backend-plans/gemini-e2e.md"
	env := newAgentTestEnvWithCfg(t, geminiAgentCfgYAML, []seedArtifact{{
		relPath: targetPath,
		content: makeArtifact("Gemini E2E Plan", "plan-backend", "planning", "gemini-e2e", "", "Plan body."),
	}})

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	runID := startAgentRun(t, env, "gemini-tester", targetPath)

	var sawStepUpdateWithDelta bool
	deadline := time.After(10 * time.Second)
collect:
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
			if evt.Type != "agent.progress" {
				continue
			}
			if gotRunID, _ := evt.Payload["run_id"].(string); gotRunID != runID {
				continue
			}
			ev, _ := evt.Payload["event"].(map[string]any)
			if ev == nil || ev["event"] != "step_update" {
				continue
			}
			if step, _ := ev["step_update"].(map[string]any); step != nil && step["text_delta"] != nil {
				sawStepUpdateWithDelta = true
				break collect
			}
		case <-deadline:
			break collect
		}
	}
	if !sawStepUpdateWithDelta {
		t.Error("never observed an agent.progress step_update event carrying text_delta on the WebSocket hub")
	}

	run := waitForRunCompletion(t, env, runID)
	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("run status = %v, want done: %+v", run["status"], run)
	}

	resp := env.doRequest("GET", "/api/p/testproject/agents/runs/"+runID+"/result", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	result, _ := data["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected a non-nil result, got %+v", data)
	}

	if got := result["num_turns"]; got != float64(1) {
		t.Errorf("num_turns = %v, want 1", got)
	}
	if got := result["duration_ms"]; got != float64(4364) {
		t.Errorf("duration_ms = %v, want 4364", got)
	}
	if got := result["is_error"]; got != false {
		t.Errorf("is_error = %v, want false", got)
	}
	if got := result["total_cost_usd"]; got != float64(0) {
		t.Errorf("total_cost_usd = %v, want 0 (agy never fabricates cost — FR-6)", got)
	}
	usage, _ := result["usage"].(map[string]any)
	if usage == nil {
		t.Fatal("expected a usage object in the result")
	}
	if got := usage["input_tokens"]; got != float64(19425) {
		t.Errorf("usage.input_tokens = %v, want 19425", got)
	}
	if got := usage["output_tokens"]; got != float64(25) {
		t.Errorf("usage.output_tokens = %v, want 25", got)
	}
	if got := usage["cache_creation_input_tokens"]; got != float64(0) {
		t.Errorf("usage.cache_creation_input_tokens = %v, want 0 (no agy equivalent — FR-3)", got)
	}
}

// Milestone 4, layer 2: a gemini-cli run that exits cleanly with no terminal
// result event is recorded failed/truncated_stream.
func TestGeminiCliRun_EndToEnd_NoResultEvent_RecordsTruncatedStream(t *testing.T) {
	setupAgyShimOnPath(t, agyFixtureLines(""))

	const targetPath = "lifecycle/backend-plans/gemini-truncated.md"
	env := newAgentTestEnvWithCfg(t, geminiAgentCfgYAML, []seedArtifact{{
		relPath: targetPath,
		content: makeArtifact("Gemini Truncated Plan", "plan-backend", "planning", "gemini-truncated", "", "Plan body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "gemini-tester", targetPath)
	run := waitForRunCompletion(t, env, runID)

	if status, _ := run["status"].(string); status != "failed" {
		t.Fatalf("status = %v, want failed: %+v", run["status"], run)
	}
	if reason, _ := run["failure_reason"].(string); reason != "truncated_stream" {
		t.Fatalf("failure_reason = %v, want truncated_stream: %+v", run["failure_reason"], run)
	}
}

// Milestone 7 (NFR-2), end-to-end: an agy that emits only plain text (e.g. it
// rejected --output-format stream-json) still completes rather than hanging
// or crashing, and — having produced no result event — is recorded
// failed/truncated_stream, the same distinguishable outcome as the no-event
// case above.
func TestGeminiCliRun_EndToEnd_PlainTextOnly_CompletesAsTruncatedStream(t *testing.T) {
	setupAgyShimOnPath(t, []string{"Just some plain text output.", "No JSON here at all."})

	const targetPath = "lifecycle/backend-plans/gemini-plaintext.md"
	env := newAgentTestEnvWithCfg(t, geminiAgentCfgYAML, []seedArtifact{{
		relPath: targetPath,
		content: makeArtifact("Gemini Plaintext Plan", "plan-backend", "planning", "gemini-plaintext", "", "Plan body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "gemini-tester", targetPath)
	run := waitForRunCompletion(t, env, runID)

	if status, _ := run["status"].(string); status != "failed" {
		t.Fatalf("status = %v, want failed: %+v", run["status"], run)
	}
	if reason, _ := run["failure_reason"].(string); reason != "truncated_stream" {
		t.Fatalf("failure_reason = %v, want truncated_stream: %+v", run["failure_reason"], run)
	}
}
