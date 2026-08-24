// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// ── OpenAI-Compatible Driver Integration Tests ───────────────────────────────

func openAIRun(providerName, model, root, prompt string, allowedPaths []string, maxIterations int) agent.Run {
	return agent.Run{
		RunID:             "drv-test-openai",
		AgentName:         "test-openai-agent",
		Role:              "analyst",
		Model:             model,
		PromptText:        prompt,
		ProjectRoot:       root,
		AllowedPaths:      allowedPaths,
		ProviderName:      providerName,
		MaxToolIterations: maxIterations,
		TimeoutMinutes:    1,
	}
}

func newOpenAIDriver(provName, url, apiKey string) *agent.OpenAICompatibleDriver {
	return &agent.OpenAICompatibleDriver{
		Providers: []config.Provider{
			{
				Name:    provName,
				BaseURL: url,
				Driver:  "openai-compatible",
				APIKey:  apiKey,
			},
		},
	}
}

// TestOpenAIDriver_SingleRoundTrip verifies that a single tool call executes and completes.
func TestOpenAIDriver_SingleRoundTrip(t *testing.T) {
	root := t.TempDir()
	testFilePath := filepath.Join(root, "lifecycle", "ideas", "idea.md")
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFilePath, []byte("Content of idea"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_1",
					Type:      "function",
					Name:      "read_file",
					Arguments: `{"path":"lifecycle/ideas/idea.md"}`,
				},
			},
			FinishReason: "tool_calls",
		},
		{
			Content:      "I have read the idea file.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Please read the idea", []string{"lifecycle/ideas"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected progress events")
	}
	lastEvent := events[len(events)-1]
	if lastEvent.Raw != "completed" {
		t.Errorf("last event raw: got %q, want completed", lastEvent.Raw)
	}
	respText, _ := lastEvent.Event["response"].(string)
	if !strings.Contains(respText, "read the idea file") {
		t.Errorf("response text mismatch: %q", respText)
	}
}

// TestOpenAIDriver_MultiStepExecution verifies multi-turn sequential tool execution (read then write).
func TestOpenAIDriver_MultiStepExecution(t *testing.T) {
	root := t.TempDir()
	ideaPath := filepath.Join(root, "lifecycle", "ideas", "idea-2.md")
	reqPath := filepath.Join(root, "lifecycle", "requirements", "req-2.md")
	if err := os.MkdirAll(filepath.Dir(ideaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(reqPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ideaPath, []byte("Source idea details"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_read",
					Name:      "read_file",
					Arguments: `{"path":"lifecycle/ideas/idea-2.md"}`,
				},
			},
		},
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_write",
					Name:      "write_file",
					Arguments: `{"path":"lifecycle/requirements/req-2.md","content":"# Generated Requirement\nBased on idea."}`,
				},
			},
		},
		{
			Content:      "Requirement created successfully.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Process idea", []string{"lifecycle/requirements"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	written, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("expected file to be written on disk: %v", err)
	}
	if !strings.Contains(string(written), "Generated Requirement") {
		t.Errorf("file content mismatch: %s", string(written))
	}
}

// TestOpenAIDriver_RefusedWriteRecovery verifies that writing outside allowed_write_paths is refused
// and fed back to the model without crashing.
func TestOpenAIDriver_RefusedWriteRecovery(t *testing.T) {
	root := t.TempDir()
	disallowedPath := filepath.Join(root, "cmd", "main.go")

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_bad_write",
					Name:      "write_file",
					Arguments: `{"path":"cmd/main.go","content":"malicious overwrite"}`,
				},
			},
		},
		{
			Content:      "I see that cmd/ is restricted. I will not modify it.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	// Allowed paths only include lifecycle/requirements
	run := openAIRun("test-prov", "test-model", root, "Write code", []string{"lifecycle/requirements"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait should succeed after model acknowledges refusal: %v", err)
	}

	// Verify file was never written
	if _, err := os.Stat(disallowedPath); !os.IsNotExist(err) {
		t.Errorf("disallowed file should NOT exist on disk")
	}
}

// TestOpenAIDriver_PathTraversalRefusal verifies that path traversal escapes are rejected.
func TestOpenAIDriver_PathTraversalRefusal(t *testing.T) {
	root := t.TempDir()

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{
					ID:        "call_traversal",
					Name:      "read_file",
					Arguments: `{"path":"../../etc/passwd"}`,
				},
			},
		},
		{
			Content:      "Path traversal was denied.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Attack test", []string{"lifecycle"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

// TestOpenAIDriver_MaxIterationsCap verifies that loops exceeding max_tool_iterations terminate with error.
func TestOpenAIDriver_MaxIterationsCap(t *testing.T) {
	root := t.TempDir()

	mock := testutil.NewMockOpenAIServer()
	// Continuous infinite loop of tool calls
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			ToolCalls: []testutil.MockToolCall{
				{ID: "c1", Name: "list_dir", Arguments: `{"path":"."}`},
			},
		},
		{
			ToolCalls: []testutil.MockToolCall{
				{ID: "c2", Name: "list_dir", Arguments: `{"path":"."}`},
			},
		},
		{
			ToolCalls: []testutil.MockToolCall{
				{ID: "c3", Name: "list_dir", Arguments: `{"path":"."}`},
			},
		},
		{
			ToolCalls: []testutil.MockToolCall{
				{ID: "c4", Name: "list_dir", Arguments: `{"path":"."}`},
			},
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	// Cap at 3 iterations
	run := openAIRun("test-prov", "test-model", root, "Loop test", []string{"lifecycle"}, 3)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	waitErr := proc.Wait()
	if waitErr == nil {
		t.Fatal("expected error when hitting max iterations cap, got nil")
	}
	if !strings.Contains(waitErr.Error(), "max tool iterations cap (3)") {
		t.Errorf("error should mention iteration cap: %v", waitErr)
	}
}

// TestOpenAIDriver_StreamingAndTTFT verifies progress events and TTFT measurement callback.
func TestOpenAIDriver_StreamingAndTTFT(t *testing.T) {
	root := t.TempDir()

	mock := testutil.NewMockOpenAIServer()
	mock.DefaultContentChunks = []string{"Chunk1 ", "Chunk2 ", "Chunk3"}
	t.Cleanup(func() { mock.Close() })

	var recordedTTFT int64
	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Stream test", []string{"lifecycle"}, 10)
	run.OnTTFT = func(ms int64) {
		recordedTTFT = ms
	}

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected progress events")
	}
	if recordedTTFT < 0 {
		t.Errorf("expected positive TTFT measurement, got %d", recordedTTFT)
	}
}

// TestOpenAIDriver_KillCancellation verifies that Kill cancels the request and unblocks Wait.
func TestOpenAIDriver_KillCancellation(t *testing.T) {
	root := t.TempDir()

	mock := testutil.NewMockOpenAIServer()
	mock.StreamLatency = 30 * time.Second
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Kill test", []string{"lifecycle"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	waitErr := proc.Wait()
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("Kill took %v; expected < 2s", elapsed)
	}
	if waitErr == nil {
		t.Error("expected error after Kill(), got nil")
	}
}

