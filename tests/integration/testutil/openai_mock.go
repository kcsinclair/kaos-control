// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Package testutil provides shared test helpers for OpenAI-compatible integration tests.
package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// OpenAIModel represents a model entry returned by /v1/models.
type OpenAIModel struct {
	ID                  string   `json:"id"`
	Object              string   `json:"object"`
	Created             int64    `json:"created"`
	OwnedBy             string   `json:"owned_by"`
	SupportedParameters []string `json:"supported_parameters,omitempty"`
}

// MockTurn defines a scripted turn for the mock OpenAI completions endpoint.
type MockTurn struct {
	Content      string
	ToolCalls    []MockToolCall
	FinishReason string
}

// MockToolCall defines a tool call emitted by the mock server.
type MockToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// MockOpenAIServer is a configurable httptest.Server mimicking an OpenAI-compatible endpoint.
type MockOpenAIServer struct {
	mu sync.Mutex

	// Models returned by GET /v1/models.
	Models []OpenAIModel

	// PreflightMode controls preflight capability response:
	// "mode-c" (default): token delta passes (prompt_tokens: 25 with tools, 5 without)
	// "mode-a": silent drop (prompt_tokens: 5 both with and without tools)
	// "mode-b": explicit HTTP 400 rejection ("<model> does not support tools")
	PreflightMode string

	// ScriptedTurns defines multi-turn sequences for chat completions.
	// When empty, DefaultContentChunks will be streamed as plain content.
	ScriptedTurns []MockTurn
	turnIndex     int

	// DefaultContentChunks are streamed when ScriptedTurns is empty.
	DefaultContentChunks []string

	// Latency per path or "" for all.
	Latency map[string]time.Duration

	// StreamLatency is injected during streaming chat completions.
	StreamLatency time.Duration

	// ErrorCodes per path. E.g. {"/v1/chat/completions": 500}.
	ErrorCodes map[string]int

	// RequireAuthToken validates Authorization: Bearer <token>.
	RequireAuthToken string

	// RequireHeader validates an arbitrary header key-value pair if set.
	RequireHeader map[string]string

	requests []RecordedRequest
	server   *httptest.Server
	closed   bool
}

// NewMockOpenAIServer starts a new mock OpenAI-compatible server.
func NewMockOpenAIServer() *MockOpenAIServer {
	m := &MockOpenAIServer{
		Models: []OpenAIModel{
			{ID: "test-model", Object: "model", Created: time.Now().Unix(), OwnedBy: "test", SupportedParameters: []string{"tools"}},
		},
		PreflightMode:        "mode-c",
		DefaultContentChunks: []string{"Hello ", "world!"},
		Latency:              make(map[string]time.Duration),
		ErrorCodes:           make(map[string]int),
		RequireHeader:        make(map[string]string),
	}
	m.server = httptest.NewServer(m)
	return m
}

// URL returns the mock server's base URL.
func (m *MockOpenAIServer) URL() string {
	return m.server.URL
}

// Requests returns a copy of all recorded requests.
func (m *MockOpenAIServer) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RecordedRequest, len(m.requests))
	copy(out, m.requests)
	return out
}

// LastRequest returns the most recent request.
func (m *MockOpenAIServer) LastRequest() *RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		return nil
	}
	r := m.requests[len(m.requests)-1]
	return &r
}

// RequestsForPath returns requests matching the given path prefix.
func (m *MockOpenAIServer) RequestsForPath(path string) []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []RecordedRequest
	for _, r := range m.requests {
		if strings.HasPrefix(r.Path, path) {
			out = append(out, r)
		}
	}
	return out
}

// ResetTurns resets the turn sequence counter.
func (m *MockOpenAIServer) ResetTurns() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.turnIndex = 0
}

// Close shuts down the server.
func (m *MockOpenAIServer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.server.Close()
		m.closed = true
	}
}

// ServeHTTP implements http.Handler.
func (m *MockOpenAIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	m.mu.Lock()
	m.requests = append(m.requests, RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header.Clone(),
		Body:    body,
	})
	latency := m.Latency[r.URL.Path]
	if latency == 0 {
		latency = m.Latency[""]
	}
	streamLatency := m.StreamLatency
	errCode := m.ErrorCodes[r.URL.Path]
	if errCode == 0 {
		errCode = m.ErrorCodes[""]
	}
	requireAuth := m.RequireAuthToken
	requireHeaders := make(map[string]string)
	for k, v := range m.RequireHeader {
		requireHeaders[k] = v
	}
	m.mu.Unlock()

	// Header checks
	if requireAuth != "" {
		got := r.Header.Get("Authorization")
		want := "Bearer " + requireAuth
		if got != want {
			http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}
	}
	for k, v := range requireHeaders {
		if r.Header.Get(k) != v {
			http.Error(w, fmt.Sprintf(`{"error":{"message":"missing header %s: %s"}}`, k, v), http.StatusBadRequest)
			return
		}
	}

	// Latency for non-completions
	if latency > 0 && r.URL.Path != "/v1/chat/completions" {
		select {
		case <-time.After(latency):
		case <-r.Context().Done():
			return
		}
	}

	// Error override for non-completions
	if errCode != 0 && r.URL.Path != "/v1/chat/completions" {
		w.WriteHeader(errCode)
		fmt.Fprintf(w, `{"error":{"message":"mock error %d"}}`, errCode)
		return
	}

	switch r.URL.Path {
	case "/v1/models":
		m.handleModels(w, r)
	case "/v1/chat/completions":
		m.handleChatCompletions(w, r, body, streamLatency, errCode)
	default:
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok":true}`)
	}
}

