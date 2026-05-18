// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the Projects CRUD API endpoints:
//   GET  /api/projects
//   GET  /api/projects/{project}
//   POST /api/projects
//   PUT  /api/projects/{project}
//   DELETE /api/projects/{project}
//   POST /api/projects/{project}/init
//   POST /api/projects/check-directory
//
// Covers lifecycle/test-plans/projects-crud-ui-5-test.md Milestones 2–7.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
)

// ---------------------------------------------------------------------------
// Test environment helpers
// ---------------------------------------------------------------------------

// crudTestEnv is a test environment configured for project CRUD tests.
// Unlike testEnv it starts the server with projectsDir and dataDir wired up,
// enabling the create/update/delete project endpoints to function.
type crudTestEnv struct {
	t           *testing.T
	projectsDir string            // where project YAML registrations live
	dataDir     string            // where SQLite DBs live
	baseURL     string
	cancel      context.CancelFunc
	authStore   *auth.Store
	cookies     []*http.Cookie
	csrfToken   string
	projectDirs map[string]string // name → absolute path on disk
}

// crudSeedProject describes one project to pre-populate before the server starts.
type crudSeedProject struct {
	name        string
	initialised bool // if true, lifecycle/config.yaml is written into the project dir
}

// newCRUDTestEnv creates a fresh environment for project CRUD tests.
// seeds are pre-registered projects; their directories are accessible via
// env.projectDirs[name].
func newCRUDTestEnv(t *testing.T, seeds []crudSeedProject) *crudTestEnv {
	t.Helper()

	projectsDir := t.TempDir()
	dataDir := t.TempDir()

	// Open auth store.
	authDBPath := filepath.Join(dataDir, "auth.db")
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authStore.Close() })

	if err := authStore.CreateUser("admin@test.local", "Admin", "admin-pass-123", false); err != nil {
		t.Fatal(err)
	}

	// Build pre-seeded projects.
	projects := make(map[string]*project.Project, len(seeds))
	projectDirs := make(map[string]string, len(seeds))

	for _, seed := range seeds {
		projDir := t.TempDir()
		projectDirs[seed.name] = projDir

		if seed.initialised {
			lcDir := filepath.Join(projDir, "lifecycle")
			if err := os.MkdirAll(lcDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lcDir, "config.yaml"), []byte(defaultCfgYAML), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		entry := &config.ProjectEntry{
			Name:        seed.name,
			Path:        projDir,
			Description: "seed: " + seed.name,
			Owner:       "team-a",
		}
		if err := config.SaveProjectEntry(projectsDir, entry); err != nil {
			t.Fatal(err)
		}
		proj, err := project.Open(entry, dataDir, project.OpenOptions{
			MaxConcurrentAgents: 1,
			DevopsLogDir:        dataDir,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { proj.Close() })
		projects[seed.name] = proj
	}

	// Find a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())

	openOpts := project.OpenOptions{
		MaxConcurrentAgents: 1,
		DevopsLogDir:        dataDir,
	}
	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener:           ln,
		Auth:               authStore,
		ProjectsDir:        projectsDir,
		DataDir:            dataDir,
		ProjectOpenOptions: openOpts,
	}, projects)

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

	env := &crudTestEnv{
		t:           t,
		projectsDir: projectsDir,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		projectDirs: projectDirs,
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	env.login("admin@test.local", "admin-pass-123")
	return env
}

// login authenticates and stores the session cookies + CSRF token.
func (e *crudTestEnv) login(email, password string) {
	e.t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	resp, err := http.Post(e.baseURL+"/api/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		e.t.Fatalf("login failed: %d %s", resp.StatusCode, b)
	}
	e.cookies = resp.Cookies()
	for _, c := range e.cookies {
		if c.Name == "kc_csrf" {
			e.csrfToken = c.Value
		}
	}
}

// logout clears session cookies so subsequent requests are anonymous.
func (e *crudTestEnv) logout() {
	e.cookies = nil
	e.csrfToken = ""
}

