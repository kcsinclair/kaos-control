// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Package testutil provides shared test helpers for Ollama integration tests.
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

// RecordedRequest captures one inbound HTTP request for assertion.
type RecordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

// MockOllamaServer is a configurable httptest.Server that mimics the Ollama REST API.
// It is safe for concurrent use.
type MockOllamaServer struct {
	// --- configurable response state ---

	// Models returned by GET /api/tags (default: [{name:"testmodel:latest",size:1000000}]).
	Models []OllamaModel

	// ChatChunks are the NDJSON message content chunks emitted by POST /api/chat.
	ChatChunks []string

	// GenChunks are the NDJSON response chunks emitted by POST /api/generate.
	GenChunks []string

	// Latency holds per-path sleep durations injected before the response is written.
	// Keys are URL paths, e.g. "/api/chat". Use "" for all paths.
	Latency map[string]time.Duration

	// ErrorCodes maps URL path → HTTP status code to return instead of the normal response.
	// E.g. {"/api/chat": 500}.
	ErrorCodes map[string]int

	// RequireAuthToken, when non-empty, causes the server to return 401 if the
	// incoming Authorization header is not "Bearer <RequireAuthToken>".
	RequireAuthToken string

	// --- internal ---

	mu       sync.Mutex
	requests []RecordedRequest
	server   *httptest.Server
	closed   bool
}

// OllamaModel is a single entry in the GET /api/tags model list.
type OllamaModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// NewMockOllamaServer starts a new mock Ollama HTTP server on a random port.
// Call Close() when done (safe to call multiple times).
func NewMockOllamaServer() *MockOllamaServer {
	m := &MockOllamaServer{
		Models:     []OllamaModel{{Name: "testmodel:latest", Size: 1_000_000}},
		ChatChunks: []string{"chunk1", "chunk2"},
		GenChunks:  []string{"chunk1", "chunk2"},
		Latency:    make(map[string]time.Duration),
		ErrorCodes: make(map[string]int),
	}
	m.server = httptest.NewServer(m)
	return m
}

// URL returns the base URL of the mock server (e.g. "http://127.0.0.1:PORT").
func (m *MockOllamaServer) URL() string {
	return m.server.URL
}

// Requests returns a snapshot of all recorded inbound requests.
func (m *MockOllamaServer) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RecordedRequest, len(m.requests))
	copy(out, m.requests)
	return out
}

// LastRequest returns the most recent recorded request, or nil if none.
func (m *MockOllamaServer) LastRequest() *RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		return nil
	}
	r := m.requests[len(m.requests)-1]
	return &r
}

// Close shuts down the mock server. Idempotent.
func (m *MockOllamaServer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.server.Close()
		m.closed = true
	}
}

// ServeHTTP implements http.Handler.
func (m *MockOllamaServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Record the request body.
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
	errCode := m.ErrorCodes[r.URL.Path]
	if errCode == 0 {
		errCode = m.ErrorCodes[""]
	}
	requireAuth := m.RequireAuthToken
	m.mu.Unlock()

	// Auth check.
	if requireAuth != "" {
		got := r.Header.Get("Authorization")
		want := "Bearer " + requireAuth
		if got != want {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Inject latency (interruptible via request context).
	if latency > 0 {
		select {
		case <-time.After(latency):
		case <-r.Context().Done():
			return
		}
	}

	// Configurable error override.
	if errCode != 0 {
		w.WriteHeader(errCode)
		fmt.Fprintf(w, `{"error":"mock error %d"}`, errCode)
		return
	}

	switch r.URL.Path {
	case "/api/tags":
		m.handleTags(w, r)
	case "/api/chat":
		m.handleChat(w, r)
	case "/api/generate":
		m.handleGenerate(w, r)
	default:
		// Health / root.
		w.WriteHeader(http.StatusOK)
	}
}

func (m *MockOllamaServer) handleTags(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	models := make([]OllamaModel, len(m.Models))
	copy(models, m.Models)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"models": models})
}

func (m *MockOllamaServer) handleChat(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	chunks := make([]string, len(m.ChatChunks))
	copy(chunks, m.ChatChunks)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/x-ndjson")
	flusher, _ := w.(http.Flusher)

	for _, chunk := range chunks {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		line := fmt.Sprintf(`{"message":{"role":"assistant","content":%q}}`, chunk)
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}
	// Terminal done line.
	fmt.Fprintln(w, `{"done":true,"done_reason":"stop"}`)
	if flusher != nil {
		flusher.Flush()
	}
}

func (m *MockOllamaServer) handleGenerate(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	chunks := make([]string, len(m.GenChunks))
	copy(chunks, m.GenChunks)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/x-ndjson")
	flusher, _ := w.(http.Flusher)

	for _, chunk := range chunks {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		line := fmt.Sprintf(`{"response":%q}`, chunk)
		fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}
	fmt.Fprintln(w, `{"done":true}`)
	if flusher != nil {
		flusher.Flush()
	}
}

// BodyAsMap parses the JSON body of a recorded request into a map.
// Returns nil if body is empty or not valid JSON.
func (rr *RecordedRequest) BodyAsMap() map[string]any {
	if len(rr.Body) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body, &m); err != nil {
		return nil
	}
	return m
}

// RequestsForPath returns all recorded requests whose path matches the given prefix.
func (m *MockOllamaServer) RequestsForPath(path string) []RecordedRequest {
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
