// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.4 — Secrets hygiene audit integration test (FS1).
//
// Conforms to lifecycle/architecture/standards/secrets-handling.md: agent
// auth tokens / provider API keys must never leak into lifecycle/ markdown,
// git commit history, WebSocket event payloads, or API responses.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

const (
	anthropicSecretKey = "sk-ant-TESTSECRET-do-not-leak-12345"
	geminiSecretKey    = "sk-gem-TESTSECRET-do-not-leak-67890"
)

// TestSecrets_FailoverAudit (FS1): drives a full automated-failover cycle
// (operations.yaml mutation, WS broadcast, REST responses) using providers
// whose app-level api_key is set, then asserts none of those keys appear
// anywhere in: lifecycle/config.yaml on disk, operations.yaml on disk (the
// new runtime-state store — Milestone 1's NFR-1), the git commit log,
// buffered WebSocket event payloads, or the queue/status REST response
// bodies.
func TestSecrets_FailoverAudit(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible", APIKey: anthropicSecretKey},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible", APIKey: geminiSecretKey},
	}

	markerPath := filepath.Join(t.TempDir(), "fs1-invoked")
	errorJSON := `{"error":{"type":"overloaded_error","message":"HTTP 529 Overloaded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	env := newFailoverTestEnv(t, failoverAutoCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/fs1-idea-1.md", content: makeApprovedArtifact("FS1 Idea", "idea", "fs1-idea")},
	})
	beforeSHA := env.headSHA()

	ws := env.connectProjectWS()

	env.enqueue("lifecycle/ideas/fs1-idea-1.md", "requirements-analyst")

	env.waitFor(15*time.Second, "requeued job to complete after failover", func() bool {
		snap := env.queueSnapshot()
		j := findJobByPath(snap, "lifecycle/ideas/fs1-idea-1.md")
		if j == nil {
			return false
		}
		attempts, _ := j["attempts"].(float64)
		return j["state"] == "completed" && attempts == 2
	})
	// Also exercise the read paths that could plausibly leak a key.
	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusBody, err := readBodyString(statusResp)
	if err != nil {
		t.Fatal(err)
	}
	reportResp := env.doRequest("GET", "/api/p/testproject/reports/failover", nil)
	requireStatus(t, reportResp, 200)
	reportBody, err := readBodyString(reportResp)
	if err != nil {
		t.Fatal(err)
	}

	// 1. lifecycle/config.yaml on disk — never touched by a switch/failover.
	assertNoSecret(t, "lifecycle/config.yaml", env.readConfigYAML())

	// 2. operations.yaml on disk — the new runtime-state store that actually
	// records the switch; it carries provider/model names only (NFR-1).
	assertNoSecret(t, "operations.yaml", env.readOperationsYAML())

	// 3. git commit log — unaffected either way, since neither file is ever
	// committed by a switch/failover.
	assertNoSecret(t, "git commit log", strings.Join(env.gitLogMessages(20), "\n"))

	// 4. Buffered WebSocket event payloads.
	drained := drainEvents(ws)
	wsJSON, err := json.Marshal(drained)
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, "WS event payloads", string(wsJSON))

	// 5. REST response bodies.
	assertNoSecret(t, "GET provider-switch/status response", statusBody)
	assertNoSecret(t, "GET reports/failover response", reportBody)

	// Milestone 1's git-isolation guarantee, exercised end-to-end here since
	// this is the one test that actually drives a real failover: the switch
	// landed in operations.yaml only — no commit, and (given the project's
	// .gitignore covers it, the documented convention) it never shows up as
	// a tracked/untracked change either.
	if got := env.headSHA(); got != beforeSHA {
		t.Errorf("expected no git commit from the failover, HEAD moved from %s to %s", beforeSHA, got)
	}
	for _, f := range env.modifiedFiles() {
		if f == "operations.yaml" {
			t.Error("operations.yaml must never appear in git status")
		}
	}
}

// assertNoSecret fails the test if either test API key literal appears in
// haystack, identified by the given source label.
func assertNoSecret(t *testing.T, source, haystack string) {
	t.Helper()
	if strings.Contains(haystack, anthropicSecretKey) {
		t.Errorf("%s leaked the anthropic-cloud API key", source)
	}
	if strings.Contains(haystack, geminiSecretKey) {
		t.Errorf("%s leaked the gemini-cloud API key", source)
	}
}

// drainEvents empties the projectWSClient's buffered channel into a slice.
func drainEvents(c *projectWSClient) []map[string]any {
	var out []map[string]any
	for {
		select {
		case msg := <-c.events:
			out = append(out, msg)
		default:
			return out
		}
	}
}
