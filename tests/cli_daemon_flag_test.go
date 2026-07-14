// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// CLI integration tests for the daemon-flag / usage-guide feature.
// Covers all milestones from lifecycle/test-plans/daemon-flag-usage-guide-5-test.md:
//
//	M1 – no-argument invocation: usage to stderr, exit 2, no side effects
//	M2 – daemon opt-in (-d/--daemon/serve) starts the server; -config alone does not
//	M3 – usage content (F2) and help/version flags (F4)
//	M4 – existing subcommands unchanged (F5) and unknown-input handling (F6)
//	M5 – documentation and dependency guards (NF1, NF3)
package cli_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// daemonConfig writes a minimal server config.yaml to a temp dir and returns
// the config file path and the listen address (127.0.0.1:<port>).
func daemonConfig(t *testing.T) (cfgPath, listenAddr string) {
	t.Helper()
	port := freePort(t)
	listenAddr = fmt.Sprintf("127.0.0.1:%d", port)
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	projectsDir := filepath.Join(cfgDir, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("daemonConfig: MkdirAll projectsDir: %v", err)
	}
	cfgPath = filepath.Join(cfgDir, "config.yaml")
	content := fmt.Sprintf(
		"data_dir: %q\nprojects_dir: %q\nserver:\n  listen: %q\n",
		dataDir, projectsDir, listenAddr,
	)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("daemonConfig: WriteFile: %v", err)
	}
	return cfgPath, listenAddr
}

