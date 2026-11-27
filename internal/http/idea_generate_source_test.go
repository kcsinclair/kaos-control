// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildDocSourceContext sends file contents to the LLM, and source_path comes
// from the request body, so it must only read indexed artifacts inside the
// project — never an arbitrary file reached through "../".
func TestBuildDocSourceContext_OnlyReadsIndexedArtifacts(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	reqRel := "lifecycle/requirements/login-2.md"
	reqAbs := filepath.Join(p.Entry.Path, reqRel)
	if err := os.WriteFile(reqAbs, []byte("---\ntitle: Login\ntype: requirement\nstatus: draft\nlineage: login\n---\n\nREQUIREMENT-BODY\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := p.Idx.IndexFile(reqAbs); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	// A secret outside the project root, and an unindexed file inside it.
	secret := filepath.Join(filepath.Dir(p.Entry.Path), "secret.yaml")
	if err := os.WriteFile(secret, []byte("api_key: OUTSIDE-SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Entry.Path, ".env"), []byte("TOKEN=INSIDE-UNINDEXED\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := buildDocSourceContext(p, "login", reqRel); !strings.Contains(got, "REQUIREMENT-BODY") {
		t.Errorf("indexed source artifact not included: %q", got)
	}
	for _, path := range []string{"../secret.yaml", ".env"} {
		if got := buildDocSourceContext(p, "login", path); strings.Contains(got, "SECRET") || strings.Contains(got, "INSIDE-UNINDEXED") {
			t.Errorf("source_path %q leaked file contents into the prompt: %q", path, got)
		}
	}
}
