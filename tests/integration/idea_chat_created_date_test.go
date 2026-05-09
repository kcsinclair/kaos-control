// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Milestone 2 — idea-chat writeIdeaArtifact: RFC3339 created field stamping
// ---------------------------------------------------------------------------

// TestIdeaChatAccept_ArtifactHasRFC3339Created verifies that after accepting a
// proposed idea via the conversational endpoint, the on-disk artifact file
// contains a `created` frontmatter field parseable as RFC3339, and that the
// value is within a 10-second window of the test's start time.
//
// Requires ANTHROPIC_API_KEY (set in the environment) to drive the LLM to
// proposal state. Skipped when the key is absent.
func TestIdeaChatAccept_ArtifactHasRFC3339Created(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	before := time.Now().Add(-time.Second)

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("created-rfc3339-check"))

	acceptResp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, acceptResp, 200)
	data := readJSON(t, acceptResp)

	after := time.Now().Add(time.Second)

	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("missing artifact_path in accept response")
	}

	absPath := filepath.Join(env.projectRoot, artifactPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading written artifact at %s: %v", absPath, err)
	}
	content := string(raw)

	// Extract created field.
	createdVal := extractCreatedFromFM(content)
	if createdVal == "" {
		t.Fatalf("no `created:` field found in on-disk artifact:\n%s", content)
	}

	// Must parse as RFC3339.
	parsed, err := time.Parse(time.RFC3339, createdVal)
	if err != nil {
		t.Errorf("on-disk artifact created field %q is not valid RFC3339: %v", createdVal, err)
		return
	}

	// Must be within the test window.
	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("on-disk created %v is outside expected window [%v, %v]", parsed, before, after)
	}
}

// TestIdeaChatAccept_IndexReflectsRFC3339Created verifies that after accepting
// a proposed idea, the index row's Created field is non-zero and within the
// expected time window, matching the RFC3339 timestamp written to disk.
//
// Requires ANTHROPIC_API_KEY.
func TestIdeaChatAccept_IndexReflectsRFC3339Created(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	before := time.Now().Add(-time.Second)

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("index-created-check"))
	acceptResp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, acceptResp, 200)
	data := readJSON(t, acceptResp)

	after := time.Now().Add(time.Second)

	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("missing artifact_path in accept response")
	}

	// GET the artifact from the API and verify `created` is a valid timestamp.
	env.login("admin@test.local", "admin-pass-123")
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+artifactPath, nil)
	requireStatus(t, getResp, 200)
	getBody := readJSON(t, getResp)

	artifact, _ := getBody["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("artifact missing from GET response")
	}

	createdRaw, _ := artifact["created"].(string)
	if createdRaw == "" {
		t.Fatal("artifact.created missing in GET response")
	}

	parsed, err := time.Parse(time.RFC3339, createdRaw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, createdRaw)
		if err != nil {
			t.Fatalf("artifact.created %q in API response is not valid RFC3339: %v", createdRaw, err)
		}
	}

	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("API artifact.created %v is outside expected window [%v, %v]", parsed, before, after)
	}
}