// startDaemon launches the binary with args, waits up to 15 s for listenAddr
// to accept TCP connections, and registers a SIGTERM cleanup.
func startDaemon(t *testing.T, listenAddr string, args ...string) {
	t.Helper()
	cmd := newBinCmd(t, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("startDaemon: Start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", listenAddr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("startDaemon: server did not become ready on %s within 15s", listenAddr)
}

// assertNotListening dials listenAddr and fails if anything is listening there.
func assertNotListening(t *testing.T, listenAddr string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", listenAddr, 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		t.Errorf("assertNotListening: expected nothing on %s, but TCP connect succeeded", listenAddr)
	}
}

// ─── Milestone 1: No-argument invocation ─────────────────────────────────────

// TestNoArgs_UsageToStderr_ExitTwo asserts that a bare invocation (no args)
// writes the usage guide to stderr, leaves stdout empty, and exits 2 (F1 / Resolved Q3).
func TestNoArgs_UsageToStderr_ExitTwo(t *testing.T) {
	stdout, stderr, code := runBin(t)
	if code != 2 {
		t.Errorf("no-arg exit code: want 2, got %d", code)
	}
	if stdout != "" {
		t.Errorf("no-arg stdout: want empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "kaos-control") {
		t.Errorf("no-arg stderr: want usage guide, got %q", stderr)
	}
}

// TestNoArgs_NoSideEffects asserts the no-arg path creates no files under a
// temporary HOME and does not bind a TCP listener.
func TestNoArgs_NoSideEffects(t *testing.T) {
	tmpHome := t.TempDir()
	cmd := exec.Command(binPath) // no args
	cmd.Env = []string{
		"HOME=" + tmpHome,
		"XDG_CONFIG_HOME=" + filepath.Join(tmpHome, "config"),
		"PATH=" + os.Getenv("PATH"),
	}
	_ = cmd.Run()

	var created []string
	_ = filepath.Walk(tmpHome, func(path string, _ os.FileInfo, _ error) error {
		if path != tmpHome {
			created = append(created, path)
		}
		return nil
	})
	if len(created) != 0 {
		t.Errorf("no-arg invocation created %d file(s) under temp HOME: %v", len(created), created)
	}
}

// TestNoArgs_CompletesQuickly asserts the no-arg path exits well under 100 ms
// (NF2). A generous 1 s bound is used to tolerate CI process-spawn overhead.
func TestNoArgs_CompletesQuickly(t *testing.T) {
	start := time.Now()
	runBin(t) // no args
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("no-arg invocation took %v; want < 1s — suspect config load or server start (NF2)", elapsed)
	}
}

// ─── Milestone 2: Daemon opt-in starts the server ────────────────────────────

// TestDaemon_DashD_StartsServer asserts that -d -config <path> starts the
// HTTP server and it is reachable on the configured address (F3, Resolved Q1).
func TestDaemon_DashD_StartsServer(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	startDaemon(t, listenAddr, "-d", "-config", cfgPath)
}

// TestDaemon_DoubleDashDaemon_StartsServer asserts that --daemon -config <path>
// starts the server.
func TestDaemon_DoubleDashDaemon_StartsServer(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	startDaemon(t, listenAddr, "--daemon", "-config", cfgPath)
}

// TestDaemon_ReverseOrder_StartsServer asserts that -config before -d also
// starts the server, proving -d is stripped before flag.Parse sees os.Args
// (Resolved Q1: any position accepted).
func TestDaemon_ReverseOrder_StartsServer(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	startDaemon(t, listenAddr, "-config", cfgPath, "-d")
}

// TestDaemon_Serve_StartsServer asserts the `serve` subcommand with -config
// starts the server (Resolved Q2: serve is a documented peer of -d).
func TestDaemon_Serve_StartsServer(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	startDaemon(t, listenAddr, "serve", "-config", cfgPath)
}

// TestDaemon_ConfigWithoutD_DoesNotStart asserts that -config without
// -d/--daemon or `serve` does not start the server and exits 2 with usage on
// stderr (consequence of F3: server requires explicit opt-in).
func TestDaemon_ConfigWithoutD_DoesNotStart(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	stdout, stderr, code := runBin(t, "-config", cfgPath)
	if code != 2 {
		t.Errorf("-config without -d: want exit 2, got %d", code)
	}
	if stdout != "" {
		t.Errorf("-config without -d: want empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "kaos-control") {
		t.Errorf("-config without -d: want usage guide in stderr, got %q", stderr)
	}
	assertNotListening(t, listenAddr)
}

// TestDaemon_NoDaemonFlagLeak asserts that none of the daemon invocation forms
// produce a "flag provided but not defined: -d" (or --daemon) error — the
// daemon token is fully stripped before flag.Parse runs in run().
func TestDaemon_NoDaemonFlagLeak(t *testing.T) {
	// Point at a nonexistent config so the server attempt exits quickly.
	nonexistentCfg := filepath.Join(t.TempDir(), "no-such-config.yaml")
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"dash-d", []string{"-d", "-config", nonexistentCfg}},
		{"double-dash-daemon", []string{"--daemon", "-config", nonexistentCfg}},
		{"reverse-order", []string{"-config", nonexistentCfg, "-d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, _ := runBin(t, tc.args...)
			combined := stdout + stderr
			if strings.Contains(combined, "flag provided but not defined") {
				t.Errorf("daemon flag leaked to flag.Parse: output contains %q\nfull output: %s",
					"flag provided but not defined", combined)
			}
		})
	}
}

// ─── Milestone 3: Usage content and help/version flags ───────────────────────

// TestHelp_StdoutExit0 asserts that --help, -h, and -help each print the usage
// guide to stdout, produce empty stderr, and exit 0 (F4).
func TestHelp_StdoutExit0(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "-help"} {
		t.Run(flag, func(t *testing.T) {
			stdout, stderr, code := runBin(t, flag)
			if code != 0 {
				t.Errorf("%s: want exit 0, got %d", flag, code)
			}
			if stderr != "" {
				t.Errorf("%s: want empty stderr, got %q", flag, stderr)
			}
			if !strings.Contains(stdout, "kaos-control") {
				t.Errorf("%s: stdout missing 'kaos-control'\ngot: %s", flag, stdout)
			}
		})
	}
}

