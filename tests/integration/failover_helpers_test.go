// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/queue"
)

// failoverTestEnv wraps testEnv with the full production wiring for
// automated provider failover: a queue dispatcher whose ProjectAccess
// exposes FailoverPolicy/AgentFailoverInfo/ProbeProviderHealth/
// SwitchAgentProvider (mirroring cmd/kaos-control/main.go), plus the
// app-level provider catalog used for health probes and REST validation.
type failoverTestEnv struct {
	*testEnv
	ctx        context.Context
	queueStore *queue.Store
	dispatcher *queue.Dispatcher
	appHub     *hub.Hub
	appCfg     *config.App
}

// mockProvider is a toggleable fake OpenAI-compatible upstream: GET
// /v1/models returns 200 while healthy, 503 while not. Used both as an
// app-level provider target for ProbeProviderHealth and as a stand-in for
// "the fallback backend is reachable".
type mockProvider struct {
	*httptest.Server
	healthy atomic.Bool
}

func newMockProvider(t *testing.T, healthy bool) *mockProvider {
	t.Helper()
	m := &mockProvider{}
	m.healthy.Store(healthy)
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	t.Cleanup(m.Server.Close)
	return m
}

func (m *mockProvider) setHealthy(v bool) { m.healthy.Store(v) }

// newFailoverTestEnv creates a fully wired test environment (project + HTTP
// server + queue dispatcher with automated failover support) whose
// lifecycle/config.yaml is cfgYAML and whose app-level provider catalog is
// providers.
func newFailoverTestEnv(t *testing.T, cfgYAML string, providers []config.Provider, seeds []seedArtifact) *failoverTestEnv {
	t.Helper()

	root := t.TempDir()
	dataDir := t.TempDir()

	for _, s := range []string{
		"ideas", "requirements", "backend-plans", "frontend-plans",
		"test-plans", "tests", "prototypes", "releases", "sprints", "defects",
	} {
		if err := os.MkdirAll(filepath.Join(root, "lifecycle", s), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	gcfg, _ := repo.Config()
	gcfg.User.Name = "Test User"
	gcfg.User.Email = "test@test.local"
	if err := repo.SetConfig(gcfg); err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("lifecycle/config.yaml"); err != nil {
		t.Fatal(err)
	}

	for _, s := range seeds {
		absPath := filepath.Join(root, s.relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(s.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(s.relPath); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@test.local", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}
	if ref, err := repo.Head(); err == nil && ref.Name().Short() != "main" {
		_ = repo.CreateBranch(&gitconfig.Branch{Name: "main", Remote: ""})
	}

	appCfg := &config.App{
		Server:    config.ServerConfig{Listen: "127.0.0.1:0"},
		Auth:      config.AuthConfig{Method: "local", SessionTTL: 24 * time.Hour},
		Providers: providers,
	}
	appCfgPath := filepath.Join(dataDir, "app-config.yaml")
	if err := config.SaveApp(appCfgPath, *appCfg); err != nil {
		t.Fatal(err)
	}

	authStore, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })
	for _, u := range []struct{ email, name, pass string }{
		{"admin@test.local", "Admin", "admin-pass-123"},
		{"dev@test.local", "Developer", "dev-pass-123"},
		{"qa@test.local", "QA Engineer", "qa-pass-123"},
	} {
		_ = authStore.CreateUser(u.email, u.name, u.pass, false)
	}

	entry := &config.ProjectEntry{Name: "testproject", Path: root, Description: "failover integration test project"}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 4,
		DevopsLogDir:        dataDir,
		Providers:           providers,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	queueStore, err := queue.Open(filepath.Join(dataDir, "queue.db"))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = queueStore.Close() })
	if err := queueStore.RecoverOrphans(); err != nil {
		t.Logf("queue: orphan recovery: %v", err)
	}

	appHub := hub.New()
	projectLookup := func(name string) (queue.ProjectAccess, bool) {
		if name != "testproject" || proj.Agents == nil {
			return queue.ProjectAccess{}, false
		}
		return queue.ProjectAccess{
			StartRun: func(runCtx context.Context, agentName, targetPath string) (string, error) {
				return proj.Agents.StartRun(runCtx, agentName, targetPath, "", nil)
			},
			ArtifactStatus: func(relPath string) string {
				row, err := proj.Idx.Get(relPath)
				if err != nil || row == nil {
					return ""
				}
				return row.Status
			},
			Hub: proj.Hub,
			FailoverPolicy: func() queue.FailoverPolicy {
				eff := proj.Config().EffectiveFailoverConfig()
				return queue.FailoverPolicy{
					Enabled:            eff.Enabled != nil && *eff.Enabled,
					AutoSwitch:         eff.AutoSwitch != nil && *eff.AutoSwitch,
					SwitchOnKinds:      eff.SwitchOnKinds,
					MaxFailoversPerRun: eff.MaxFailoversPerRun,
				}
			},
			AgentFailoverInfo: func(agentName string) (queue.AgentFailoverInfo, bool) {
				ag, ok := proj.Agents.GetAgent(agentName)
				if !ok || ag.FallbackProvider == "" {
					return queue.AgentFailoverInfo{}, false
				}
				return queue.AgentFailoverInfo{FallbackProvider: ag.FallbackProvider, FallbackModel: ag.FallbackModel}, true
			},
			ProbeProviderHealth: func(probeCtx context.Context, providerName string) bool {
				for i := range appCfg.Providers {
					if appCfg.Providers[i].Name == providerName {
						return agent.ProbeProviderHealth(probeCtx, nil, appCfg.Providers[i], 2*time.Second)
					}
				}
				return false
			},
			SwitchAgentProvider: func(agentName, providerName, model, reason string, isFailover bool) error {
				return proj.SwitchAgentProvider(agentName, providerName, model, reason, isFailover)
			},
		}, true
	}

	dispatcher := queue.New(queueStore, projectLookup, appHub, queue.Config{TickInterval: 50 * time.Millisecond})
	dispatcher.Start(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener:   ln,
		Auth:       authStore,
		Queue:      dispatcher,
		AppHub:     appHub,
		AppCfg:     appCfg,
		AppCfgPath: appCfgPath,
	}, map[string]*project.Project{"testproject": proj})

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
		projectRoot: root,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		proj:        proj,
	}
	env.login("admin@test.local", "admin-pass-123")

	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	return &failoverTestEnv{
		testEnv:    env,
		ctx:        ctx,
		queueStore: queueStore,
		dispatcher: dispatcher,
		appHub:     appHub,
		appCfg:     appCfg,
	}
}

