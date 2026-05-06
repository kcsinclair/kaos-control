//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// ── Milestone 4 — OllamaDriver Unit Tests ────────────────────────────────────

// ollamaRun builds a minimal Run for driver tests.
func ollamaRun(instanceName, model, endpoint, prompt string) agent.Run {
	return agent.Run{
		RunID:              "drv-test-run",
		AgentName:          "test-agent",
		Role:               "analyst",
		Model:              model,
		PromptText:         prompt,
		OllamaInstanceName: instanceName,
		OllamaEndpoint:     endpoint,
		TimeoutMinutes:     1,
	}
}

// newOllamaDriver builds an OllamaDriver with a single instance pointing at url.
func newOllamaDriver(instanceName, url string) *agent.OllamaDriver {
	return &agent.OllamaDriver{
		Instances: []config.OllamaInstance{
			{Name: instanceName, BaseURL: url},
		},
	}
}

// collectEvents reads all events from the process progress channel until it is
// closed, with a per-event timeout to avoid hanging tests.
func collectEvents(t *testing.T, proc agent.Process, timeout time.Duration) []agent.ProgressEvent {
	t.Helper()
	var events []agent.ProgressEvent
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case ev, ok := <-proc.Progress():
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline.C:
			t.Errorf("timed out collecting events after %v; got %d so far", timeout, len(events))
			return events
		}
	}
}

// TestOllamaDriver_SuccessfulChatRun verifies that a chat run emits "started",
// content chunk events, and a "completed" event in order.
func TestOllamaDriver_SuccessfulChatRun(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	mock.ChatChunks = []string{"hello", " world"}
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Say hello")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}

	// Verify order: first event is "started".
	if len(events) == 0 {
		t.Fatal("no events received")
	}
	if events[0].Raw != "started" {
		t.Errorf("first event: got %q, want %q", events[0].Raw, "started")
	}

	// Last event must be "completed" with a response field.
	last := events[len(events)-1]
	if last.Raw != "completed" {
		t.Errorf("last event raw: got %q, want %q", last.Raw, "completed")
	}
	if last.Event["type"] != "completed" {
		t.Errorf("last event type: %v", last.Event)
	}
	fullResponse, _ := last.Event["response"].(string)
	if !strings.Contains(fullResponse, "hello") {
		t.Errorf("completed response should contain 'hello', got %q", fullResponse)
	}
	if !strings.Contains(fullResponse, " world") {
		t.Errorf("completed response should contain ' world', got %q", fullResponse)
	}
}

// TestOllamaDriver_SuccessfulGenerateRun verifies a generate endpoint run emits
// events in the same order as chat.
func TestOllamaDriver_SuccessfulGenerateRun(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	mock.GenChunks = []string{"gen1", "gen2"}
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "generate", "Generate text")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	events := collectEvents(t, proc, 5*time.Second)
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("no events received")
	}
	if events[0].Raw != "started" {
		t.Errorf("first event: got %q, want %q", events[0].Raw, "started")
	}
	last := events[len(events)-1]
	if last.Raw != "completed" {
		t.Errorf("last event raw: got %q, want %q", last.Raw, "completed")
	}
	fullResponse, _ := last.Event["response"].(string)
	if !strings.Contains(fullResponse, "gen1") || !strings.Contains(fullResponse, "gen2") {
		t.Errorf("generate response: got %q, want gen1+gen2", fullResponse)
	}
}

// TestOllamaDriver_SystemPromptSeparation verifies that a prompt with
// ---SYSTEM--- / ---USER--- delimiters sends system and user messages separately
// in the request body.
func TestOllamaDriver_SystemPromptSeparation(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	prompt := "---SYSTEM---\nYou are a helpful assistant.\n---USER---\nWhat is Go?"
	run := ollamaRun("test", "testmodel:latest", "chat", prompt)

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	collectEvents(t, proc, 5*time.Second)
	proc.Wait()

	reqs := mock.RequestsForPath("/api/chat")
	if len(reqs) == 0 {
		t.Fatal("no requests recorded by mock")
	}
	body := reqs[len(reqs)-1].BodyAsMap()
	if body == nil {
		t.Fatal("request body is empty or not valid JSON")
	}

	messages, _ := body["messages"].([]any)
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages (system + user), got %d", len(messages))
	}

	sysMsg, _ := messages[0].(map[string]any)
	if role, _ := sysMsg["role"].(string); role != "system" {
		t.Errorf("messages[0].role: got %q, want %q", role, "system")
	}
	if content, _ := sysMsg["content"].(string); !strings.Contains(content, "helpful assistant") {
		t.Errorf("system message content: got %q", content)
	}

	userMsg, _ := messages[1].(map[string]any)
	if role, _ := userMsg["role"].(string); role != "user" {
		t.Errorf("messages[1].role: got %q, want %q", role, "user")
	}
	if content, _ := userMsg["content"].(string); !strings.Contains(content, "What is Go") {
		t.Errorf("user message content: got %q", content)
	}
}