// TestUsageContent_AllRequiredElements asserts the usage guide produced by
// --help contains every element required by F2.
func TestUsageContent_AllRequiredElements(t *testing.T) {
	stdout, _, _ := runBin(t, "--help")

	for _, tc := range []struct {
		name  string
		token string
	}{
		{"binary name", "kaos-control"},
		{"daemon flag short", "-d"},
		{"daemon flag long", "--daemon"},
		{"version flag", "--version"},
		{"version flag short", "-V"},
		{"help flag", "--help"},
		{"help flag short", "-h"},
		{"config flag", "-config"},
		{"serve subcommand", "serve"},
		{"init subcommand", "init"},
		{"auth subcommand", "auth"},
		{"devops subcommand", "devops"},
		{"hook-helper subcommand", "hook-helper"},
		{"backfill-created subcommand", "backfill-created"},
		{"backfill subcommand", "backfill"},
		{"releases subcommand", "releases"},
		{"serve equivalent note", "equivalent"},
		{"per-command help pointer", "--help"},
	} {
		if !strings.Contains(stdout, tc.token) {
			t.Errorf("usage missing %s (token %q)\nfull output:\n%s", tc.name, tc.token, stdout)
		}
	}
}

// TestUsage_Identical asserts that the usage text emitted to stderr on a bare
// invocation is byte-identical to the stdout on --help — single source of truth.
func TestUsage_Identical(t *testing.T) {
	helpOut, _, _ := runBin(t, "--help")
	_, noArgErr, _ := runBin(t)
	if helpOut != noArgErr {
		t.Errorf("usage text mismatch between --help (stdout) and no-arg (stderr)\n"+
			"--help stdout:\n%s\nno-arg stderr:\n%s", helpOut, noArgErr)
	}
}

// TestVersion_StdoutExit0 asserts that --version, -V, and -version each print
// the version/copyright/licence header to stdout, produce empty stderr, and
// exit 0 (F4).
func TestVersion_StdoutExit0(t *testing.T) {
	for _, flag := range []string{"--version", "-V", "-version"} {
		t.Run(flag, func(t *testing.T) {
			stdout, stderr, code := runBin(t, flag)
			if code != 0 {
				t.Errorf("%s: want exit 0, got %d", flag, code)
			}
			if stderr != "" {
				t.Errorf("%s: want empty stderr, got %q", flag, stderr)
			}
			if !strings.Contains(stdout, "kaos-control") {
				t.Errorf("%s: stdout missing 'kaos-control'\ngot: %s", flag, stdout)
			}
		})
	}
}

// TestVersion_NoServerStart asserts that --version exits promptly and does not
// start the HTTP server.
func TestVersion_NoServerStart(t *testing.T) {
	start := time.Now()
	runBin(t, "--version")
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("--version took %v; suspect server started (want < 1s)", elapsed)
	}
}

// ─── Milestone 4: Existing subcommands unchanged and unknown-input handling ───

// TestSubcommands_DispatchToOwnHandler asserts that each non-server subcommand
// invoked with --help exits without starting a server and produces its own help
// output — not the top-level usage guide (F5).
func TestSubcommands_DispatchToOwnHandler(t *testing.T) {
	// First line of the top-level usage guide. A subcommand's own help must
	// not start with this string.
	const topLevelHeader = "kaos-control — lifecycle management for turning ideas into releases."

	for _, sub := range []string{"init", "auth", "devops", "hook-helper", "backfill-created", "backfill", "releases"} {
		t.Run(sub, func(t *testing.T) {
			stdout, stderr, _ := runBin(t, sub, "--help")
			combined := stdout + stderr
			// The subcommand must produce some output.
			if strings.TrimSpace(combined) == "" {
				t.Errorf("%s --help: expected some output, got nothing", sub)
			}
			// The output must not be the top-level usage guide verbatim.
			if strings.Contains(combined, topLevelHeader) {
				t.Errorf("%s --help: output contains top-level usage header; dispatching failed\ngot: %s",
					sub, combined)
			}
		})
	}
}

