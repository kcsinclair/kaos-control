//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeArtifactWithRelease builds a markdown artifact that includes a release
// frontmatter field. Pass release="" to omit the field.
func makeArtifactWithRelease(title, typ, status, lineage, release, body string) string {
	s := "---\ntitle: " + title + "\ntype: " + typ + "\nstatus: " + status + "\nlineage: " + lineage + "\n"
	if release != "" {
		s += "release: " + release + "\n"
	}
	s += "---\n\n" + body + "\n"
	return s
}

// createRelease POSTs a new release and returns the decoded response body.
// It fails the test if the response is not 201.
func createRelease(t *testing.T, env *testEnv, body map[string]any) map[string]any {
	t.Helper()
	resp := env.doRequest("POST", "/api/p/testproject/releases", body)
	requireStatus(t, resp, http.StatusCreated)
	return readJSON(t, resp)
}

// releaseID extracts the numeric id from a createRelease response.
func releaseID(t *testing.T, data map[string]any) int64 {
	t.Helper()
	rel, _ := data["release"].(map[string]any)
	id, _ := rel["id"].(float64)
	if id == 0 {
		t.Fatal("release id is zero or missing")
	}
	return int64(id)
}

// releasePath returns the /api/p/testproject/releases/{id} URL string.
func releasePath(id int64) string {
	return fmt.Sprintf("/api/p/testproject/releases/%d", id)
}

// readArtifactRelease reads the release frontmatter field from an artifact file
// on disk. Returns "" if the field is absent.
func readArtifactRelease(t *testing.T, projectRoot, relPath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(projectRoot, relPath))
	if err != nil {
		t.Fatalf("readArtifactRelease: %v", err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "release:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "release:"))
		}
	}
	return ""
}

// ── Milestone 1: Release CRUD ─────────────────────────────────────────────────

// TestReleases_CreateHappyPath verifies that a scheduled release is created with
// 201, the response includes an id, and the release persists across a GET.
func TestReleases_CreateHappyPath(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":       "v1.0",
		"status":     "planned",
		"start_date": "2026-01-01",
		"end_date":   "2026-03-31",
	})

	rel, _ := data["release"].(map[string]any)
	if rel["id"] == nil {
		t.Error("create response missing id")
	}
	if name, _ := rel["name"].(string); name != "v1.0" {
		t.Errorf("name: want %q, got %q", "v1.0", name)
	}
	if status, _ := rel["status"].(string); status != "planned" {
		t.Errorf("status: want %q, got %q", "planned", status)
	}
}

// TestReleases_CreateWithDuration verifies that supplying duration instead of
// end_date auto-calculates end_date.
func TestReleases_CreateWithDuration(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":       "v2.0-dur",
		"status":     "planned",
		"start_date": "2026-06-01",
		"duration":   "14d",
	})

	rel, _ := data["release"].(map[string]any)
	endDate, _ := rel["end_date"].(string)
	// 14 days after 2026-06-01 = 2026-06-15
	if !strings.HasPrefix(endDate, "2026-06-15") {
		t.Errorf("end_date: want 2026-06-15, got %q", endDate)
	}
}

// TestReleases_CreateUnscheduled verifies that a release with null dates is
// accepted and returns 201.
func TestReleases_CreateUnscheduled(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":   "unscheduled-v1",
		"status": "planned",
	})

	rel, _ := data["release"].(map[string]any)
	if rel["start_date"] != nil {
		t.Errorf("start_date should be nil for unscheduled release, got %v", rel["start_date"])
	}
	if rel["end_date"] != nil {
		t.Errorf("end_date should be nil for unscheduled release, got %v", rel["end_date"])
	}
}

// TestReleases_CreateDuplicateName verifies that creating a release with a
// duplicate name returns 409 Conflict.
func TestReleases_CreateDuplicateName(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "dup-release", "status": "planned"})

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "dup-release",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

// TestReleases_CreateMissingName verifies that a missing name field returns 400.
func TestReleases_CreateMissingName(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReleases_CreateNameTooLong verifies that a name exceeding 120 characters
// returns 400.
func TestReleases_CreateNameTooLong(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	longName := strings.Repeat("x", 121)
	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   longName,
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReleases_CreateEndDateBeforeStartDate verifies that end_date < start_date
// returns 400.
func TestReleases_CreateEndDateBeforeStartDate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":       "bad-dates",
		"status":     "planned",
		"start_date": "2026-06-01",
		"end_date":   "2026-01-01",
	})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReleases_CreateInvalidStatus verifies that an unknown status value