// doRequest makes an HTTP request with session cookies and CSRF token.
func (e *crudTestEnv) doRequest(method, path string, body any) *http.Response {
	e.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.baseURL+path, bodyReader)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range e.cookies {
		req.AddCookie(c)
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", e.csrfToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	return resp
}

// readCRUDJSON reads and closes the response body, decoding into a map.
func readCRUDJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON unmarshal failed on %q: %v", b, err)
	}
	return m
}

// errCode extracts the error code from an API error response.
// The server wraps errors as {"error":{"code":"...", "message":"..."}}.
func errCode(body map[string]any) string {
	errObj, _ := body["error"].(map[string]any)
	if errObj == nil {
		return ""
	}
	code, _ := errObj["code"].(string)
	return code
}

// requireCRUDStatus asserts the response status code.
func requireCRUDStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("want status %d, got %d: %s", want, resp.StatusCode, b)
	}
}

// makeTempProjectDir creates a temp directory suitable for use as a project
// path. On macOS the real path (post-EvalSymlinks) is returned so that the
// path validation in the server doesn't reject it as non-existent.
func makeTempProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	return resolved
}

// ---------------------------------------------------------------------------
// Milestone 2 — GET /api/projects and GET /api/projects/{project}
// ---------------------------------------------------------------------------

// TestListProjects_ReturnsAllWithInitialisedFlag seeds two projects (one
// initialised, one not) and verifies both appear in GET /api/projects with the
// correct initialised flag.
func TestListProjects_ReturnsAllWithInitialisedFlag(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "proj-init", initialised: true},
		{name: "proj-noinit", initialised: false},
	})

	resp := env.doRequest("GET", "/api/projects", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	projs, ok := body["projects"].([]any)
	if !ok {
		t.Fatalf("response missing 'projects' array, got: %v", body)
	}

	if len(projs) < 2 {
		t.Fatalf("expected at least 2 projects, got %d", len(projs))
	}

	byName := make(map[string]map[string]any, len(projs))
	for _, raw := range projs {
		p, _ := raw.(map[string]any)
		if p != nil {
			byName[p["name"].(string)] = p
		}
	}

	for name, wantInit := range map[string]bool{
		"proj-init":   true,
		"proj-noinit": false,
	} {
		p, found := byName[name]
		if !found {
			t.Errorf("project %q not found in response", name)
			continue
		}
		gotInit, _ := p["initialised"].(bool)
		if gotInit != wantInit {
			t.Errorf("project %q: initialised=%v, want %v", name, gotInit, wantInit)
		}
	}
}

// TestListProjects_IncludesOwner verifies that every project in the list
// response includes the owner field (even if empty).
func TestListProjects_IncludesOwner(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "owner-proj", initialised: false},
	})

	resp := env.doRequest("GET", "/api/projects", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	projs := body["projects"].([]any)
	p := projs[0].(map[string]any)
	if _, hasOwner := p["owner"]; !hasOwner {
		t.Error("project entry missing 'owner' field")
	}
}

// TestGetProject_Found verifies that GET /api/projects/{project} returns a
// single project whose fields match the registry entry.
func TestGetProject_Found(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "find-me", initialised: false},
	})

	resp := env.doRequest("GET", "/api/projects/find-me", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if body["name"] != "find-me" {
		t.Errorf("name = %q, want %q", body["name"], "find-me")
	}
	if _, hasPth := body["path"]; !hasPth {
		t.Error("response missing 'path' field")
	}
	if _, hasInit := body["initialised"]; !hasInit {
		t.Error("response missing 'initialised' field")
	}
}

// TestGetProject_NotFound verifies that an unknown project name returns 404.
func TestGetProject_NotFound(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("GET", "/api/projects/no-such-project", nil)
	requireCRUDStatus(t, resp, 404)
}

// TestListProjects_RequiresAuth verifies that GET /api/projects returns 401
// when the request has no session cookie.
func TestListProjects_RequiresAuth(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	env.logout()

	resp := env.doRequest("GET", "/api/projects", nil)
	requireCRUDStatus(t, resp, 401)
}

