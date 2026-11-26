package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func grepEnv(t *testing.T) *ToolExecutor {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lifecycle/tests/a.md", "NEEDLE in a real artifact\n")
	write("web/dist/bundle.js", "var x=1;//NEEDLE\n")
	write("node_modules/pkg/index.js", "NEEDLE in a dependency\n")
	write("long.txt", "NEEDLE "+strings.Repeat("x", 5000)+"\n")
	return &ToolExecutor{ProjectRoot: root}
}

// TestGrep_EmptyPathRejected pins the fix for run ec4f45c70ba0b39a, where
// grep("FAIL", "") walked the whole repo and returned 82 KB — mostly minified
// JS — which the model then mistook for test output.
func TestGrep_EmptyPathRejected(t *testing.T) {
	e := grepEnv(t)
	for _, p := range []string{"", ".", "/"} {
		out, err := e.grep("NEEDLE", p)
		if err != nil {
			t.Fatalf("grep(%q) returned error: %v", p, err)
		}
		if !strings.Contains(out, "path is required") {
			t.Errorf("grep(%q) should refuse a whole-repo search, got: %.80s", p, out)
		}
		if strings.Contains(out, "bundle.js") {
			t.Errorf("grep(%q) searched build output", p)
		}
	}
}

func TestGrep_SkipsBuildOutputAndDependencies(t *testing.T) {
	e := grepEnv(t)
	out, err := e.grep("NEEDLE", "web")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "bundle.js") {
		t.Error("web/dist should be skipped")
	}
	if out2, _ := e.grep("NEEDLE", "node_modules"); strings.Contains(out2, "index.js") {
		t.Error("node_modules should be skipped")
	}
}

func TestGrep_StillFindsRealMatches(t *testing.T) {
	e := grepEnv(t)
	out, err := e.grep("NEEDLE", "lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "lifecycle/tests/a.md") {
		t.Errorf("expected the real artifact match, got: %q", out)
	}
}

func TestGrep_TruncatesLongLines(t *testing.T) {
	e := grepEnv(t)
	out, err := e.grep("NEEDLE", "long.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > grepLineLimit+200 {
		t.Errorf("long line not truncated: %d bytes", len(out))
	}
	if !strings.Contains(out, "line truncated") {
		t.Error("truncation should be announced")
	}
}