// enqueue enqueues an agent run for artifactPath via POST /api/queue and
// returns the parsed response.
func (e *failoverTestEnv) enqueue(artifactPath, agentName string) map[string]any {
	e.t.Helper()
	resp := e.doRequest("POST", "/api/queue", map[string]any{
		"project":       "testproject",
		"artifact_path": artifactPath,
		"agent":         agentName,
	})
	requireStatus(e.t, resp, 201)
	return readJSON(e.t, resp)
}

// queueSnapshot calls GET /api/queue and returns the parsed JSON map.
func (e *failoverTestEnv) queueSnapshot() map[string]any {
	e.t.Helper()
	resp := e.doRequest("GET", "/api/queue", nil)
	requireStatus(e.t, resp, 200)
	return readJSON(e.t, resp)
}

// waitFor polls cond (via the queue snapshot when useful, but here a generic
// predicate) until it returns true or the timeout elapses, failing the test
// on timeout.
func (e *failoverTestEnv) waitFor(timeout time.Duration, what string, cond func() bool) {
	e.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Fatalf("timed out waiting for: %s", what)
}

// findJobByPath scans a queue snapshot's running/pending/recent lists for a
// job matching artifactPath, returning its record or nil.
func findJobByPath(snap map[string]any, artifactPath string) map[string]any {
	if running, _ := snap["running"].(map[string]any); running != nil {
		if running["artifact_path"] == artifactPath {
			return running
		}
	}
	for _, key := range []string{"pending", "recent"} {
		items, _ := snap[key].([]any)
		for _, raw := range items {
			j, _ := raw.(map[string]any)
			if j["artifact_path"] == artifactPath {
				return j
			}
		}
	}
	return nil
}

// readAgentConfigYAML reads the raw lifecycle/config.yaml off disk.
func (e *failoverTestEnv) readConfigYAML() string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.projectRoot, "lifecycle", "config.yaml"))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

// loadConfig parses the on-disk lifecycle/config.yaml into a config.Project,
// for assertions that need structured agent fields rather than raw text
// matching (which is fragile against YAML formatting/comment changes).
func (e *failoverTestEnv) loadConfig() *config.Project {
	e.t.Helper()
	cfg, err := config.LoadProject(e.projectRoot)
	if err != nil {
		e.t.Fatal(err)
	}
	return cfg
}

// findFailoverAgentConfig returns the named agent's config from cfg, or ok=false.
func findFailoverAgentConfig(cfg *config.Project, name string) (config.AgentConfig, bool) {
	for _, a := range cfg.Agents {
		if a.Name == name {
			return a, true
		}
	}
	return config.AgentConfig{}, false
}

// gitLogMessages returns the last n commit messages on the project repo's
// current HEAD, most recent first.
func (e *failoverTestEnv) gitLogMessages(n int) []string {
	e.t.Helper()
	commits, err := e.proj.Git.Log("lifecycle/config.yaml", n)
	if err != nil {
		e.t.Fatal(err)
	}
	msgs := make([]string, 0, len(commits))
	for _, c := range commits {
		msgs = append(msgs, c.Message)
	}
	return msgs
}

// projectWSClient is a buffered WebSocket subscriber on the project hub
// (/api/p/testproject/ws), used to assert that specific event types
// (provider.switched, provider.restored, provider.primary_recovered,
// config.reloaded, ...) were broadcast during a test.
type projectWSClient struct {
	events chan map[string]any
}

// connectProjectWS dials the project WebSocket endpoint and starts buffering
// incoming events. Call before triggering the action under test so no event
// is missed.
func (e *failoverTestEnv) connectProjectWS() *projectWSClient {
	e.t.Helper()
	wsURL := "ws://" + strings.TrimPrefix(e.baseURL, "http://") + "/api/p/testproject/ws"
	cookieHeader := buildCookieHeader(e.cookies)

	dialCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Cookie": []string{cookieHeader}},
	})
	if err != nil {
		e.t.Fatalf("websocket dial failed: %v", err)
	}
	e.t.Cleanup(func() { conn.CloseNow() })

	c := &projectWSClient{events: make(chan map[string]any, 64)}
	go func() {
		ctx := context.Background()
		for {
			var msg map[string]any
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				return
			}
			select {
			case c.events <- msg:
			default:
			}
		}
	}()
	// Small delay so the subscription is registered before the caller
	// triggers the action under test.
	time.Sleep(50 * time.Millisecond)
	return c
}

// waitForEventType drains buffered events until one with the given "type"
// field arrives, or the timeout elapses (in which case it fails the test).
// Returns the matching event's payload.
func (c *projectWSClient) waitForEventType(t *testing.T, timeout time.Duration, eventType string) map[string]any {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.events:
			if msg["type"] == eventType {
				payload, _ := msg["payload"].(map[string]any)
				return payload
			}
		case <-deadline:
			t.Fatalf("timed out waiting for WS event type %q", eventType)
			return nil
		}
	}
}