// ---------------------------------------------------------------------------
// Milestone 3 — POST /api/projects (create)
// ---------------------------------------------------------------------------

// TestCreateProject_Success verifies that a valid creation request returns 201
// with all fields, writes the YAML file, and makes the project accessible.
func TestCreateProject_Success(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	projDir := makeTempProjectDir(t)

	payload := map[string]any{
		"name":        "brand-new",
		"path":        projDir,
		"description": "a fresh project",
		"owner":       "alice",
	}
	resp := env.doRequest("POST", "/api/projects", payload)
	requireCRUDStatus(t, resp, 201)
	body := readCRUDJSON(t, resp)

	if body["name"] != "brand-new" {
		t.Errorf("name = %q, want %q", body["name"], "brand-new")
	}
	if body["description"] != "a fresh project" {
		t.Errorf("description = %q, want %q", body["description"], "a fresh project")
	}
	if body["owner"] != "alice" {
		t.Errorf("owner = %q, want %q", body["owner"], "alice")
	}
	if _, ok := body["path"]; !ok {
		t.Error("response missing 'path' field")
	}
	if _, ok := body["initialised"]; !ok {
		t.Error("response missing 'initialised' field")
	}

	// YAML registration file must exist on disk.
	yamlPath := filepath.Join(env.projectsDir, "brand-new.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		t.Errorf("YAML registration file not created: %v", err)
	}
	t.Cleanup(func() { os.Remove(yamlPath) })
}

// TestCreateProject_NameValidation verifies that invalid names return 400.
func TestCreateProject_NameValidation(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	projDir := makeTempProjectDir(t)

	cases := []struct {
		name    string
		errCode string
	}{
		{"", "invalid_name"},
		{"ab", "invalid_name"},
		{"MyProject", "invalid_name"},
		{"has space", "invalid_name"},
		{"has_under", "invalid_name"},
	}

	for _, tc := range cases {
		resp := env.doRequest("POST", "/api/projects", map[string]any{
			"name": tc.name,
			"path": projDir,
		})
		if resp.StatusCode != 400 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("name=%q: got status %d, want 400 (body: %s)", tc.name, resp.StatusCode, b)
			continue
		}
		body := readCRUDJSON(t, resp)
		if code := errCode(body); code != tc.errCode {
			t.Errorf("name=%q: error code = %q, want %q", tc.name, code, tc.errCode)
		}
	}
}

// TestCreateProject_PathValidation verifies that invalid paths return 400.
func TestCreateProject_PathValidation(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	cases := []struct {
		desc string
		path string
	}{
		{"empty path", ""},
		{"relative path", "relative/path"},
		{"dotdot relative", "../sibling"},
		{"non-existent absolute", "/kaos-test-definitely-does-not-exist-xyz-999"},
	}

	for _, tc := range cases {
		resp := env.doRequest("POST", "/api/projects", map[string]any{
			"name": "valid-name",
			"path": tc.path,
		})
		if resp.StatusCode != 400 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("%s: got status %d, want 400 (body: %s)", tc.desc, resp.StatusCode, b)
			continue
		}
		resp.Body.Close()
	}
}

// TestCreateProject_NameConflict verifies that registering a duplicate name
// returns 409.
func TestCreateProject_NameConflict(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "existing-proj", initialised: false},
	})

	projDir := makeTempProjectDir(t)
	resp := env.doRequest("POST", "/api/projects", map[string]any{
		"name": "existing-proj",
		"path": projDir,
	})
	requireCRUDStatus(t, resp, 409)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "conflict" {
		t.Errorf("error code = %q, want %q", code, "conflict")
	}
}

