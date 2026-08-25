// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

func TestClassifyHTTPFailure(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantKind   RateLimitKind
		wantOK     bool
	}{
		{"connection error (no status)", 0, RateLimitKindUnreachable, true},
		{"529 overloaded", 529, RateLimitKindOverloaded, true},
		{"429 rate limit", 429, RateLimitKindRateLimit, true},
		{"502 bad gateway", 502, RateLimitKindUnreachable, true},
		{"503 service unavailable", 503, RateLimitKindUnreachable, true},
		{"504 gateway timeout", 504, RateLimitKindUnreachable, true},
		{"400 bad request unclassified", 400, "", false},
		{"401 unauthorized unclassified", 401, "", false},
		{"404 not found unclassified", 404, "", false},
		{"500 internal server error unclassified", 500, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := classifyHTTPFailure(tc.statusCode)
			if ok != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if kind != tc.wantKind {
				t.Errorf("kind: got %q, want %q", kind, tc.wantKind)
			}
		})
	}
}

func TestWrapHTTPError(t *testing.T) {
	base := errors.New("boom")

	wrapped := wrapHTTPError(base, 529)
	var rerr *RunError
	if !errors.As(wrapped, &rerr) {
		t.Fatalf("expected *RunError for a classifiable status, got %T", wrapped)
	}
	if rerr.Kind != RateLimitKindOverloaded {
		t.Errorf("kind: got %q, want overloaded", rerr.Kind)
	}
	if !errors.Is(wrapped, base) {
		t.Error("expected wrapped error to unwrap to the original error")
	}

	unclassified := wrapHTTPError(base, 401)
	if errors.As(unclassified, &rerr) {
		t.Errorf("expected an unclassifiable status to pass through unwrapped, got %v", unclassified)
	}
	if unclassified != base {
		t.Errorf("expected the original error unchanged, got %v", unclassified)
	}
}

// openAIPreflightOKHandler returns an http.HandlerFunc that satisfies
// VerifyToolCapability's two non-streaming preflight requests (with/without
// tools) before delegating to next for the streaming completion request.
func openAIPreflightOKHandler(t *testing.T, next http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if stream, _ := req["stream"].(bool); !stream {
			promptTokens := 10
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 50
			}
			resp := openAIChatCompletionResponse{Usage: openAIUsage{PromptTokens: promptTokens}}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		next(w, r)
	}
}

func TestOpenAICompatibleDriver_HTTPStatusSurfacesAsClassifiedRunError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantKind RateLimitKind
	}{
		{"529 overloaded", 529, RateLimitKindOverloaded},
		{"429 rate limit", 429, RateLimitKindRateLimit},
		{"503 gateway unreachable", 503, RateLimitKindUnreachable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(openAIPreflightOKHandler(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
			}))
			defer ts.Close()

			drv := &OpenAICompatibleDriver{Providers: []config.Provider{{Name: "p", BaseURL: ts.URL, Driver: "openai-compatible"}}}
			run := Run{
				RunID:          "r1",
				ProviderName:   "p",
				Model:          "m",
				PromptText:     "do the thing",
				ProjectRoot:    t.TempDir(),
				TimeoutMinutes: 1,
			}
			proc, err := drv.Start(context.Background(), run)
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			waitErr := proc.Wait()
			if waitErr == nil {
				t.Fatal("expected an error")
			}
			var rerr *RunError
			if !errors.As(waitErr, &rerr) {
				t.Fatalf("expected *RunError, got %T: %v", waitErr, waitErr)
			}
			if rerr.Kind != tc.wantKind {
				t.Errorf("kind: got %q, want %q", rerr.Kind, tc.wantKind)
			}
		})
	}
}

// TestOpenAICompatibleDriver_MidStreamConnectionDropSurfacesAsUnreachable
// covers the "provider was reachable for preflight but drops the actual
// completion connection" case: the server accepts the streaming request's
// TCP connection then hangs up before writing any HTTP response, which
// deterministically produces a transport-level (statusCode==0) error from
// client.Do — the same failure shape as connection refused / DNS failure /
// TLS handshake failure, all classified as RateLimitKindUnreachable.
func TestOpenAICompatibleDriver_MidStreamConnectionDropSurfacesAsUnreachable(t *testing.T) {
	ts := httptest.NewServer(openAIPreflightOKHandler(t, func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("expected ResponseWriter to support Hijack")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		conn.Close()
	}))
	defer ts.Close()

	drv := &OpenAICompatibleDriver{Providers: []config.Provider{{Name: "p", BaseURL: ts.URL, Driver: "openai-compatible"}}}
	run := Run{
		RunID:          "r1",
		ProviderName:   "p",
		Model:          "m",
		PromptText:     "do the thing",
		ProjectRoot:    t.TempDir(),
		TimeoutMinutes: 1,
	}
	proc, err := drv.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitErr := proc.Wait()
	if waitErr == nil {
		t.Fatal("expected an error")
	}
	var rerr *RunError
	if !errors.As(waitErr, &rerr) {
		t.Fatalf("expected *RunError, got %T: %v", waitErr, waitErr)
	}
	if rerr.Kind != RateLimitKindUnreachable {
		t.Errorf("kind: got %q, want unreachable", rerr.Kind)
	}
}

func TestProbeProviderHealth(t *testing.T) {
	t.Run("2xx/4xx response counts as healthy", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/models" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusUnauthorized) // reachable, just unauthenticated
		}))
		defer ts.Close()

		healthy := ProbeProviderHealth(context.Background(), nil, config.Provider{Name: "p", BaseURL: ts.URL}, time.Second)
		if !healthy {
			t.Error("expected a reachable server (even with a 401) to be considered healthy")
		}
	})

	t.Run("5xx response counts as unhealthy", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer ts.Close()

		healthy := ProbeProviderHealth(context.Background(), nil, config.Provider{Name: "p", BaseURL: ts.URL}, time.Second)
		if healthy {
			t.Error("expected a 503 response to be considered unhealthy")
		}
	})

	t.Run("connection failure counts as unhealthy", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		badURL := ts.URL
		ts.Close()

		healthy := ProbeProviderHealth(context.Background(), nil, config.Provider{Name: "p", BaseURL: badURL}, time.Second)
		if healthy {
			t.Error("expected an unreachable server to be considered unhealthy")
		}
	})
}