// TestOllamaDriver_NoSystemPrompt verifies that a prompt without the delimiter
// is sent as a single user message.
func TestOllamaDriver_NoSystemPrompt(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Just a plain user prompt")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	collectEvents(t, proc, 5*time.Second)
	proc.Wait()

	reqs := mock.RequestsForPath("/api/chat")
	if len(reqs) == 0 {
		t.Fatal("no /api/chat requests recorded")
	}
	body := reqs[len(reqs)-1].BodyAsMap()
	messages, _ := body["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message (user only), got %d", len(messages))
	}
	msg, _ := messages[0].(map[string]any)
	if role, _ := msg["role"].(string); role != "user" {
		t.Errorf("messages[0].role: got %q, want %q", role, "user")
	}
}

// TestOllamaDriver_WaitBlocks verifies that Wait() blocks until the stream completes.
func TestOllamaDriver_WaitBlocks(t *testing.T) {
	// Use a slow mock so we can verify Wait blocks.
	mock := testutil.NewMockOllamaServer()
	mock.Latency["/api/chat"] = 200 * time.Millisecond
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Block test")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Drain progress channel in background.
	go collectEvents(t, proc, 10*time.Second)

	start := time.Now()
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait returned unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// Wait() must have taken at least 200ms (the mock latency).
	if elapsed < 150*time.Millisecond {
		t.Errorf("Wait() returned too quickly (%v); expected at least 150ms", elapsed)
	}
}

// TestOllamaDriver_KillCancels verifies that Kill() mid-stream cancels the HTTP
// request and Wait() returns a context-related error.
func TestOllamaDriver_KillCancels(t *testing.T) {
	// Use a mock that never finishes (very long latency before first byte).
	mock := testutil.NewMockOllamaServer()
	mock.Latency["/api/chat"] = 30 * time.Second
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Kill test")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Drain progress in background.
	go collectEvents(t, proc, 35*time.Second)

	// Give the driver a moment to make the HTTP request.
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	waitErr := proc.Wait()
	elapsed := time.Since(start)

	// Kill should unblock Wait within 2 seconds.
	if elapsed > 2*time.Second {
		t.Errorf("Kill + Wait took %v; expected under 2s", elapsed)
	}

	// Wait must return an error (context cancelled).
	if waitErr == nil {
		t.Error("Wait() should return an error after Kill(), got nil")
	}
}

// TestOllamaDriver_HTTPError verifies that an HTTP 500 from Ollama causes the
// driver to emit an error and StderrTail() contains useful text.
func TestOllamaDriver_HTTPError(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	mock.ErrorCodes["/api/chat"] = 500
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Error test")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	waitErr := proc.Wait()

	if waitErr == nil {
		t.Error("expected Wait() to return an error for HTTP 500, got nil")
	}

	stderr := proc.StderrTail()
	if !strings.Contains(stderr, "500") {
		t.Errorf("StderrTail should mention '500', got: %q", stderr)
	}
}

// TestOllamaDriver_ConnectionRefused verifies that when the Ollama instance is
// unreachable, the driver fails fast with a clear error.
func TestOllamaDriver_ConnectionRefused(t *testing.T) {
	drv := &agent.OllamaDriver{
		Instances: []config.OllamaInstance{
			{Name: "unreachable", BaseURL: "http://127.0.0.1:19876"}, // nothing listening
		},
	}
	run := ollamaRun("unreachable", "testmodel:latest", "chat", "Connection test")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v (should succeed, error comes from Wait)", err)
	}

	collectEvents(t, proc, 5*time.Second)
	waitErr := proc.Wait()

	if waitErr == nil {
		t.Error("expected error from Wait() for connection-refused instance")
	}

	stderr := proc.StderrTail()
	if stderr == "" {
		t.Error("StderrTail should be non-empty for connection error")
	}
}