// returns 400.
func TestReleases_CreateInvalidStatus(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "bad-status",
		"status": "nonsense",
	})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReleases_ListOrderedByStartDate verifies that GET /releases returns
// releases ordered by start_date, with unscheduled last.
func TestReleases_ListOrderedByStartDate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in reverse order so sorting is exercised.
	createRelease(t, env, map[string]any{"name": "v3", "status": "planned", "start_date": "2026-07-01", "end_date": "2026-09-30"})
	createRelease(t, env, map[string]any{"name": "v1", "status": "planned", "start_date": "2026-01-01", "end_date": "2026-03-31"})
	createRelease(t, env, map[string]any{"name": "v2", "status": "planned", "start_date": "2026-04-01", "end_date": "2026-06-30"})
	createRelease(t, env, map[string]any{"name": "unscheduled", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	releases, _ := data["releases"].([]any)
	if len(releases) != 4 {
		t.Fatalf("expected 4 releases, got %d", len(releases))
	}

	names := make([]string, len(releases))
	for i, r := range releases {
		rel, _ := r.(map[string]any)
		names[i], _ = rel["name"].(string)
	}

	want := []string{"v1", "v2", "v3", "unscheduled"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("releases[%d]: want %q, got %q", i, w, names[i])
		}
	}
}

// TestReleases_ListEmptyProject verifies that an empty project returns an
// empty array (not null) for releases.
func TestReleases_ListEmptyProject(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	releases, ok := data["releases"].([]any)
	if !ok {
		t.Fatal("releases field must be an array, got nil or wrong type")
	}
	if len(releases) != 0 {
		t.Errorf("expected empty releases array, got %d items", len(releases))
	}
}

// TestReleases_GetWithCounts verifies that GET /releases/:id returns the
// release with idea_count and defect_count summary fields.
func TestReleases_GetWithCounts(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/r-get-idea-1.md",
			content: makeArtifactWithRelease("Get Idea 1", "idea", "draft", "r-get-idea-1", "v-get-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/r-get-idea-2.md",
			content: makeArtifactWithRelease("Get Idea 2", "idea", "draft", "r-get-idea-2", "v-get-1", "Body."),
		},
		{
			relPath: "lifecycle/defects/r-get-defect-1.md",
			content: makeArtifactWithRelease("Get Defect 1", "defect", "draft", "r-get-defect-1", "v-get-1", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-get-1", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	ideaCount, _ := rel["idea_count"].(float64)
	defectCount, _ := rel["defect_count"].(float64)

	if int(ideaCount) != 2 {
		t.Errorf("idea_count: want 2, got %d", int(ideaCount))
	}
	if int(defectCount) != 1 {
		t.Errorf("defect_count: want 1, got %d", int(defectCount))
	}
}

// TestReleases_GetNotFound verifies that GET on a non-existent release ID
// returns 404.
func TestReleases_GetNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/releases/99999", nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestReleases_UpdateDates verifies that PUT /releases/:id can update dates.
func TestReleases_UpdateDates(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name":       "v-upd-dates",
		"status":     "planned",
		"start_date": "2026-01-01",
		"end_date":   "2026-03-31",
	})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":       "v-upd-dates",
		"status":     "planned",
		"start_date": "2026-04-01",
		"end_date":   "2026-06-30",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	startDate, _ := rel["start_date"].(string)
	endDate, _ := rel["end_date"].(string)

	if !strings.HasPrefix(startDate, "2026-04-01") {
		t.Errorf("start_date: want 2026-04-01, got %q", startDate)
	}
	if !strings.HasPrefix(endDate, "2026-06-30") {
		t.Errorf("end_date: want 2026-06-30, got %q", endDate)
	}
}

// TestReleases_UpdateStatus verifies that the status field can be updated.
func TestReleases_UpdateStatus(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-upd-status", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-upd-status",
		"status": "active",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	if status, _ := rel["status"].(string); status != "active" {
		t.Errorf("status: want %q, got %q", "active", status)
	}
}