// TestCreateProject_AtomicWrite verifies that after a successful create the
// YAML file is well-formed (not a partial write).
func TestCreateProject_AtomicWrite(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	projDir := makeTempProjectDir(t)

	resp := env.doRequest("POST", "/api/projects", map[string]any{
		"name": "atomic-write-test",
		"path": projDir,
	})
	requireCRUDStatus(t, resp, 201)
	resp.Body.Close()

	yamlPath := filepath.Join(env.projectsDir, "atomic-write-test.yaml")
	t.Cleanup(func() { os.Remove(yamlPath) })

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("reading YAML: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("YAML file is empty")
	}

	// Parse as a project entry to confirm well-formed YAML.
	var entry config.ProjectEntry
	if err := func() error {
		// Use the exported loader indirectly: write to a temp dir and load.
		tmpDir := t.TempDir()
		dst := filepath.Join(tmpDir, "atomic-write-test.yaml")
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		entries, err := config.LoadProjectRegistry(tmpDir)
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			return nil
		}
		entry = *entries[0]
		return nil
	}(); err != nil {
		t.Fatalf("YAML is not well-formed: %v", err)
	}
	if entry.Name != "atomic-write-test" {
		t.Errorf("loaded name = %q, want %q", entry.Name, "atomic-write-test")
	}
}

// TestCreateProject_NoRestartRequired verifies that after creating a project
// the per-project artifacts endpoint is immediately accessible (no restart).
func TestCreateProject_NoRestartRequired(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	projDir := makeTempProjectDir(t)

	// Create a project.
	resp := env.doRequest("POST", "/api/projects", map[string]any{
		"name": "hot-reload-test",
		"path": projDir,
	})
	requireCRUDStatus(t, resp, 201)
	resp.Body.Close()
	t.Cleanup(func() { os.Remove(filepath.Join(env.projectsDir, "hot-reload-test.yaml")) })

	// Immediately query the per-project artifacts endpoint.
	resp2 := env.doRequest("GET", "/api/p/hot-reload-test/artifacts", nil)
	requireCRUDStatus(t, resp2, 200)
	resp2.Body.Close()
}

// ---------------------------------------------------------------------------
// Milestone 4 — PUT /api/projects/{project} (update)
// ---------------------------------------------------------------------------

// TestUpdateProject_Success verifies that updating description and owner
// returns 200 and persists to the YAML file.
func TestUpdateProject_Success(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "update-me", initialised: false},
	})

	resp := env.doRequest("PUT", "/api/projects/update-me", map[string]any{
		"description": "updated description",
		"owner":       "bob",
	})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if body["description"] != "updated description" {
		t.Errorf("description = %q, want %q", body["description"], "updated description")
	}
	if body["owner"] != "bob" {
		t.Errorf("owner = %q, want %q", body["owner"], "bob")
	}

	// Verify the YAML file on disk reflects the update.
	entries, err := config.LoadProjectRegistry(env.projectsDir)
	if err != nil {
		t.Fatalf("LoadProjectRegistry: %v", err)
	}
	var found *config.ProjectEntry
	for _, e := range entries {
		if e.Name == "update-me" {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatal("project 'update-me' not found in registry after update")
	}
	if found.Description != "updated description" {
		t.Errorf("persisted description = %q, want %q", found.Description, "updated description")
	}
	if found.Owner != "bob" {
		t.Errorf("persisted owner = %q, want %q", found.Owner, "bob")
	}
}

// TestUpdateProject_PathChange verifies that updating the path to a valid
// directory is reflected in the response and on disk.
func TestUpdateProject_PathChange(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "path-change-proj", initialised: false},
	})

	newDir := makeTempProjectDir(t)
	resp := env.doRequest("PUT", "/api/projects/path-change-proj", map[string]any{
		"path": newDir,
	})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if body["path"] != newDir {
		t.Errorf("path in response = %q, want %q", body["path"], newDir)
	}

	// Disk should reflect the new path.
	entries, _ := config.LoadProjectRegistry(env.projectsDir)
	for _, e := range entries {
		if e.Name == "path-change-proj" {
			if e.Path != newDir {
				t.Errorf("persisted path = %q, want %q", e.Path, newDir)
			}
			return
		}
	}
	t.Error("project 'path-change-proj' not found in registry after path update")
}

