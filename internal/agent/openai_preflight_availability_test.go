// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// newMinimalManagerWithProviders is newMinimalManager plus an app-level
// providers list, needed to exercise the model-availability preflight
// (Manager.preflightModelAvailability resolves providers by name).
func newMinimalManagerWithProviders(t *testing.T, agents []config.AgentConfig, providers []config.Provider) (*Manager, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	h := hub.New()
	idx, err := index.Open(filepath.Join(tmpDir, "mgr-test.db"), tmpDir, nil,
		index.WithHub(h),
	)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	locks := lock.New(idx, h)
	mgr := New(agents, 4, idx, nil, h, locks, nil, tmpDir, "", providers, config.AppAgentConfig{})
	return mgr, func() { idx.Close() }
}

func TestCheckModelAvailability(t *testing.T) {
	t.Run("model present", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"gemma-4-26b"},{"id":"other-model"}]}`))
		}))
		defer ts.Close()

		provider := config.Provider{Name: "local", BaseURL: ts.URL}
		ok, err := CheckModelAvailability(context.Background(), ts.Client(), provider, "gemma-4-26b")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !ok {
			t.Fatal("expected model to be reported available")
		}
	})

	t.Run("model missing", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"other-model"}]}`))
		}))
		defer ts.Close()

		provider := config.Provider{Name: "local", BaseURL: ts.URL}
		ok, err := CheckModelAvailability(context.Background(), ts.Client(), provider, "gemma-4-26b")
		if ok {
			t.Fatal("expected model to be reported unavailable")
		}
		if !errors.Is(err, ErrModelNotFound) {
			t.Fatalf("expected ErrModelNotFound, got %v", err)
		}
	})

	t.Run("endpoint unreachable", func(t *testing.T) {
		provider := config.Provider{Name: "local", BaseURL: "http://127.0.0.1:1"} // reserved, connection refused
		ok, err := CheckModelAvailability(context.Background(), http.DefaultClient, provider, "gemma-4-26b")
		if ok {
			t.Fatal("expected model to be reported unavailable")
		}
		if !errors.Is(err, ErrEndpointUnreachable) {
			t.Fatalf("expected ErrEndpointUnreachable, got %v", err)
		}
	})

	t.Run("endpoint returns 500", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		provider := config.Provider{Name: "local", BaseURL: ts.URL}
		ok, err := CheckModelAvailability(context.Background(), ts.Client(), provider, "gemma-4-26b")
		if ok {
			t.Fatal("expected model to be reported unavailable")
		}
		if !errors.Is(err, ErrEndpointUnreachable) {
			t.Fatalf("expected ErrEndpointUnreachable, got %v", err)
		}
	})

	t.Run("llama.cpp load state surfaces on the richer probe", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"gemma-4-26b","state":"unloaded"}]}`))
		}))
		defer ts.Close()

		provider := config.Provider{Name: "local", BaseURL: ts.URL}
		status, err := probeModelAvailability(context.Background(), ts.Client(), provider, "gemma-4-26b")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !status.Available || status.LoadState != "unloaded" {
			t.Fatalf("expected available with load state %q, got %+v", "unloaded", status)
		}
	})
}

// TestStartRun_Preflight verifies that a nonexistent model or unreachable
// endpoint aborts StartRun before any lineage lock is acquired, without
// mutating artifact status, and leaves a classified failed run record behind
// (local-model-operability Milestone 2, NFR-1/NFR-3).
func TestStartRun_Preflight(t *testing.T) {
	t.Run("model not found fails fast without acquiring the lock", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"other-model"}]}`))
		}))
		defer ts.Close()

		agents := []config.AgentConfig{
			{
				Name:            "local-agent",
				Roles:           []string{"backend-developer"},
				Driver:          "openai-compatible",
				Provider:        "local",
				Model:           "gemma-4-26b",
				PromptTemplates: map[string]string{"backend-developer": "test prompt {target_path}"},
			},
		}
		providers := []config.Provider{{Name: "local", BaseURL: ts.URL}}
		mgr, cleanup := newMinimalManagerWithProviders(t, agents, providers)
		defer cleanup()

		_, err := mgr.StartRun(context.Background(), "local-agent", "lifecycle/backend-plans/test-3-be.md", "backend-developer", nil)
		if err == nil {
			t.Fatal("expected preflight error, got nil")
		}
		if !strings.Contains(err.Error(), "model_not_found") {
			t.Fatalf("expected error to mention model_not_found, got: %v", err)
		}

		// No lock should have been acquired for this lineage.
		lockRow, lockErr := mgr.locks.Get("lifecycle/backend-plans/test-3-be.md")
		if lockErr != nil {
			t.Fatalf("checking lock: %v", lockErr)
		}
		if lockRow != nil {
			t.Fatalf("expected no lineage lock, got %+v", lockRow)
		}

		// A classified failed run record should exist for observability.
		runs, err := mgr.idx.ListAgentRuns("failed", 0)
		if err != nil {
			t.Fatalf("ListAgentRuns: %v", err)
		}
		if len(runs) != 1 {
			t.Fatalf("expected 1 failed run, got %d", len(runs))
		}
		if runs[0].FailureReason == nil || *runs[0].FailureReason != "model_not_found" {
			t.Fatalf("expected failure_reason=model_not_found, got %+v", runs[0].FailureReason)
		}
	})

	t.Run("endpoint unreachable fails fast", func(t *testing.T) {
		agents := []config.AgentConfig{
			{
				Name:            "local-agent",
				Roles:           []string{"backend-developer"},
				Driver:          "openai-compatible",
				Provider:        "local",
				Model:           "gemma-4-26b",
				PromptTemplates: map[string]string{"backend-developer": "test prompt {target_path}"},
			},
		}
		providers := []config.Provider{{Name: "local", BaseURL: "http://127.0.0.1:1"}}
		mgr, cleanup := newMinimalManagerWithProviders(t, agents, providers)
		defer cleanup()

		_, err := mgr.StartRun(context.Background(), "local-agent", "lifecycle/backend-plans/test-3-be.md", "backend-developer", nil)
		if err == nil {
			t.Fatal("expected preflight error, got nil")
		}
		if !strings.Contains(err.Error(), "endpoint_unreachable") {
			t.Fatalf("expected error to mention endpoint_unreachable, got: %v", err)
		}
	})

	t.Run("model found lets the run proceed past preflight", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"gemma-4-26b"}]}`))
		}))
		defer ts.Close()

		agents := []config.AgentConfig{
			{
				Name:            "local-agent",
				Roles:           []string{"backend-developer"},
				Driver:          "openai-compatible",
				Provider:        "local",
				Model:           "gemma-4-26b",
				PromptTemplates: map[string]string{"backend-developer": "test prompt {target_path}"},
			},
		}
		providers := []config.Provider{{Name: "local", BaseURL: ts.URL}}
		mgr, cleanup := newMinimalManagerWithProviders(t, agents, providers)
		defer cleanup()

		_, err := mgr.StartRun(context.Background(), "local-agent", "lifecycle/backend-plans/test-3-be.md", "backend-developer", nil)
		// The run itself will fail shortly after (the test server doesn't
		// implement /v1/chat/completions), but it must get past the
		// preflight — i.e. not fail synchronously with a preflight error.
		if err != nil && strings.Contains(err.Error(), "preflight") {
			t.Fatalf("expected preflight to pass, got: %v", err)
		}
	})
}
