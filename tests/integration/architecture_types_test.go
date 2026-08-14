// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 1
// (FR-18): the architecture/tech-stack/adr type vocabulary indexes cleanly end
// to end (parser → index → API) and an unknown type still surfaces a parse
// error, guarding that the vocabulary was extended rather than disabled.

import (
	"testing"
)

// makeCleanSlugArtifact builds a markdown artifact with no lineage/parent
// fields, as used by standing reference artefacts under lifecycle/architecture/.
func makeCleanSlugArtifact(title, typ, status, body string) string {
	return "---\n" +
		"title: " + title + "\n" +
		"type: " + typ + "\n" +
		"status: " + status + "\n" +
		"---\n\n" + body + "\n"
}

// findArtifactRow locates an artifact row by path in the "items" (or
// "artifacts") array of a /artifacts list response.
func findArtifactRow(t *testing.T, data map[string]any, path string) map[string]any {
	t.Helper()
	items, _ := data["items"].([]any)
	for _, raw := range items {
		row, _ := raw.(map[string]any)
		if row["path"] == path {
			return row
		}
	}
	return nil
}

// parseErrorPaths extracts the set of paths present in a /parse-errors response.
func parseErrorPaths(t *testing.T, data map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{}
	errs, _ := data["errors"].([]any)
	for _, raw := range errs {
		e, _ := raw.(map[string]any)
		path, _ := e["path"].(string)
		msg, _ := e["message"].(string)
		out[path] = msg
	}
	return out
}

// TestArchitectureTypes_IndexCleanly seeds one artifact of each new type
// (architecture, tech-stack, adr) under lifecycle/architecture/ and asserts
// each is retrievable via GET /artifacts with the correct type and records
// no parse error.
func TestArchitectureTypes_IndexCleanly(t *testing.T) {
	archPath := "lifecycle/architecture/architectures/tt-arch.md"
	stackPath := "lifecycle/architecture/tech-stacks/tt-stack.md"
	adrPath := "lifecycle/architecture/decisions/adr-0001-tt.md"

	seeds := []seedArtifact{
		{relPath: archPath, content: makeCleanSlugArtifact("TT Architecture", "architecture", "draft", "Body.")},
		{relPath: stackPath, content: makeCleanSlugArtifact("TT Tech Stack", "tech-stack", "draft", "Body.")},
		{relPath: adrPath, content: makeCleanSlugArtifact("TT Adopt", "adr", "draft", "Body.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	cases := []struct {
		path    string
		wantTyp string
	}{
		{archPath, "architecture"},
		{stackPath, "tech-stack"},
		{adrPath, "adr"},
	}
	for _, c := range cases {
		row := findArtifactRow(t, data, c.path)
		if row == nil {
			t.Errorf("expected %q in /artifacts response, not found", c.path)
			continue
		}
		if got, _ := row["type"].(string); got != c.wantTyp {
			t.Errorf("%s: type = %q, want %q", c.path, got, c.wantTyp)
		}
	}

	perrResp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, perrResp, 200)
	perrData := readJSON(t, perrResp)
	perrs := parseErrorPaths(t, perrData)

	for _, c := range cases {
		if msg, ok := perrs[c.path]; ok {
			t.Errorf("%s: unexpected parse error: %s", c.path, msg)
		}
	}
}

// TestArchitectureTypes_UnknownTypeStillRejected guards that extending the
// vocabulary with architecture/tech-stack/adr did not accidentally disable
// type validation altogether: a bogus type must still surface a parse error.
func TestArchitectureTypes_UnknownTypeStillRejected(t *testing.T) {
	bogusPath := "lifecycle/architecture/tt-bogus.md"
	seeds := []seedArtifact{
		{relPath: bogusPath, content: makeCleanSlugArtifact("TT Bogus", "bogus", "draft", "Body.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	perrs := parseErrorPaths(t, data)

	msg, ok := perrs[bogusPath]
	if !ok {
		t.Fatalf("expected a parse error for %q (unknown type), found none", bogusPath)
	}
	if want := `unknown type "bogus"`; msg != want {
		t.Errorf("parse error message = %q, want %q", msg, want)
	}
}