// TestOllamaDriver_Timeout verifies that context cancellation (simulating a
// timeout) causes the driver to fail and Wait() to return an error.
func TestOllamaDriver_Timeout(t *testing.T) {
	// Mock that delays longer than our context deadline.
	mock := testutil.NewMockOllamaServer()
	mock.Latency["/api/chat"] = 10 * time.Second
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "chat", "Timeout test")

	// Inject a short-lived context to simulate the timeout.
	ctx, ctxCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer ctxCancel()

	proc, err := drv.Start(ctx, run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	collectEvents(t, proc, 5*time.Second)
	waitErr := proc.Wait()

	if waitErr == nil {
		t.Error("expected error from Wait() after context timeout")
	}
}

// TestOllamaDriver_ModelFieldForwarded verifies that the model field in the
// request body matches Run.Model.
func TestOllamaDriver_ModelFieldForwarded(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	const wantModel = "llama3:8b"
	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", wantModel, "chat", "Model test")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	collectEvents(t, proc, 5*time.Second)
	proc.Wait()

	reqs := mock.RequestsForPath("/api/chat")
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	body := reqs[len(reqs)-1].BodyAsMap()
	if body == nil {
		t.Fatal("empty request body")
	}
	gotModel, _ := body["model"].(string)
	if gotModel != wantModel {
		t.Errorf("model field: got %q, want %q", gotModel, wantModel)
	}
}

// TestOllamaDriver_InstanceNotFound verifies that Start() returns an error
// immediately when the instance name doesn't match any configured instance.
func TestOllamaDriver_InstanceNotFound(t *testing.T) {
	drv := &agent.OllamaDriver{
		Instances: []config.OllamaInstance{
			{Name: "real-instance", BaseURL: "http://localhost:11434"},
		},
	}
	run := ollamaRun("nonexistent", "testmodel:latest", "chat", "test")

	_, err := drv.Start(context.Background(), run)
	if err == nil {
		t.Fatal("expected Start() to return error for unknown instance, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention instance name 'nonexistent': %v", err)
	}
}

// TestOllamaDriver_StreamFieldParsing verifies that generate endpoint chunks
// accumulate the "response" JSON field correctly into the full response text.
func TestOllamaDriver_StreamFieldParsing(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	mock.GenChunks = []string{"The ", "answer ", "is 42"}
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "testmodel:latest", "generate", "What is the answer?")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	events := collectEvents(t, proc, 5*time.Second)
	proc.Wait()

	// Find the completed event.
	var completedEvent *agent.ProgressEvent
	for i := range events {
		if events[i].Raw == "completed" {
			completedEvent = &events[i]
			break
		}
	}
	if completedEvent == nil {
		t.Fatal("no 'completed' event received")
	}

	fullResponse, _ := completedEvent.Event["response"].(string)
	want := "The answer is 42"
	if fullResponse != want {
		t.Errorf("accumulated response: got %q, want %q", fullResponse, want)
	}
}

// TestOllamaDriver_StreamRequestBody_Generate verifies that generate endpoint
// request body has the correct structure (prompt, model, stream:true).
func TestOllamaDriver_StreamRequestBody_Generate(t *testing.T) {
	mock := testutil.NewMockOllamaServer()
	t.Cleanup(func() { mock.Close() })

	drv := newOllamaDriver("test", mock.URL())
	run := ollamaRun("test", "phi3:mini", "generate", "Prompt text")

	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	collectEvents(t, proc, 5*time.Second)
	proc.Wait()

	reqs := mock.RequestsForPath("/api/generate")
	if len(reqs) == 0 {
		t.Fatal("no /api/generate requests recorded")
	}
	body := reqs[len(reqs)-1].BodyAsMap()

	if model, _ := body["model"].(string); model != "phi3:mini" {
		t.Errorf("model: got %q, want %q", model, "phi3:mini")
	}
	if prompt, _ := body["prompt"].(string); prompt != "Prompt text" {
		t.Errorf("prompt: got %q, want %q", prompt, "Prompt text")
	}
	streamRaw, _ := json.Marshal(body["stream"])
	if string(streamRaw) != "true" {
		t.Errorf("stream: got %s, want true", streamRaw)
	}
}
