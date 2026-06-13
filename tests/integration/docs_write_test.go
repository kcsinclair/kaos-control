// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestDocsPut_HappyPath(t *testing.T) {
	env := newTestEnv(t, nil)
	original := "# Alpha\n\nOriginal content.\n"
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": original})

	// GET to obtain the current sha.
	getResp := env.doRequest("GET", "/api/p/testproject/docs/alpha.md", nil)
	requireStatus(t, getResp, http.StatusOK)
	getJSON := readJSON(t, getResp)
	currentSHA, _ := getJSON["file_sha"].(string)

	// PUT with the edit.
	newBody := "# Alpha\n\nEdited content.\n"
	putResp := env.doRequest("PUT", "/api/p/testproject/docs/alpha.md",
		map[string]any{"body": newBody, "expected_sha": currentSHA})
	requireStatus(t, putResp, http.StatusOK)
	putJSON := readJSON(t, putResp)

	newSHA, _ := putJSON["file_sha"].(string)
	if newSHA == "" {
		t.Error("expected non-empty file_sha in PUT response")
	}
	if newSHA == currentSHA {
		t.Error("expected file_sha to change after edit")
	}

	// Verify the file on disk was actually updated.
	getResp2 := env.doRequest("GET", "/api/p/testproject/docs/alpha.md", nil)
	requireStatus(t, getResp2, http.StatusOK)
	getJSON2 := readJSON(t, getResp2)
	if body, _ := getJSON2["body"].(string); body != newBody {
		t.Errorf("disk content: got %q, want %q", body, newBody)
	}
}

func TestDocsPut_ShaMismatch(t *testing.T) {
	env := newTestEnv(t, nil)
	original := "# Alpha\n\nOriginal.\n"
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": original})

	staleSHA := "0000000000000000000000000000000000000000000000000000000000000000"
	resp := env.doRequest("PUT", "/api/p/testproject/docs/alpha.md",
		map[string]any{"body": "# Alpha\n\nTampered.\n", "expected_sha": staleSHA})
	requireStatus(t, resp, http.StatusConflict)
	data := readJSON(t, resp)
	if code, _ := data["code"].(string); code != "sha_mismatch" {
		t.Errorf("error code: expected %q, got %q", "sha_mismatch", code)
	}

	// File must not have been modified.
	getResp := env.doRequest("GET", "/api/p/testproject/docs/alpha.md", nil)
	requireStatus(t, getResp, http.StatusOK)
	getJSON := readJSON(t, getResp)
	if body, _ := getJSON["body"].(string); body != original {
		t.Errorf("file was modified despite sha mismatch: got %q", body)
	}
}

func TestDocsPut_NotMarkdown(t *testing.T) {
	env := newTestEnv(t, nil)
	pngHeader := string([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	seedDocs(t, env.projectRoot, map[string]string{"diagram.png": pngHeader})

	resp := env.doRequest("PUT", "/api/p/testproject/docs/diagram.png",
		map[string]any{"body": "not allowed"})
	requireStatus(t, resp, http.StatusUnsupportedMediaType)
	data := readJSON(t, resp)
	if code, _ := data["code"].(string); code != "not_markdown" {
		t.Errorf("error code: expected %q, got %q", "not_markdown", code)
	}
}

func TestDocsPut_CreateNotAllowed(t *testing.T) {
	env := newTestEnv(t, nil)
	// The docs/ directory must exist for the request to reach the write path
	// rather than being caught by the sandbox before even trying.
	seedDocs(t, env.projectRoot, map[string]string{"placeholder.md": "# Placeholder\n"})

	resp := env.doRequest("PUT", "/api/p/testproject/docs/brand-new.md",
		map[string]any{"body": "# New\n\nCreated.\n"})
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDocsPut_NoRoleForbidden(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": "# Alpha\n"})

	// Create a user who exists in the auth store but has no project roles
	// (they don't appear in the project config's users section).
	if err := env.authStore.CreateUser("norole@test.local", "No Role User", "norole-pass-123", false); err != nil {
		t.Fatalf("create norole user: %v", err)
	}
	env.login("norole@test.local", "norole-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/docs/alpha.md",
		map[string]any{"body": "# Alpha\n\nUnauthorised edit.\n"})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)
	if code, _ := data["code"].(string); code != "forbidden" {
		t.Errorf("error code: expected %q, got %q", "forbidden", code)
	}
}

func TestDocsPut_QARoleAllowed(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": "# Alpha\n"})

	// qa@test.local is in RolesArtifactEditors, so PUT should succeed.
	env.login("qa@test.local", "qa-pass-123")

	newBody := "# Alpha\n\nFixed by QA.\n"
	resp := env.doRequest("PUT", "/api/p/testproject/docs/alpha.md",
		map[string]any{"body": newBody})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestDocsPut_BroadcastsDocChanged(t *testing.T) {
	env := newTestEnv(t, nil)
	seedDocs(t, env.projectRoot, map[string]string{"alpha.md": "# Alpha\n"})

	// Register hub channel BEFORE issuing the PUT so no events are missed.
	ch := make(chan []byte, 32)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Compute sha for optimistic concurrency.
	sum := sha256.Sum256([]byte("# Alpha\n"))
	currentSHA := hex.EncodeToString(sum[:])

	newBody := "# Alpha\n\nEdited for broadcast test.\n"
	putResp := env.doRequest("PUT", "/api/p/testproject/docs/alpha.md",
		map[string]any{"body": newBody, "expected_sha": currentSHA})
	requireStatus(t, putResp, http.StatusOK)
	putResp.Body.Close()

	// The PUT handler broadcasts doc.changed synchronously before returning,
	// so the event should already be in the buffered channel.
	var docChangedPath string
	timeout := time.After(500 * time.Millisecond)

COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "doc.changed" {
				docChangedPath = evt.Payload["path"]
				break COLLECT
			}
		case <-timeout:
			break COLLECT
		}
	}

	if docChangedPath == "" {
		t.Fatal("did not receive a doc.changed event within 500 ms")
	}
	if docChangedPath != "alpha.md" {
		t.Errorf("doc.changed payload.path: expected %q, got %q", "alpha.md", docChangedPath)
	}
}
