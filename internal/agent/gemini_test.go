// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFormatGeminiSummary verifies that formatting Gemini token and status
// metadata works with complete and partial usage fields.
func TestFormatGeminiSummary(t *testing.T) {
	t.Run("All fields", func(t *testing.T) {
		chunk := &geminiResponseChunk{
			Candidates: []geminiCandidate{
				{FinishReason: "STOP"},
			},
			UsageMetadata: geminiUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
		}
		got := formatGeminiSummary(chunk)
		wantContains := []string{
			"# summary:",
			"finish_reason=STOP",
			"prompt_tokens=10",
			"candidates_tokens=20",
			"total_tokens=30",
		}
		for _, w := range wantContains {
			if !strings.Contains(got, w) {
				t.Errorf("summary missing %q, got: %q", w, got)
			}
		}
	})

	t.Run("Partial fields", func(t *testing.T) {
		chunk := &geminiResponseChunk{
			UsageMetadata: geminiUsageMetadata{
				TotalTokenCount: 15,
			},
		}
		got := formatGeminiSummary(chunk)
		if !strings.Contains(got, "total_tokens=15") {
			t.Errorf("expected total_tokens=15, got: %q", got)
		}
		if strings.Contains(got, "prompt_tokens") {
			t.Errorf("unexpected prompt_tokens in partial summary: %q", got)
		}
	})
}

// TestGeminiDriver_Start_Success simulates a successful chunked JSON stream
// from the Gemini API and verifies the progress output events and logs.
func TestGeminiDriver_Start_Success(t *testing.T) {
	// Start a mock HTTP server to stream chunked JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// The Gemini API streams Candidate response chunks as elements of a JSON array.
		// Start with an opening square bracket.
		_, _ = w.Write([]byte("[\n"))

		// Write first chunk.
		chunk1 := `{"candidates": [{"content": {"parts": [{"text": "Hello "}]}}], "usageMetadata": {"promptTokenCount": 5}}`
		_, _ = w.Write([]byte(chunk1 + ",\n"))

		// Write second/final chunk.
		chunk2 := `{"candidates": [{"content": {"parts": [{"text": "world!"}]}}], "usageMetadata": {"promptTokenCount": 5, "candidatesTokenCount": 2, "totalTokenCount": 7}}`
		_, _ = w.Write([]byte(chunk2 + "\n"))

		// End the JSON array stream.
		_, _ = w.Write([]byte("]"))
	}))
	defer server.Close()

	// Configure environment variables for the test.
	t.Setenv("GEMINI_API_KEY", "dummy-key-for-test")
	t.Setenv("GEMINI_BASE_URL", server.URL)

	// Prepare run args and temp log directory.
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_run.log")

	run := Run{
		RunID:          "test-run-123",
		AgentName:      "test-gemini-agent",
		Role:           "test-role",
		Model:          "gemini-2.5-flash",
		PromptText:     "---SYSTEM---\nYou are a helpful assistant.\n---USER---\nSay hello world",
		LogPath:        logPath,
		TimeoutMinutes: 1,
	}

	driver := &GeminiDriver{
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	proc, err := driver.Start(ctx, run)
	if err != nil {
		t.Fatalf("driver.Start failed: %v", err)
	}

	// Consume and record the stream events.
	var outputs []string
	var completedResponse string

	for ev := range proc.Progress() {
		if ev.Event != nil {
			if ev.Event["type"] == "output" {
				if txt, ok := ev.Event["text"].(string); ok {
					outputs = append(outputs, txt)
				}
			} else if ev.Event["type"] == "completed" {
				if resp, ok := ev.Event["response"].(string); ok {
					completedResponse = resp
				}
			}
		}
	}

	// Wait for process completion.
	if err := proc.Wait(); err != nil {
		t.Fatalf("proc.Wait returned error: %v", err)
	}

	// Verify events.
	expectedText := "Hello world!"
	combinedOutputs := strings.Join(outputs, "")
	if combinedOutputs != expectedText {
		t.Errorf("combined progress output %q, expected %q", combinedOutputs, expectedText)
	}
	if completedResponse != expectedText {
		t.Errorf("completed response %q, expected %q", completedResponse, expectedText)
	}

	// Verify log file contents.
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logContent := string(logBytes)

	expectedLogSnippets := []string{
		"# kaos-control agent run test-run-123",
		"# agent=test-gemini-agent role=test-role driver=gemini model=gemini-2.5-flash",
		"# system_prompt:",
		"You are a helpful assistant.",
		"# user_prompt:",
		"Say hello world",
		"# event: started",
		`"promptTokenCount":5`,
		`"totalTokenCount":7`,
		"# event: completed",
		"Hello world!",
		"# summary: prompt_tokens=5 candidates_tokens=2 total_tokens=7",
		"# finished=",
	}

	for _, snippet := range expectedLogSnippets {
		if !strings.Contains(logContent, snippet) {
			t.Errorf("log file missing expected snippet: %q", snippet)
		}
	}
}