func (m *MockOpenAIServer) handleModels(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	models := make([]OpenAIModel, len(m.Models))
	copy(models, m.Models)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   models,
	})
}

func (m *MockOpenAIServer) handleChatCompletions(w http.ResponseWriter, r *http.Request, body []byte, streamLatency time.Duration, errCode int) {
	var reqMap map[string]any
	_ = json.Unmarshal(body, &reqMap)

	isStream := false
	if reqMap != nil {
		if s, ok := reqMap["stream"].(bool); ok {
			isStream = s
		}
	}

	m.mu.Lock()
	preflightMode := m.PreflightMode
	m.mu.Unlock()

	if !isStream {
		// Non-streaming / preflight request
		if preflightMode == "mode-b" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":{"message":"%v does not support tools","type":"invalid_request_error"}}`, reqMap["model"])
			return
		}

		if errCode == 400 || errCode == 500 {
			w.WriteHeader(errCode)
			fmt.Fprintf(w, `{"error":{"message":"mock error %d"}}`, errCode)
			return
		}

		hasTools := false
		if reqMap != nil {
			if tools, ok := reqMap["tools"].([]any); ok && len(tools) > 0 {
				hasTools = true
			}
		}

		promptTokens := 5
		if hasTools && preflightMode == "mode-c" {
			promptTokens = 25
		}

		resp := map[string]any{
			"id":      "chatcmpl-preflight",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"choices": []any{
				map[string]any{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "preflight-ok",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": 1,
				"total_tokens":      promptTokens + 1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Streaming completion
	if streamLatency > 0 {
		select {
		case <-time.After(streamLatency):
		case <-r.Context().Done():
			return
		}
	}

	if errCode != 0 {
		w.WriteHeader(errCode)
		fmt.Fprintf(w, `{"error":{"message":"mock error %d"}}`, errCode)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	m.mu.Lock()
	var currentTurn MockTurn
	hasScriptedTurn := false
	if len(m.ScriptedTurns) > 0 && m.turnIndex < len(m.ScriptedTurns) {
		currentTurn = m.ScriptedTurns[m.turnIndex]
		m.turnIndex++
		hasScriptedTurn = true
	}
	defaultChunks := make([]string, len(m.DefaultContentChunks))
	copy(defaultChunks, m.DefaultContentChunks)
	m.mu.Unlock()

	if hasScriptedTurn {
		// Emit scripted turn content or tool calls
		if len(currentTurn.ToolCalls) > 0 {
			for i, tc := range currentTurn.ToolCalls {
				tcID := tc.ID
				if tcID == "" {
					tcID = fmt.Sprintf("call_%d", i+1)
				}
				tcType := tc.Type
				if tcType == "" {
					tcType = "function"
				}
				chunkObj := map[string]any{
					"id": "chatcmpl-stream",
					"choices": []any{
						map[string]any{
							"index": 0,
							"delta": map[string]any{
								"role": "assistant",
								"tool_calls": []any{
									map[string]any{
										"index": i,
										"id":    tcID,
										"type":  tcType,
										"function": map[string]any{
											"name":      tc.Name,
											"arguments": tc.Arguments,
										},
									},
								},
							},
						},
					},
				}
				data, _ := json.Marshal(chunkObj)
				fmt.Fprintf(w, "data: %s\n\n", data)
				if flusher != nil {
					flusher.Flush()
				}
			}
		}

		if currentTurn.Content != "" {
			chunkObj := map[string]any{
				"id": "chatcmpl-stream",
				"choices": []any{
					map[string]any{
						"index": 0,
						"delta": map[string]any{
							"role":    "assistant",
							"content": currentTurn.Content,
						},
					},
				},
			}
			data, _ := json.Marshal(chunkObj)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
		}

		finish := currentTurn.FinishReason
		if finish == "" {
			if len(currentTurn.ToolCalls) > 0 {
				finish = "tool_calls"
			} else {
				finish = "stop"
			}
		}

		termObj := map[string]any{
			"id": "chatcmpl-stream",
			"choices": []any{
				map[string]any{
					"index":         0,
					"delta":         map[string]any{},
					"finish_reason": finish,
				},
			},
		}
		termData, _ := json.Marshal(termObj)
		fmt.Fprintf(w, "data: %s\n\n", termData)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		return
	}

	// Default plain chunks
	for _, chunk := range defaultChunks {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		chunkObj := map[string]any{
			"id": "chatcmpl-stream",
			"choices": []any{
				map[string]any{
					"index": 0,
					"delta": map[string]any{
						"role":    "assistant",
						"content": chunk,
					},
				},
			},
		}
		data, _ := json.Marshal(chunkObj)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
	}

	termObj := map[string]any{
		"id": "chatcmpl-stream",
		"choices": []any{
			map[string]any{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": "stop",
			},
		},
	}
	termData, _ := json.Marshal(termObj)
	fmt.Fprintf(w, "data: %s\n\n", termData)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}