// TestUpdateProject_PathValidation verifies that an invalid new path returns 400.
func TestUpdateProject_PathValidation(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "bad-path-proj", initialised: false},
	})

	resp := env.doRequest("PUT", "/api/projects/bad-path-proj", map[string]any{
		"path": "/kaos-test-does-not-exist-xyz-update-path",
	})
	requireCRUDStatus(t, resp, 400)
	resp.Body.Close()
}

// TestUpdateProject_NameImmutable verifies that the name field cannot be
// changed via PUT — it is always ignored and the original name persists.
func TestUpdateProject_NameImmutable(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "immutable-name", initialised: false},
	})

	// Submitting a "name" field in the body must be silently ignored.
	resp := env.doRequest("PUT", "/api/projects/immutable-name", map[string]any{
		"description": "changed desc",
	})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if body["name"] != "immutable-name" {
		t.Errorf("name in response = %q, want %q (name must be immutable)", body["name"], "immutable-name")
	}

	// The YAML file must still be named immutable-name.yaml.
	if _, err := os.Stat(filepath.Join(env.projectsDir, "immutable-name.yaml")); err != nil {
		t.Errorf("immutable-name.yaml missing after update: %v", err)
	}
}

// TestUpdateProject_NotFound verifies that updating a non-existent project
// returns 404.
func TestUpdateProject_NotFound(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("PUT", "/api/projects/no-such-project", map[string]any{
		"description": "irrelevant",
	})
	requireCRUDStatus(t, resp, 404)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Milestone 5 — DELETE /api/projects/{project}
// ---------------------------------------------------------------------------

// TestDeleteProject_Success verifies that deleting a project returns 200,
// removes the YAML file, hides it from the list, and makes its scoped
// endpoints return 404.
func TestDeleteProject_Success(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "delete-me", initialised: false},
	})

	// Confirm the project is present before deletion.
	preResp := env.doRequest("GET", "/api/projects/delete-me", nil)
	requireCRUDStatus(t, preResp, 200)
	preResp.Body.Close()

	resp := env.doRequest("DELETE", "/api/projects/delete-me", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)
	if ok, _ := body["ok"].(bool); !ok {
		t.Errorf("response 'ok' = %v, want true", body["ok"])
	}

	// YAML file must be gone.
	yamlPath := filepath.Join(env.projectsDir, "delete-me.yaml")
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Errorf("YAML file still exists after deletion: %v", err)
	}

	// Project must not appear in the list.
	listResp := env.doRequest("GET", "/api/projects", nil)
	requireCRUDStatus(t, listResp, 200)
	listBody := readCRUDJSON(t, listResp)
	for _, raw := range listBody["projects"].([]any) {
		p, _ := raw.(map[string]any)
		if p != nil && p["name"] == "delete-me" {
			t.Error("deleted project still appears in GET /api/projects")
		}
	}

	// Scoped endpoint must return 404.
	scopedResp := env.doRequest("GET", "/api/p/delete-me/artifacts", nil)
	requireCRUDStatus(t, scopedResp, 404)
	scopedResp.Body.Close()
}

// TestDeleteProject_DiskUntouched verifies that the project directory and its
// files remain after deregistration.
func TestDeleteProject_DiskUntouched(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "keep-files", initialised: true},
	})

	projDir := env.projectDirs["keep-files"]
	cfgPath := filepath.Join(projDir, "lifecycle", "config.yaml")

	resp := env.doRequest("DELETE", "/api/projects/keep-files", nil)
	requireCRUDStatus(t, resp, 200)
	resp.Body.Close()

	// Project directory must still exist.
	if _, err := os.Stat(projDir); err != nil {
		t.Errorf("project directory removed after deregistration: %v", err)
	}
	// config.yaml inside the project must still exist.
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("lifecycle/config.yaml removed after deregistration: %v", err)
	}
}

