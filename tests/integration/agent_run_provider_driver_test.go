// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// ── agent-logging-provider-driver integration tests ─────────────────────────
//
// Covers test plan Milestones 3 (API-driver end-to-end), 4 (CLI-driver
// end-to-end), and 6 (secret hygiene) for the requirement "Record Provider
// and Driver on Every Agent Run"
// (lifecycle/requirements/agent-logging-provider-driver-2.md).

// fetchRunLogText fetches the raw run log via GET .../agents/runs/{id}/log
// and returns the body as text (the endpoint serves text/plain, not JSON).
func fetchRunLogText(t *testing.T, env *testEnv, runID string) string {
	t.Helper()
	resp := env.doRequest(http.MethodGet, "/api/p/testproject/agents/runs/"+runID+"/log", nil)
	requireStatus(t, resp, http.StatusOK)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading run log body: %v", err)
	}
	return string(b)
}

// TestAgentRunProviderDriver_APIDriver_RecordsRow verifies (Milestone 3) that
// an openai-compatible run records driver id + bound provider name on the DB
// row (via the API) and writes both tokens in the log header, before any
// output line.
func TestAgentRunProviderDriver_APIDriver_RecordsRow(t *testing.T) {
	turns := []testutil.MockTurn{
		{Content: "Analysis complete.", FinishReason: "stop"},
	}
	env := newOpenAIAgentTestEnv(t, turns, 4)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "openai-analyst", "lifecycle/ideas/test-idea.md")
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("expected status=done, got %q (run=%v)", status, run)
	}
	if driver, _ := run["driver"].(string); driver != "openai-compatible" {
		t.Errorf("run.driver: got %q, want openai-compatible", driver)
	}
	if provider, _ := run["provider"].(string); provider != "test-provider" {
		t.Errorf("run.provider: got %q, want test-provider", provider)
	}

	logText := fetchRunLogText(t, env.testEnv, runID)
	if !strings.HasPrefix(logText, "# kaos-control agent run") {
		t.Fatalf("log does not start with the header block:\n%s", logText)
	}
	headerBlock := logText
	if idx := strings.Index(logText, "\n\n"); idx != -1 {
		headerBlock = logText[:idx]
	}
	if !strings.Contains(headerBlock, "driver=openai-compatible") {
		t.Errorf("log header missing driver=openai-compatible; header block:\n%s", headerBlock)
	}
	if !strings.Contains(headerBlock, "provider=test-provider") {
		t.Errorf("log header missing provider=test-provider; header block:\n%s", headerBlock)
	}
}

// TestAgentRunProviderDriver_CLIDriver_RecordsRow verifies (Milestone 4) that
// a claude-code-cli run records the driver id, an empty provider, and writes
// both the literal `driver=` and `provider=` tokens (empty, not omitted) in
// the log header.
func TestAgentRunProviderDriver_CLIDriver_RecordsRow(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/ideas/cli-driver-test.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("CLI Driver Test", "idea", "draft", "cli-driver-test", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("expected status=done, got %q (run=%v)", status, run)
	}
	if driver, _ := run["driver"].(string); driver != "claude-code-cli" {
		t.Errorf("run.driver: got %q, want claude-code-cli", driver)
	}
	if provider, _ := run["provider"].(string); provider != "" {
		t.Errorf("run.provider: got %q, want empty string", provider)
	}

	logText := fetchRunLogText(t, env, runID)
	if !strings.HasPrefix(logText, "# kaos-control agent run") {
		t.Fatalf("log does not start with the header block:\n%s", logText)
	}
	headerBlock := logText
	if idx := strings.Index(logText, "\n\n"); idx != -1 {
		headerBlock = logText[:idx]
	}
	if !strings.Contains(headerBlock, "driver=claude-code-cli") {
		t.Errorf("log header missing driver=claude-code-cli; header block:\n%s", headerBlock)
	}
	if !strings.Contains(headerBlock, "provider=") {
		t.Errorf("log header missing literal empty provider= token; header block:\n%s", headerBlock)
	}
}