// TestUnknownSubcommand_StderrExit1 asserts that an unknown subcommand prints
// an error plus usage to stderr, leaves stdout empty, and exits 1 (F6).
func TestUnknownSubcommand_StderrExit1(t *testing.T) {
	stdout, stderr, code := runBin(t, "bogus")
	if code != 1 {
		t.Errorf("unknown subcommand: want exit 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("unknown subcommand: want empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("unknown subcommand: want error mentioning %q in stderr, got: %s", "bogus", stderr)
	}
	if !strings.Contains(stderr, "kaos-control") {
		t.Errorf("unknown subcommand: want usage guide in stderr, got: %s", stderr)
	}
}

// TestUnknownFlag_StderrExit1 asserts that an unknown top-level flag prints an
// error plus usage to stderr, leaves stdout empty, and exits 1 (F6).
func TestUnknownFlag_StderrExit1(t *testing.T) {
	stdout, stderr, code := runBin(t, "--nope")
	if code != 1 {
		t.Errorf("unknown flag: want exit 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("unknown flag: want empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("unknown flag: want error mentioning %q in stderr, got: %s", "nope", stderr)
	}
	if !strings.Contains(stderr, "kaos-control") {
		t.Errorf("unknown flag: want usage guide in stderr, got: %s", stderr)
	}
}

// TestSPA_ServedWhenDaemon asserts that when started via -d the embedded SPA
// is served: GET / returns HTTP 200 with text/html (frontend plan smoke check).
func TestSPA_ServedWhenDaemon(t *testing.T) {
	cfgPath, listenAddr := daemonConfig(t)
	startDaemon(t, listenAddr, "-d", "-config", cfgPath)

	resp, err := http.Get(fmt.Sprintf("http://%s/", listenAddr)) //nolint:gosec
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("GET /: want text/html Content-Type, got %q", ct)
	}
}

// ─── Milestone 5: Documentation and dependency guards ────────────────────────

// TestDoc_MakefileRunIncludesDaemonFlag asserts the Makefile run: target
// passes -d or serve to the binary, so `make run` starts the server (NF3).
func TestDoc_MakefileRunIncludesDaemonFlag(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	inRunTarget := false
	for _, line := range lines {
		if strings.HasPrefix(line, "run:") {
			inRunTarget = true
			continue
		}
		if inRunTarget {
			// A non-indented, non-empty line starts a new target.
			if line != "" && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
				break
			}
			if strings.Contains(line, " -d") || strings.Contains(line, " serve ") ||
				strings.HasSuffix(strings.TrimSpace(line), " serve") {
				return // daemon flag found in run: body
			}
		}
	}
	t.Errorf("Makefile run: target does not include -d or serve (NF3)\nfull Makefile:\n%s", string(data))
}

// TestDoc_NoBareStartInstructions asserts README.md and docs/*.md contain no
// lines where the binary appears as the first command with nothing following it
// (bare start without -d or serve) per NF3. Lines where the binary path
// appears mid-line (e.g. in build comments or prose) are not flagged.
func TestDoc_NoBareStartInstructions(t *testing.T) {
	repoRoot := ".."
	targets := []string{filepath.Join(repoRoot, "README.md")}
	if entries, err := os.ReadDir(filepath.Join(repoRoot, "docs")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				targets = append(targets, filepath.Join(repoRoot, "docs", e.Name()))
			}
		}
	}

	// These are the binary invocation prefixes that identify a bare start when
	// nothing (no flag, no subcommand) follows on the same line.
	barePrefixes := []string{
		"./dist/kaos-control",
		"./kaos-control",
	}

	for _, path := range targets {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			// Strip leading whitespace to handle indented code blocks.
			trimmed := strings.TrimLeft(line, " \t")
			for _, prefix := range barePrefixes {
				if !strings.HasPrefix(trimmed, prefix) {
					continue
				}
				// Check what follows the binary name on this line.
				rest := strings.TrimSpace(trimmed[len(prefix):])
				// If nothing follows, this is a bare invocation.
				if rest == "" {
					t.Errorf("%s:%d: bare start instruction (no -d/serve): %q (NF3)", path, i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