// TestReleases_UpdateDuplicateName verifies that renaming a release to an
// existing name returns 409.
func TestReleases_UpdateDuplicateName(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-existing", "status": "planned"})
	data2 := createRelease(t, env, map[string]any{"name": "v-to-rename", "status": "planned"})
	id2 := releaseID(t, data2)

	resp := env.doRequest("PUT", releasePath(id2), map[string]any{
		"name":   "v-existing",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

// TestReleases_UpdateNotFound verifies that PUT on a non-existent release ID
// returns 404. NOTE: the current implementation returns 500 for this case;
// this test records the intended spec behaviour and will pass once the handler
// is fixed to distinguish not-found from other DB errors.
func TestReleases_UpdateNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/releases/99999", map[string]any{
		"name":   "ghost",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestReleases_DeleteNoArtifacts verifies deleting a release with no assigned
// artifacts returns 200 with orphaned_artifact_count=0.
func TestReleases_DeleteNoArtifacts(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-del-empty", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("DELETE", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	count, _ := body["orphaned_artifact_count"].(float64)
	if int(count) != 0 {
		t.Errorf("orphaned_artifact_count: want 0, got %d", int(count))
	}

	// Release should no longer appear in the list.
	listResp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, listResp, http.StatusOK)
	listData := readJSON(t, listResp)
	releases, _ := listData["releases"].([]any)
	if len(releases) != 0 {
		t.Errorf("expected 0 releases after deletion, got %d", len(releases))
	}
}

// TestReleases_DeleteWithArtifacts verifies deleting a release with assigned
// artifacts returns the correct orphaned count.
func TestReleases_DeleteWithArtifacts(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/r-del-idea-1.md",
			content: makeArtifactWithRelease("Del Idea 1", "idea", "draft", "r-del-idea-1", "v-del-with-arts", "Body."),
		},
		{
			relPath: "lifecycle/ideas/r-del-idea-2.md",
			content: makeArtifactWithRelease("Del Idea 2", "idea", "draft", "r-del-idea-2", "v-del-with-arts", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-del-with-arts", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("DELETE", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	count, _ := body["orphaned_artifact_count"].(float64)
	if int(count) != 2 {
		t.Errorf("orphaned_artifact_count: want 2, got %d", int(count))
	}
}

// TestReleases_DeleteNotFound verifies that DELETE on a non-existent release
// returns 404. NOTE: the current implementation returns 500 for this case;
// this test records the intended spec behaviour and will pass once the handler
// is fixed to distinguish not-found from other DB errors.
func TestReleases_DeleteNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("DELETE", "/api/p/testproject/releases/99999", nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestReleases_ListArtifactsForRelease verifies that GET /releases/:id/artifacts
// returns only artifacts assigned to that release.
func TestReleases_ListArtifactsForRelease(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/r-list-arts-a.md",
			content: makeArtifactWithRelease("List Arts A", "idea", "draft", "r-list-arts-a", "v-list-arts", "Body."),
		},
		{
			relPath: "lifecycle/ideas/r-list-arts-b.md",
			content: makeArtifactWithRelease("List Arts B", "idea", "draft", "r-list-arts-b", "v-list-arts", "Body."),
		},
		{
			relPath: "lifecycle/ideas/r-list-arts-other.md",
			content: makeArtifactWithRelease("List Arts Other", "idea", "draft", "r-list-arts-other", "v-other", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-list-arts", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id)+"/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Errorf("want 2 artifacts for release, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		fm, _ := item["frontmatter"].(map[string]any)
		rel, _ := fm["release"].(string)
		if rel != "v-list-arts" {
			t.Errorf("artifact in list has release %q, want %q", rel, "v-list-arts")
		}
	}
}

// TestReleases_ListArtifactsEmpty verifies that listing artifacts for a release
// with none assigned returns an empty array.
func TestReleases_ListArtifactsEmpty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-arts-empty", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id)+"/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("items must be an array, got nil or wrong type")
	}
	if len(items) != 0 {
		t.Errorf("expected empty items, got %d", len(items))
	}
}

// ── Milestone 4: Release artifacts endpoint returns all types (no server-side filter) ──

// TestReleases_ListArtifactsReturnsAllTypes verifies that
// GET /releases/:id/artifacts returns every artifact type assigned to a
// release — the filtering to ideas+defects is intentionally client-side only,
// so the API must not silently drop requirements, plans, or other types.
func TestReleases_ListArtifactsReturnsAllTypes(t *testing.T) {
	const releaseName = "v-all-types"

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/all-types-idea.md",
			content: makeArtifactWithRelease("All Types Idea", "idea", "draft", "all-types-idea", releaseName, "Body."),
		},
		{
			relPath: "lifecycle/defects/all-types-defect.md",
			content: makeArtifactWithRelease("All Types Defect", "defect", "draft", "all-types-defect", releaseName, "Body."),
		},
		{
			relPath: "lifecycle/requirements/all-types-req.md",
			content: makeArtifactWithRelease("All Types Requirement", "requirement", "draft", "all-types-req", releaseName, "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/all-types-plan.md",
			content: makeArtifactWithRelease("All Types Plan", "plan-backend", "draft", "all-types-plan", releaseName, "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": releaseName, "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id)+"/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("want 4 artifacts (all types), got %d", len(items))
	}

	// total field must match the unfiltered count.
	total, _ := body["total"].(float64)
	if int(total) != 4 {
		t.Errorf("total: want 4, got %d", int(total))
	}

	// Collect the set of types returned.
	gotTypes := map[string]bool{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		if typ == "" {
			t.Errorf("artifact item missing type field: %v", item)
			continue
		}
		gotTypes[typ] = true
	}

	for _, want := range []string{"idea", "defect", "requirement", "plan-backend"} {
		if !gotTypes[want] {
			t.Errorf("type %q missing from artifacts response; got types: %v", want, gotTypes)
		}
	}
}

// ── Milestone 2: Release status lifecycle ─────────────────────────────────────

// TestReleaseStatus_ValidTransition verifies that a release can be created as
// planned, updated to active, then updated to shipped.
func TestReleaseStatus_ValidTransition(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-lifecycle", "status": "planned"})
	id := releaseID(t, data)

	for _, status := range []string{"active", "shipped"} {
		resp := env.doRequest("PUT", releasePath(id), map[string]any{
			"name":   "v-lifecycle",
			"status": status,
		})
		requireStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		rel, _ := body["release"].(map[string]any)
		got, _ := rel["status"].(string)
		if got != status {
			t.Errorf("after updating to %q, got status %q", status, got)
		}
	}
}

// TestReleaseStatus_AllValidOnCreate verifies that all three valid status values
// are accepted at creation time.
func TestReleaseStatus_AllValidOnCreate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	for _, status := range []string{"planned", "active", "shipped"} {
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":   "v-status-" + status,
			"status": status,
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}
}

// TestReleaseStatus_InvalidStatusRejected verifies that an invalid status value
// is rejected with 400 on both create and update.
func TestReleaseStatus_InvalidStatusRejected(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create with invalid status.
	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "v-bad-status",
		"status": "in-progress",
	})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()

	// Create valid, then update to invalid.
	data := createRelease(t, env, map[string]any{"name": "v-upd-bad", "status": "planned"})
	id := releaseID(t, data)

	resp2 := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-upd-bad",
		"status": "cancelled",
	})
	requireStatus(t, resp2, http.StatusBadRequest)
	resp2.Body.Close()
}

// TestReleaseStatus_AllStatusesInList verifies that planned, active, and
// shipped releases all appear in the GET /releases list.
func TestReleaseStatus_AllStatusesInList(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-listed-planned", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-listed-active", "status": "active"})
	createRelease(t, env, map[string]any{"name": "v-listed-shipped", "status": "shipped"})

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	releases, _ := body["releases"].([]any)
	if len(releases) != 3 {
		t.Fatalf("want 3 releases, got %d", len(releases))
	}

	statusSet := map[string]bool{}
	for _, raw := range releases {
		rel, _ := raw.(map[string]any)
		s, _ := rel["status"].(string)
		statusSet[s] = true
	}
	for _, want := range []string{"planned", "active", "shipped"} {
		if !statusSet[want] {
			t.Errorf("status %q not found in list", want)
		}
	}
}