// TestGeminiDriver_Start_HTTPError verifies that non-OK HTTP responses from
// the Gemini API are handled gracefully and return structured error messages.
func TestGeminiDriver_Start_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"code": 400, "message": "API key not valid", "status": "INVALID_ARGUMENT"}}`))
	}))
	defer server.Close()

	t.Setenv("GEMINI_API_KEY", "invalid-key")
	t.Setenv("GEMINI_BASE_URL", server.URL)

	run := Run{
		RunID:      "test-run-error",
		AgentName:  "test-gemini-agent",
		Role:       "test-role",
		Model:      "gemini-2.5-flash",
		PromptText: "Hello",
	}

	driver := &GeminiDriver{
		HTTPClient: server.Client(),
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start failed to initialize: %v", err)
	}

	// Consume channel to allow background routine to progress.
	for range proc.Progress() {
	}

	err = proc.Wait()
	if err == nil {
		t.Fatal("expected proc.Wait to return HTTP error, got nil")
	}

	expectedErrorSnippet := "gemini returned HTTP 400"
	if !strings.Contains(err.Error(), expectedErrorSnippet) {
		t.Errorf("expected error to contain %q, got: %v", expectedErrorSnippet, err)
	}

	tail := proc.StderrTail()
	if !strings.Contains(tail, expectedErrorSnippet) {
		t.Errorf("expected StderrTail to contain %q, got: %q", expectedErrorSnippet, tail)
	}
}

// TestGeminiDriver_Start_MalformedJSON verifies that malformed JSON streamed
// by the API returns a parsing error.
func TestGeminiDriver_Start_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[\n {"invalid_json": `)) // Malformed JSON array item
	}))
	defer server.Close()

	t.Setenv("GEMINI_API_KEY", "dummy-key")
	t.Setenv("GEMINI_BASE_URL", server.URL)

	run := Run{
		RunID:      "test-run-malformed",
		AgentName:  "test-gemini-agent",
		Role:       "test-role",
		Model:      "gemini-2.5-flash",
		PromptText: "Hello",
	}

	driver := &GeminiDriver{
		HTTPClient: server.Client(),
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start failed to initialize: %v", err)
	}

	for range proc.Progress() {
	}

	err = proc.Wait()
	if err == nil {
		t.Fatal("expected proc.Wait to fail on malformed JSON, got nil")
	}
}

// TestGeminiDriver_MissingApiKey verifies that an error is returned immediately
// when the GEMINI_API_KEY environment variable is not configured.
func TestGeminiDriver_MissingApiKey(t *testing.T) {
	os.Unsetenv("GEMINI_API_KEY")

	driver := &GeminiDriver{}
	_, err := driver.Start(context.Background(), Run{PromptText: "Hello"})
	if err == nil {
		t.Fatal("expected error when GEMINI_API_KEY is not set, got nil")
	}
	expected := "GEMINI_API_KEY environment variable is not set"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}
