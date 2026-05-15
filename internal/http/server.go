// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/queue"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

// Server is the HTTP server for kaos-control.
type Server struct {
	cfg      ServerConfig
	router   chi.Router
	httpSrv  *http.Server
	queue    *queue.Dispatcher // nil when queue not configured
	appHub   *hub.Hub          // app-level hub for queue.* broadcast; nil if unconfigured

	// projects is the live map of registered projects.
	// All reads and writes must be done under projectsMu.
	// projectCancels holds the cancel function for each project's background context.
	// servCtx is set in ListenAndServe; new project goroutines are children of it.
	projectsMu     sync.RWMutex
	projects       map[string]*project.Project
	projectCancels map[string]context.CancelFunc
	servCtx        context.Context

	// Project CRUD runtime state.
	projectsDir string
	dataDir     string
	openOpts    project.OpenOptions

	// App-level config mutation (Ollama instance CRUD).
	appCfgMu   sync.RWMutex
	appCfg     *config.App
	appCfgPath string
}

// ServerConfig holds what the HTTP layer needs.
type ServerConfig struct {
	Listen     string
	TLSCert    string
	TLSKey     string
	TLSOn      bool
	Frontend   fs.FS
	Auth       *auth.Store  // nil when auth is not configured
	AppCfg     *config.App  // may be nil; required for Ollama instance management
	AppCfgPath string       // path to app config.yaml; required for Ollama instance management
	// PublicHost is a comma-separated list of additional hostnames the server
	// is reachable at; used to populate WebSocket OriginPatterns.
	// Local listen addresses (localhost, 127.0.0.1) are always allowed.
	PublicHost string
	// Queue is the app-level queue dispatcher. May be nil when the queue is
	// not configured (e.g. in minimal test environments).
	Queue *queue.Dispatcher
	// AppHub is the app-level WebSocket hub used by the /api/ws endpoint
	// to broadcast queue-level events (queue.added, queue.paused, etc.).
	AppHub *hub.Hub

	// ProjectsDir is the directory where project YAML registration files live
	// (e.g. ~/.kaos-control/projects/). Required for project CRUD endpoints.
	ProjectsDir string
	// DataDir is the app-level data directory used when opening new projects
	// (e.g. ~/.kaos-control/data/). Required for project CRUD endpoints.
	DataDir string
	// ProjectOpenOptions are the options forwarded to project.Open when a new
	// project is registered at runtime. Required for project CRUD endpoints.
	ProjectOpenOptions project.OpenOptions
}

// allowedWSOrigins returns the set of hostnames that are permitted as the
// Origin header on an incoming WebSocket upgrade. It always includes
// "localhost" and "127.0.0.1". If the configured Listen address has a
// non-wildcard host component, that host is included too. Any hosts listed
// in cfg.PublicHost (comma-separated) are appended.
func (s *Server) allowedWSOrigins() []string {
	out := []string{"localhost", "127.0.0.1"}
	if h, _, err := net.SplitHostPort(s.cfg.Listen); err == nil && h != "" && h != "0.0.0.0" && h != "::" {
		out = append(out, h)
	}
	for _, h := range strings.Split(s.cfg.PublicHost, ",") {
		if h = strings.TrimSpace(h); h != "" {
			out = append(out, h)
		}
	}
	return out
}

