// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

func goVueModel() DirectiveModel {
	return DirectiveModel{
		ProjectName: "kaos-control",
		StackTitle:  "Go + Vue (High-Performance Lean Stack)",
		RepoLayout: []architecture.RepoLayoutEntry{
			{Path: "internal/", Note: "Go packages — backend logic"},
			{Path: "cmd/", Note: "binary entry points"},
			{Path: "web/src/", Note: "Vue 3 + TypeScript SPA source"},
			{Path: "web/dist/", Note: "built SPA, embedded into the binary"},
			{Path: "tests/", Note: "integration + e2e tests"},
		},
		ArchitecturePointer: true,
		Stack: architecture.StackProfile{
			Run: "go run ./cmd/<app>",
			Roles: map[string]architecture.RoleProfile{
				"backend-developer": {
					WritePaths: []string{"internal", "cmd"},
					Build:      "go build ./...",
					Lint:       "go vet ./...",
					Test:       "go test ./... -short",
				},
				"frontend-developer": {
					WritePaths: []string{"web/src"},
					Build:      "cd web && pnpm build",
				},
			},
		},
	}
}

func TestRenderAgents_GoVue(t *testing.T) {
	out, err := RenderAgents(goVueModel())
	if err != nil {
		t.Fatalf("RenderAgents: %v", err)
	}

	for _, want := range []string{"internal/", "cmd/", "web/src/", "lifecycle/architecture/"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	if !bytes.HasPrefix(out, []byte(genStart)) {
		t.Errorf("output does not start with genStart marker:\n%s", out)
	}
	if !bytes.Contains(out, []byte(genEnd)) {
		t.Errorf("output missing genEnd marker:\n%s", out)
	}
}

func TestRenderAgents_Deterministic(t *testing.T) {
	m := goVueModel()
	a, err := RenderAgents(m)
	if err != nil {
		t.Fatalf("RenderAgents: %v", err)
	}
	b, err := RenderAgents(m)
	if err != nil {
		t.Fatalf("RenderAgents: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("RenderAgents not deterministic:\n--- a ---\n%s\n--- b ---\n%s", a, b)
	}
}

func TestRenderPointer(t *testing.T) {
	got := RenderPointer("AGENTS.md")
	if string(got) != "@AGENTS.md\n" {
		t.Errorf("RenderPointer: got %q", got)
	}
}

func TestMergeManaged_PreservesProseOutsideMarkers(t *testing.T) {
	existing := []byte("# My notes\n\nSome user prose above.\n\n" +
		genStart + "\nOLD GENERATED CONTENT\n" + genEnd +
		"\n\nUser prose below, never touched.\n")
	fresh := []byte(genStart + "\nNEW GENERATED CONTENT\n" + genEnd + "\n")

	merged, changed, err := mergeManaged(existing, fresh)
	if err != nil {
		t.Fatalf("mergeManaged: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if !strings.Contains(string(merged), "Some user prose above.") {
		t.Error("prose above markers was not preserved")
	}
	if !strings.Contains(string(merged), "User prose below, never touched.") {
		t.Error("prose below markers was not preserved")
	}
	if strings.Contains(string(merged), "OLD GENERATED CONTENT") {
		t.Error("old generated content was not replaced")
	}
	if !strings.Contains(string(merged), "NEW GENERATED CONTENT") {
		t.Error("new generated content missing")
	}
}

func TestMergeManaged_NoMarkers_WholeFileReplaced(t *testing.T) {
	existing := []byte("# Legacy hand-written file\n\nNo markers here.\n")
	fresh := []byte(genStart + "\nNEW\n" + genEnd + "\n")

	merged, changed, err := mergeManaged(existing, fresh)
	if err != nil {
		t.Fatalf("mergeManaged: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if !bytes.Equal(merged, fresh) {
		t.Errorf("expected whole-file replacement, got:\n%s", merged)
	}
}

func TestMergeManaged_NoChange(t *testing.T) {
	body := []byte(genStart + "\nSAME\n" + genEnd + "\n")
	merged, changed, err := mergeManaged(body, body)
	if err != nil {
		t.Fatalf("mergeManaged: %v", err)
	}
	if changed {
		t.Error("expected changed=false for identical content")
	}
	if !bytes.Equal(merged, body) {
		t.Errorf("expected unchanged content, got:\n%s", merged)
	}
}
