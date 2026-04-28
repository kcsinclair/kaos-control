package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/project"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

// Server is the HTTP server for kaos-control.
type Server struct {
	cfg      ServerConfig
	router   chi.Router
	httpSrv  *http.Server
	projects map[string]*project.Project
}

// ServerConfig holds what the HTTP layer needs.
type ServerConfig struct {
	Listen   string
	TLSCert  string
	TLSKey   string
	TLSOn    bool
	Frontend fs.FS
	Auth     *auth.Store // nil when auth is not configured
}

// New constructs and wires the server. projects maps project name → project.Project.
func New(cfg ServerConfig, projects map[string]*project.Project) *Server {
	s := &Server{
		cfg:      cfg,
		projects: projects,
	}
	s.router = s.buildRouter()
	s.httpSrv = &http.Server{
		Addr:         cfg.Listen,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(s.sessionMiddleware)
	r.Use(s.csrfMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)

		// Auth endpoints
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// Admin: user management
		r.Post("/admin/users", s.handleCreateUser)

		// Project registry
		r.Get("/projects", s.handleListProjects)

		// Per-project routes
		r.Route("/p/{project}", func(r chi.Router) {
			r.Use(s.projectMiddleware)

			// Artifacts
			r.Get("/artifacts", s.handleListArtifacts)
			r.Post("/artifacts", s.handleCreateArtifact)
			// Chi wildcards are greedy, so dispatch sub-routes manually.
			r.Get("/artifacts/*", func(w http.ResponseWriter, r *http.Request) {
				param := chi.URLParam(r, "*")
				if strings.HasSuffix(param, "/history") {
					s.handleGetArtifactHistory(w, r)
					return
				}
				s.handleGetArtifact(w, r)
			})
			r.Put("/artifacts/*", func(w http.ResponseWriter, r *http.Request) {
				s.handleUpdateArtifact(w, r)
			})
			r.Delete("/artifacts/*", func(w http.ResponseWriter, r *http.Request) {
				param := chi.URLParam(r, "*")
				// Strip any accidental trailing slash.
				r2 := r.WithContext(r.Context())
				_ = param
				s.handleDeleteArtifact(w, r2)
			})
			r.Post("/artifacts/*", func(w http.ResponseWriter, r *http.Request) {
				param := chi.URLParam(r, "*")
				if strings.HasSuffix(param, "/rename") {
					s.handleRenameArtifact(w, r)
					return
				}
				if strings.HasSuffix(param, "/transition") {
					s.handleTransitionArtifact(w, r)
					return
				}
				writeJSON(w, http.StatusNotFound, apiError("not_found", "unknown sub-route"))
			})
			r.Patch("/artifacts/*", func(w http.ResponseWriter, r *http.Request) {
				param := chi.URLParam(r, "*")
				if strings.HasSuffix(param, "/priority") {
					s.handlePatchPriority(w, r)
					return
				}
				writeJSON(w, http.StatusNotFound, apiError("not_found", "unknown sub-route"))
			})

			// Agents
			r.Get("/agents", s.handleListAgents)
			r.Post("/agents/{name}/run", s.handleStartAgentRun)
			r.Get("/agents/runs", s.handleListAgentRuns)
			r.Get("/agents/runs/{run_id}", s.handleGetAgentRun)
			r.Get("/agents/runs/{run_id}/log", s.handleGetAgentRunLog)
			r.Post("/agents/runs/{run_id}/kill", s.handleKillAgentRun)

			// Locks
			r.Get("/locks", s.handleListLocks)
			r.Post("/locks", s.handleAcquireLock)
			r.Delete("/locks/{lineage}", s.handleReleaseLock)
			r.Post("/locks/{lineage}/heartbeat", s.handleHeartbeatLock)

			// Conversational idea capture
			r.Post("/ideas/converse", s.handleIdeaConverse)
			// Single-submit idea / defect capture (preview-only, no disk write)
			r.Post("/ideas/generate", s.handleIdeaGenerate)

			// WebSocket
			r.Get("/ws", s.handleWebSocket)

			// Graph and discovery
			r.Get("/graph", s.handleGraph)
			r.Get("/labels", s.handleLabels)
			r.Get("/lineages", s.handleLineages)
			r.Get("/priorities", s.handlePriorities)
			r.Get("/parse-errors", s.handleParseErrors)

			// Project config
			r.With(requireAuth).Get("/config", s.handleGetConfig)
			r.With(requireAuth).Put("/config", s.handleUpdateConfig)
			r.With(requireAuth).Get("/config/kanban", s.handleGetKanbanConfig)

			// Roles and users
			r.With(requireAuth).Get("/roles", s.handleGetRoles)
		})
	})

	r.Get("/*", s.handleFrontend)
	return r
}

// ListenAndServe starts the server and blocks until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.cfg.Listen, err)
	}
	slog.Info("kaos-control started", "addr", ln.Addr().String(), "version", Version)

	errCh := make(chan error, 1)
	go func() {
		if s.cfg.TLSOn {
			errCh <- s.httpSrv.ServeTLS(ln, s.cfg.TLSCert, s.cfg.TLSKey)
		} else {
			errCh <- s.httpSrv.Serve(ln)
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(shutCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// handleHealth returns a simple liveness response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": Version,
	})
}

// handleListProjects returns all registered projects.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	type projectSummary struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Path        string `json:"path"`
	}
	var out []projectSummary
	for _, p := range s.projects {
		out = append(out, projectSummary{
			Name:        p.Entry.Name,
			Description: p.Entry.Description,
			Path:        p.Entry.Path,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

// handleFrontend serves the embedded Vue SPA.
// Static assets are served as-is; unknown paths fall back to index.html.
func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Frontend == nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}
	dist, err := fs.Sub(s.cfg.Frontend, "dist")
	if err != nil {
		http.Error(w, "frontend unavailable", http.StatusInternalServerError)
		return
	}

	path := r.URL.Path
	if path == "" || path == "/" {
		serveFSFile(w, r, dist, "index.html")
		return
	}
	if path[0] == '/' {
		path = path[1:]
	}

	f, err := dist.Open(path)
	if err != nil {
		serveFSFile(w, r, dist, "index.html")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		serveFSFile(w, r, dist, "index.html")
		return
	}
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f.(io.ReadSeeker))
}

func serveFSFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		http.Error(w, name+" not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "stat error", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f.(io.ReadSeeker))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode failed", "err", err)
	}
}

func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