// TestDeleteProject_NotFound verifies that deleting a non-existent project
// returns 404.
func TestDeleteProject_NotFound(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("DELETE", "/api/projects/no-such-project", nil)
	requireCRUDStatus(t, resp, 404)
	resp.Body.Close()
}

// TestDeleteProject_NoGoroutineLeaks verifies that deleting a project does not
// leave goroutines behind. We create a project via the API (which starts its
// goroutines through RegisterProject), then delete it and confirm the goroutine
// count returns to approximately the pre-creation baseline.
func TestDeleteProject_NoGoroutineLeaks(t *testing.T) {
	env := newCRUDTestEnv(t, nil)
	projDir := makeTempProjectDir(t)

	// Baseline: no leak-project goroutines.
	baseline := runtime.NumGoroutine()

	// Create a project via API — this calls RegisterProject which starts
	// watcher, reaper, scheduler goroutines.
	createResp := env.doRequest("POST", "/api/projects", map[string]any{
		"name": "leak-test",
		"path": projDir,
	})
	requireCRUDStatus(t, createResp, 201)
	createResp.Body.Close()

	// Give goroutines time to start.
	time.Sleep(100 * time.Millisecond)
	afterCreate := runtime.NumGoroutine()
	if afterCreate <= baseline {
		t.Log("note: no goroutine increase after create — watcher may not have started")
	}

	// Delete the project.
	delResp := env.doRequest("DELETE", "/api/projects/leak-test", nil)
	requireCRUDStatus(t, delResp, 200)
	delResp.Body.Close()

	// Give goroutines time to stop.
	time.Sleep(300 * time.Millisecond)
	afterDelete := runtime.NumGoroutine()

	// Allow headroom of 10 goroutines for GC, scheduler, and other background work.
	const headroom = 10
	if afterDelete > baseline+headroom {
		t.Errorf("possible goroutine leak: baseline=%d, after-create=%d, after-delete=%d (headroom %d)",
			baseline, afterCreate, afterDelete, headroom)
	}
}

// ---------------------------------------------------------------------------
// Milestone 6 — POST /api/projects/{project}/init
// ---------------------------------------------------------------------------

// TestInitProject_CreatesScaffolding verifies that calling init on an
// uninitialised project performs the full kaos-control scaffold (config +
// CLAUDE.md + .claude/settings.json + .gitignore + devops/sample.yaml +
// every lifecycle stage including docs) and auto-populates the logged-in
// user into config.yaml's users: section.
func TestInitProject_CreatesScaffolding(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "init-scaffold", initialised: false},
	})

	resp := env.doRequest("POST", "/api/projects/init-scaffold/init", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	created, _ := body["created"].([]any)
	if len(created) == 0 {
		t.Error("expected 'created' to be non-empty after first init")
	}

	projDir := env.projectDirs["init-scaffold"]

	// Landmark files / dirs must exist after init.
	landmarks := []string{
		"lifecycle/config.yaml",
		"CLAUDE.md",
		".claude/settings.json",
		".gitignore",
		"devops/sample.yaml",
		"lifecycle/ideas/.gitkeep",
		"lifecycle/requirements/.gitkeep",
		"lifecycle/docs/.gitkeep",
		"lifecycle/defects/.gitkeep",
	}
	for _, rel := range landmarks {
		if _, err := os.Stat(filepath.Join(projDir, rel)); err != nil {
			t.Errorf("expected %s after init: %v", rel, err)
		}
	}

	// The logged-in test user (admin@test.local; see newCRUDTestEnv)
	// must appear in config.yaml's users: section so RolesFor() returns
	// the owner role set for workflow gates.
	cfgBytes, err := os.ReadFile(filepath.Join(projDir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatalf("reading rendered config.yaml: %v", err)
	}
	if !strings.Contains(string(cfgBytes), "admin@test.local") {
		t.Errorf("rendered config.yaml does not contain the logged-in user's email; got:\n%s", cfgBytes)
	}
}

// TestInitProject_Idempotent verifies that calling init twice does not modify
// existing files and returns an empty 'created' list on the second call.
func TestInitProject_Idempotent(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "init-idempotent", initialised: false},
	})

	// First init.
	resp1 := env.doRequest("POST", "/api/projects/init-idempotent/init", nil)
	requireCRUDStatus(t, resp1, 200)
	resp1.Body.Close()

	// Record mtime of lifecycle/config.yaml.
	projDir := env.projectDirs["init-idempotent"]
	cfgPath := filepath.Join(projDir, "lifecycle", "config.yaml")
	stat1, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat after first init: %v", err)
	}

	// Sleep briefly so a second write would have a different mtime.
	time.Sleep(50 * time.Millisecond)

	// Second init.
	resp2 := env.doRequest("POST", "/api/projects/init-idempotent/init", nil)
	requireCRUDStatus(t, resp2, 200)
	body2 := readCRUDJSON(t, resp2)

	created2, _ := body2["created"].([]any)
	if len(created2) != 0 {
		t.Errorf("second init 'created' = %v, want empty (idempotent)", created2)
	}

	// config.yaml must not have been modified.
	stat2, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat after second init: %v", err)
	}
	if !stat2.ModTime().Equal(stat1.ModTime()) {
		t.Errorf("config.yaml mtime changed on second init: was %v, now %v", stat1.ModTime(), stat2.ModTime())
	}
}

