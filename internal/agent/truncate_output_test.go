package agent

import (
	"strings"
	"testing"
)

// TestTruncateCommandOutput_KeepsVerdictAtTheEnd pins the behaviour that run
// 8fe31d94d48f2b67 depended on: `go test` and `vitest` print their verdict
// last, so truncation must not discard the tail. That run reported "All tests
// passed" from head-truncated output containing no verdict at all.
func TestTruncateCommandOutput_KeepsVerdictAtTheEnd(t *testing.T) {
	verdict := "--- FAIL: TestFailover_AutoSwitch_HTTP529 (0.52s)\nFAIL\n"
	body := strings.Repeat("2026/08/28 INFO index rebuilding from disk db=/tmp/x\n", 2000)
	got := truncateCommandOutput(body + verdict)

	if len(got) > bashOutputLimit+256 {
		t.Errorf("result length %d exceeds cap %d plus marker allowance", len(got), bashOutputLimit)
	}
	if !strings.Contains(got, "--- FAIL: TestFailover_AutoSwitch_HTTP529") {
		t.Error("verdict lost — the failing test name must survive truncation")
	}
	if !strings.HasSuffix(got, verdict) {
		t.Error("output must end with the real tail of the command output")
	}
	if !strings.HasPrefix(got, "2026/08/28 INFO") {
		t.Error("head must be kept so the early output is still visible")
	}
	if !strings.Contains(got, "bytes of output omitted") {
		t.Error("truncation must be announced")
	}
}

func TestTruncateCommandOutput_ShortOutputUnchanged(t *testing.T) {
	in := "ok  \tgithub.com/kaos-control/kaos-control/internal/agent\t1.2s\n"
	if got := truncateCommandOutput(in); got != in {
		t.Errorf("short output altered:\n got %q\nwant %q", got, in)
	}
}

func TestTruncateCommandOutput_DoesNotSplitRunes(t *testing.T) {
	// vitest output is full of multi-byte check marks and ANSI colour.
	in := strings.Repeat("✓ src/components/map/__tests__/graphConstants.spec.ts\n", 2000) + "Tests  277 passed\n"
	got := truncateCommandOutput(in)
	if strings.ContainsRune(got, '�') {
		t.Error("truncation split a UTF-8 rune, producing U+FFFD")
	}
	if !strings.Contains(got, "Tests  277 passed") {
		t.Error("verdict lost")
	}
}