// New constructs and wires the server. projects maps project name → project.Project.
func New(cfg ServerConfig, projects map[string]*project.Project) *Server {
	s := &Server{
		cfg:            cfg,
		projects:       projects,
		projectCancels: make(map[string]context.CancelFunc),
		queue:          cfg.Queue,
		appHub:         cfg.AppHub,
		appCfg:         cfg.AppCfg,
		appCfgPath:     cfg.AppCfgPath,
		projectsDir:    cfg.ProjectsDir,
		dataDir:        cfg.DataDir,
		openOpts:       cfg.ProjectOpenOptions,
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

	// Hook permission endpoint: exempt from session auth and CSRF.
	// It authenticates via the per-run secret instead (FR8).
	r.Post("/api/agent/{run_id}/permission", s.handleHookPermission)

	// All remaining routes require session authentication and CSRF protection.
	r.Group(func(r chi.Router) {
		r.Use(s.sessionMiddleware)
		r.Use(s.csrfMiddleware)
		r.Use(s.requireAuth)

		r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/version", s.handleVersion)

		// Auth endpoints
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// Admin: user management
		r.Post("/admin/users", s.handleCreateUser)

		// Project registry
		r.Get("/projects", s.handleListProjects)
		r.Post("/projects", s.handleCreateProject)
		r.Get("/projects/{project}", s.handleGetProject)
		r.Put("/projects/{project}", s.handleUpdateProject)
		r.Delete("/projects/{project}", s.handleDeleteProject)
		r.Post("/projects/{project}/init", s.handleInitProject)
		// check-directory must be registered before /{project} to avoid
		// "check-directory" being matched as a project name.
		r.Post("/projects/check-directory", s.handleCheckDirectory)

		// App-level WebSocket (queue events, etc.)
		r.Get("/ws", s.handleAppWebSocket)

		// Queue management (app-level, not project-scoped)
		r.Post("/queue", s.handleEnqueue)
		r.Get("/queue", s.handleListQueue)
		r.Delete("/queue/{id}", s.handleCancelQueue)
		r.Post("/queue/pause", s.handlePauseQueue)
		r.Post("/queue/resume", s.handleResumeQueue)

		// Ollama instance management (app-level, not project-scoped)
		r.Route("/ollama/instances", func(r chi.Router) {
			r.Get("/", s.handleListOllamaInstances)
			r.Post("/", s.handleCreateOllamaInstance)
			r.Put("/{name}", s.handleUpdateOllamaInstance)
			r.Delete("/{name}", s.handleDeleteOllamaInstance)
			r.Get("/{name}/health", s.handleOllamaHealth)
			r.Get("/{name}/models", s.handleOllamaModels)
		})

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
				if strings.HasSuffix(param, "/allowed-targets") {
					s.handleAllowedTargets(w, r)
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
				if strings.HasSuffix(param, "/release") {
					s.handlePatchRelease(w, r)
					return
				}
				writeJSON(w, http.StatusNotFound, apiError("not_found", "unknown sub-route"))
			})

			// Agents
			r.Get("/agents", s.handleListAgents)
			r.Get("/agents/ready-counts", s.handleGetReadyCounts)
			r.Post("/agents/{name}/run", s.handleStartAgentRun)
			r.Get("/agents/runs", s.handleListAgentRuns)
			r.Get("/agents/runs/{run_id}", s.handleGetAgentRun)
			r.Get("/agents/runs/{run_id}/log", s.handleGetAgentRunLog)
			r.Get("/agents/runs/{run_id}/result", s.handleGetAgentRunResult)
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

			// Event feed
			r.Get("/feed", s.handleGetFeed)

			// Graph and discovery
			r.Get("/graph", s.handleGraph)
			r.Get("/labels", s.handleLabels)
			r.Get("/lineages", s.handleLineages)
			r.Get("/priorities", s.handlePriorities)
			r.Get("/parse-errors", s.handleParseErrors)

			// Project config
			r.Get("/config", s.handleGetConfig)
			r.Put("/config", s.handleUpdateConfig)
			r.Get("/config/kanban", s.handleGetKanbanConfig)
			r.Get("/config/roadmap", s.handleGetRoadmapConfig)

			// Roles and users
			r.Get("/roles", s.handleGetRoles)

			// Lineage status checker
			r.Get("/status-check", s.handleStatusCheck)
			r.Post("/status-check/advance", s.handleStatusCheckAdvance)

			// Releases
			r.Route("/releases", func(r chi.Router) {
				r.Get("/", s.handleListReleases)
				r.Post("/", s.handleCreateRelease)
				r.Get("/graph", s.handleRoadmapGraph)
				r.Get("/{releaseID}", s.handleGetRelease)
				r.Put("/{releaseID}", s.handleUpdateRelease)
				r.Delete("/{releaseID}", s.handleDeleteRelease)
				r.Get("/{releaseID}/artifacts", s.handleListReleaseArtifacts)
			})

			// Git context
			r.Get("/git/status", s.handleGetGitStatus)

			// Dashboard
			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/stats", s.handleGetDashboardStats)
				r.Get("/status-distribution", s.handleGetStatusDistribution)
				r.Get("/stage-distribution", s.handleGetStageDistribution)
				r.Get("/velocity", s.handleGetVelocity)
			})

			// DevOps pipelines
			r.Get("/devops/pipelines", s.handleListPipelines)
			r.Post("/devops/pipelines", s.handleCreatePipeline)
			r.Get("/devops/pipelines/{slug}", s.handleGetPipeline)
			r.Put("/devops/pipelines/{slug}", s.handleUpdatePipeline)
			r.Post("/devops/pipelines/{slug}/run", s.handleRunPipeline)
			r.Post("/devops/pipelines/{slug}/cancel", s.handleCancelPipeline)
			r.Get("/devops/runs/{run_id}", s.handleGetRunLog)

			// Scheduler
			r.Route("/scheduler", func(r chi.Router) {
				r.Get("/jobs", s.handleListSchedulerJobs)
				r.Post("/jobs", s.handleCreateSchedulerJob)
				r.Get("/jobs/{name}", s.handleGetSchedulerJob)
				r.Put("/jobs/{name}", s.handleUpdateSchedulerJob)
				r.Delete("/jobs/{name}", s.handleDeleteSchedulerJob)
				r.Post("/jobs/{name}/trigger", s.handleTriggerSchedulerJob)
				r.Post("/jobs/{name}/pause", s.handlePauseSchedulerJob)
				r.Post("/jobs/{name}/resume", s.handleResumeSchedulerJob)
				r.Get("/jobs/{name}/runs", s.handleListSchedulerRuns)
				r.Get("/jobs/{name}/runs/{id}/log", s.handleGetSchedulerRunLog)
			})
		})
		})

		r.Get("/*", s.handleFrontend)
	}) // end auth group

	return r
}