// newSecretHygieneTestEnv builds an openai-compatible test env whose
// provider is configured with a recognisable fake secret token (both as the
// API key sent in Authorization: Bearer, and enforced by the mock server),
// so TestAgentRunProviderDriver_SecretHygiene can prove the token never
// lands in the driver/provider columns or the log.
func newSecretHygieneTestEnv(t *testing.T, apiKey string) *openAIAgentTestEnv {
	t.Helper()

	mock := testutil.NewMockOpenAIServer()
	mock.RequireAuthToken = apiKey
	mock.ScriptedTurns = []testutil.MockTurn{
		{Content: "Analysis complete.", FinishReason: "stop"},
	}
	t.Cleanup(func() { mock.Close() })

	appCfg := &config.App{
		Server: config.ServerConfig{Listen: "127.0.0.1:0"},
		Auth:   config.AuthConfig{Method: "local", SessionTTL: 24 * time.Hour},
		Limits: config.LimitsConfig{
			MaxConcurrentAgents:        4,
			MaxConcurrentSchedulerJobs: 2,
			SchedulerRunRetentionDays:  90,
		},
		Providers: []config.Provider{
			{
				Name:    "test-provider",
				BaseURL: mock.URL(),
				Driver:  "openai-compatible",
				APIKey:  apiKey,
			},
		},
	}

	projRoot, dataDir := setupOpenAIAgentProject(t)

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        projRoot,
		Description: "secret hygiene test project",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 4,
		Providers:           appCfg.Providers,
	})
	if err != nil {
		t.Fatalf("project.Open: %v", err)
	}
	t.Cleanup(func() { proj.Close() })

	authDBPath := filepath.Join(dataDir, "auth.db")
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })
	if err := authStore.CreateUser("admin@test.local", "Admin", "admin-pass-123", false); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	cfgDir := t.TempDir()
	appCfgPath := filepath.Join(cfgDir, "config.yaml")
	_ = config.SaveApp(appCfgPath, *appCfg)

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener:   ln,
		Auth:       authStore,
		AppCfg:     appCfg,
		AppCfgPath: appCfgPath,
	}, map[string]*project.Project{
		"testproject": proj,
	})

	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe(ctx) }()

	baseURL := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	env := &testEnv{
		t:           t,
		projectRoot: projRoot,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		proj:        proj,
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	return &openAIAgentTestEnv{testEnv: env, mock: mock}
}

// TestAgentRunProviderDriver_SecretHygiene verifies (Milestone 6, NFR-1) that
// the provider's secret API key never appears in the driver/provider columns
// nor anywhere in the run log — including the log header this change adds.
func TestAgentRunProviderDriver_SecretHygiene(t *testing.T) {
	const secretToken = "SECRET-DO-NOT-LOG"
	env := newSecretHygieneTestEnv(t, secretToken)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "openai-analyst", "lifecycle/ideas/test-idea.md")
	run := waitForRunCompletion(t, env.testEnv, runID)

	if status, _ := run["status"].(string); status != "done" {
		t.Fatalf("expected status=done, got %q (run=%v)", status, run)
	}
	if driver, _ := run["driver"].(string); strings.Contains(driver, secretToken) {
		t.Errorf("run.driver leaked the secret token: %q", driver)
	}
	if provider, _ := run["provider"].(string); strings.Contains(provider, secretToken) {
		t.Errorf("run.provider leaked the secret token: %q", provider)
	}

	logText := fetchRunLogText(t, env.testEnv, runID)
	if strings.Contains(logText, secretToken) {
		t.Errorf("run log contains the secret token:\n%s", logText)
	}
	if strings.Contains(logText, "Authorization:") {
		t.Errorf("run log contains a raw Authorization header value:\n%s", logText)
	}
}