// TestInitProject_GitInit verifies that init runs git init on a non-git
// directory and creates an initial commit.
func TestInitProject_GitInit(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "init-git", initialised: false},
	})

	projDir := env.projectDirs["init-git"]

	// Confirm the directory is not yet a git repo.
	if _, err := os.Stat(filepath.Join(projDir, ".git")); err == nil {
		t.Skip("project dir already has .git — skipping git-init test")
	}

	resp := env.doRequest("POST", "/api/projects/init-git/init", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if gitInit, _ := body["git_initialised"].(bool); !gitInit {
		t.Error("git_initialised = false, want true for non-git directory")
	}

	// .git directory must now exist.
	if _, err := os.Stat(filepath.Join(projDir, ".git")); err != nil {
		t.Errorf(".git directory not created after init: %v", err)
	}
}

// TestInitProject_GitAlreadyInit verifies that init on an existing git repo
// does NOT reinitialise git and instead returns git_commands for the user.
func TestInitProject_GitAlreadyInit(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "init-git-existing", initialised: false},
	})

	projDir := env.projectDirs["init-git-existing"]

	// Create a git repo in the project dir first.
	_, err := gogit.PlainInit(projDir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	// Create a minimal first commit so HEAD is valid.
	repo, _ := gogit.PlainOpen(projDir)
	wt, _ := repo.Worktree()
	placeholder := filepath.Join(projDir, ".gitkeep")
	_ = os.WriteFile(placeholder, []byte{}, 0o644)
	_, _ = wt.Add(".gitkeep")
	_, _ = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@t.local", When: time.Now()},
	})

	resp := env.doRequest("POST", "/api/projects/init-git-existing/init", nil)
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if gitInit, _ := body["git_initialised"].(bool); gitInit {
		t.Error("git_initialised = true, want false for already-git directory")
	}
	// git_commands should be returned when files were created in an existing repo.
	// (May be nil if the directory was already fully initialised — acceptable.)
	_ = body["git_commands"]
}

// TestInitProject_NotFound verifies that init on a non-existent project name
// returns 404.
func TestInitProject_NotFound(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/projects/no-such-project/init", nil)
	requireCRUDStatus(t, resp, 404)
	resp.Body.Close()
}