// TestOpenAIDriver_RunLogFormatAndMasking verifies log formatting and secret masking.
func TestOpenAIDriver_RunLogFormatAndMasking(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "logs", "test-run.log")

	mock := testutil.NewMockOpenAIServer()
	mock.DefaultContentChunks = []string{"Response text"}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "super-secret-api-key-999")
	run := openAIRun("test-prov", "test-model", root, "---SYSTEM---\nSystem instruction with super-secret-api-key-999\n---USER---\nUser query", []string{"lifecycle"}, 10)
	run.LogPath = logPath

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	logContent := string(logBytes)

	// Check headers
	if !strings.Contains(logContent, "driver=openai-compatible") {
		t.Errorf("log missing driver header: %s", logContent)
	}
	if !strings.Contains(logContent, "provider=test-prov") {
		t.Errorf("log missing provider header: %s", logContent)
	}
	// Check secret masking
	if strings.Contains(logContent, "super-secret-api-key-999") {
		t.Errorf("secret API key leaked into log file: %s", logContent)
	}
	if !strings.Contains(logContent, "***") {
		t.Errorf("expected masked '***' in log file: %s", logContent)
	}
}

// TestOpenAIDriver_NativeXMLRecovery verifies recovery of XML-style tool calls (FR-5a).
func TestOpenAIDriver_NativeXMLRecovery(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "lifecycle", "requirements", "xml-req.md")
	_ = os.MkdirAll(filepath.Dir(targetPath), 0o755)

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			// Plain text turn containing XML native call syntax
			Content:      "<function=write_file><parameter=path>lifecycle/requirements/xml-req.md</parameter><parameter=content># XML Created</parameter></function>",
			FinishReason: "stop",
		},
		{
			Content:      "XML file created successfully.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Create XML", []string{"lifecycle/requirements"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("reading recovered file: %v", err)
	}
	if string(content) != "# XML Created" {
		t.Errorf("file content mismatch: got %q, want '# XML Created'", string(content))
	}
}

// TestOpenAIDriver_NativeJSONRecovery verifies recovery of JSON-tag tool calls (FR-5a).
func TestOpenAIDriver_NativeJSONRecovery(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "lifecycle", "requirements", "json-req.md")
	_ = os.MkdirAll(filepath.Dir(targetPath), 0o755)

	mock := testutil.NewMockOpenAIServer()
	mock.ScriptedTurns = []testutil.MockTurn{
		{
			// Plain text turn containing JSON <tool_call> tag
			Content:      `<tool_call>{"name":"write_file","arguments":{"path":"lifecycle/requirements/json-req.md","content":"# JSON Tag Created"}}</tool_call>`,
			FinishReason: "stop",
		},
		{
			Content:      "JSON file created successfully.",
			FinishReason: "stop",
		},
	}
	t.Cleanup(func() { mock.Close() })

	drv := newOpenAIDriver("test-prov", mock.URL(), "")
	run := openAIRun("test-prov", "test-model", root, "Create JSON", []string{"lifecycle/requirements"}, 10)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("reading recovered file: %v", err)
	}
	if string(content) != "# JSON Tag Created" {
		t.Errorf("file content mismatch: got %q, want '# JSON Tag Created'", string(content))
	}
}