// SetQueue wires the queue dispatcher into the server. Must be called before
// ListenAndServe when the queue is created after New().
func (s *Server) SetQueue(q *queue.Dispatcher) {
	s.queue = q
}

// ListenAndServe starts the server and blocks until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	// Store the server context so that projects registered at runtime can derive
	// their goroutine contexts from it and be cancelled on shutdown.
	s.projectsMu.Lock()
	s.servCtx = ctx
	s.projectsMu.Unlock()

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

// handleVersion returns the server version as JSON.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": Version})
}

// getProject returns the named project under the read-lock.
func (s *Server) getProject(name string) (*project.Project, bool) {
	s.projectsMu.RLock()
	p, ok := s.projects[name]
	s.projectsMu.RUnlock()
	return p, ok
}

// GetProject is the exported variant used by external callers (e.g. queue dispatcher).
func (s *Server) GetProject(name string) (*project.Project, bool) {
	return s.getProject(name)
}

// RegisterProject opens a project, starts its goroutines, and adds it to the
// server's live project map. The project's goroutines run until
// UnregisterProject is called or the server shuts down.
func (s *Server) RegisterProject(entry *config.ProjectEntry) error {
	// Snapshot the server context under the read-lock.
	s.projectsMu.RLock()
	parent := s.servCtx
	q := s.queue
	s.projectsMu.RUnlock()
	if parent == nil {
		parent = context.Background()
	}
	pCtx, cancel := context.WithCancel(parent)

	p, err := project.Open(entry, s.dataDir, s.openOpts)
	if err != nil {
		cancel()
		return fmt.Errorf("opening project %q: %w", entry.Name, err)
	}
	p.StartWatcher(pCtx)
	p.StartLockReaper(pCtx)
	p.StartSessionReaper(pCtx)
	p.StartScheduler(pCtx)

	// Wire queue pause callback so denied tool calls pause the queue.
	if q != nil && p.Agents != nil {
		p.Agents.PauseQueue = func(reason string) { q.Pause(reason) }
	}

	s.projectsMu.Lock()
	s.projects[entry.Name] = p
	s.projectCancels[entry.Name] = cancel
	s.projectsMu.Unlock()
	return nil
}

// UnregisterProject stops a project's goroutines, closes its index, and removes
// it from the server's live project map. Does not delete any on-disk files.
func (s *Server) UnregisterProject(name string) error {
	s.projectsMu.Lock()
	p, ok := s.projects[name]
	cancel := s.projectCancels[name]
	delete(s.projects, name)
	delete(s.projectCancels, name)
	s.projectsMu.Unlock()

	if !ok {
		return fmt.Errorf("project not found: %s", name)
	}
	// Cancel the project context to stop watcher/reaper/scheduler goroutines.
	if cancel != nil {
		cancel()
	}
	return p.Close()
}

// TrackCancel records a cancel function for a project that was opened and
// started externally (e.g. during server startup in main). This ensures that
// UnregisterProject can cleanly stop those goroutines too.
func (s *Server) TrackCancel(name string, cancel context.CancelFunc) {
	s.projectsMu.Lock()
	s.projectCancels[name] = cancel
	s.projectsMu.Unlock()
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