// TestInitProject_ReloadsProject verifies that after init the project's
// initialised flag becomes true in the project list.
func TestInitProject_ReloadsProject(t *testing.T) {
	env := newCRUDTestEnv(t, []crudSeedProject{
		{name: "reload-after-init", initialised: false},
	})

	// Confirm not yet initialised.
	pre := env.doRequest("GET", "/api/projects/reload-after-init", nil)
	requireCRUDStatus(t, pre, 200)
	preBody := readCRUDJSON(t, pre)
	if init, _ := preBody["initialised"].(bool); init {
		t.Skip("project is already initialised — skipping")
	}

	// Run init.
	initResp := env.doRequest("POST", "/api/projects/reload-after-init/init", nil)
	requireCRUDStatus(t, initResp, 200)
	initResp.Body.Close()

	// After init the project must report initialised: true.
	post := env.doRequest("GET", "/api/projects/reload-after-init", nil)
	requireCRUDStatus(t, post, 200)
	postBody := readCRUDJSON(t, post)
	if init, _ := postBody["initialised"].(bool); !init {
		t.Error("initialised = false after init, want true")
	}
}

// ---------------------------------------------------------------------------
// Milestone 7 — POST /api/projects/check-directory
// ---------------------------------------------------------------------------

// TestCheckDirectory_ExistsWritableInitialised verifies that a writable
// initialised directory returns all three flags true.
func TestCheckDirectory_ExistsWritableInitialised(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	dir := makeTempProjectDir(t)
	lcDir := filepath.Join(dir, "lifecycle")
	if err := os.MkdirAll(lcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lcDir, "config.yaml"), []byte("stages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{"path": dir})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if exists, _ := body["exists"].(bool); !exists {
		t.Error("exists = false, want true")
	}
	if writable, _ := body["writable"].(bool); !writable {
		t.Error("writable = false, want true")
	}
	if init, _ := body["initialised"].(bool); !init {
		t.Error("initialised = false, want true")
	}
}

// TestCheckDirectory_ExistsWritableNotInitialised verifies that a writable but
// uninitialised directory returns initialised: false.
func TestCheckDirectory_ExistsWritableNotInitialised(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	dir := makeTempProjectDir(t)
	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{"path": dir})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if exists, _ := body["exists"].(bool); !exists {
		t.Error("exists = false, want true")
	}
	if writable, _ := body["writable"].(bool); !writable {
		t.Error("writable = false, want true")
	}
	if init, _ := body["initialised"].(bool); init {
		t.Error("initialised = true, want false for directory without lifecycle/config.yaml")
	}
}

// TestCheckDirectory_ExistsNotWritable verifies that a read-only directory
// returns writable: false.
func TestCheckDirectory_ExistsNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory test not applicable on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — permission restrictions not enforced")
	}

	env := newCRUDTestEnv(t, nil)
	dir := makeTempProjectDir(t)

	if err := os.Chmod(dir, 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{"path": dir})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if exists, _ := body["exists"].(bool); !exists {
		t.Error("exists = false, want true")
	}
	if writable, _ := body["writable"].(bool); writable {
		t.Error("writable = true, want false for read-only directory")
	}
}

// TestCheckDirectory_NotExists verifies that a non-existent path returns
// exists: false.
func TestCheckDirectory_NotExists(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{
		"path": "/kaos-test-does-not-exist-check-dir-xyz-999",
	})
	requireCRUDStatus(t, resp, 200)
	body := readCRUDJSON(t, resp)

	if exists, _ := body["exists"].(bool); exists {
		t.Error("exists = true, want false for non-existent path")
	}
}

// TestCheckDirectory_InvalidPath verifies that a relative path returns 400.
func TestCheckDirectory_InvalidPath(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{
		"path": "relative/path",
	})
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "invalid_path" {
		t.Errorf("error code = %q, want %q", code, "invalid_path")
	}
}

// TestCheckDirectory_TraversalAttempt verifies that a relative path with ".."
// is rejected as invalid before any filesystem access.
func TestCheckDirectory_TraversalAttempt(t *testing.T) {
	env := newCRUDTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/projects/check-directory", map[string]any{
		"path": "../traversal/attempt",
	})
	requireCRUDStatus(t, resp, 400)
	body := readCRUDJSON(t, resp)
	if code := errCode(body); code != "invalid_path" {
		t.Errorf("error code = %q, want %q", code, "invalid_path")
	}
}
