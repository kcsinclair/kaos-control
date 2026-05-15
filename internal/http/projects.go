// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/project"
)

// projectSummary is the JSON representation of a registered project.
type projectSummary struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Initialised bool   `json:"initialised"`
}

func entryToSummary(e *config.ProjectEntry) projectSummary {
	return projectSummary{
		Name:        e.Name,
		Path:        e.Path,
		Description: e.Description,
		Owner:       e.Owner,
		Initialised: config.IsInitialised(e.Path),
	}
}

func projectToSummary(p *project.Project) projectSummary {
	return entryToSummary(p.Entry)
}

// handleListProjects returns all registered projects.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	s.projectsMu.RLock()
	out := make([]projectSummary, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, projectToSummary(p))
	}
	s.projectsMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

// handleGetProject returns a single project by name.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}
	writeJSON(w, http.StatusOK, projectToSummary(p))
}

// handleInitProject creates kaos-control scaffolding inside a registered
// project's path (lifecycle/config.yaml + default stage directories).
// The operation is idempotent: existing files and directories are left untouched.
func (s *Server) handleInitProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}

	projectPath := p.Entry.Path
	lifecycleDir := filepath.Join(projectPath, "lifecycle")
	var created []string

	// Create lifecycle/config.yaml if absent.
	cfgPath := filepath.Join(lifecycleDir, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.MkdirAll(lifecycleDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("init_failed", "creating lifecycle dir: "+err.Error()))
			return
		}
		if err := os.WriteFile(cfgPath, []byte(config.DefaultProjectConfigYAML()), 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("init_failed", "writing config.yaml: "+err.Error()))
			return
		}
		created = append(created, "lifecycle/config.yaml")
	}

	// Create one subdirectory (+ .gitkeep) per default stage.
	for _, stage := range config.DefaultStages() {
		stageDir := filepath.Join(lifecycleDir, stage)
		gitkeep := filepath.Join(stageDir, ".gitkeep")

		newDir := false
		if _, err := os.Stat(stageDir); os.IsNotExist(err) {
			if err := os.MkdirAll(stageDir, 0o755); err != nil {
				writeJSON(w, http.StatusInternalServerError, apiError("init_failed", fmt.Sprintf("creating %s: %s", stage, err)))
				return
			}
			newDir = true
		}
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
				writeJSON(w, http.StatusInternalServerError, apiError("init_failed", fmt.Sprintf("writing .gitkeep in %s: %s", stage, err)))
				return
			}
			if newDir {
				created = append(created, filepath.Join("lifecycle", stage))
			} else {
				created = append(created, filepath.Join("lifecycle", stage, ".gitkeep"))
			}
		}
	}

	// Git handling.
	gitInitialised := false
	var gitCommands []string

	if !kgit.IsRepo(projectPath) {
		// Initialise git and commit the scaffolding.
		if _, err := gogit.PlainInit(projectPath, false); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("git_init_failed", "git init: "+err.Error()))
			return
		}
		gitInitialised = true

		if len(created) > 0 {
			repo, err := kgit.Open(projectPath)
			if err != nil {
				slog.Warn("init: opened git repo but failed to open kgit repo", "project", name, "err", err)
			} else {
				// Build relative paths for git add.
				relPaths := make([]string, 0, len(created))
				for _, c := range created {
					relPaths = append(relPaths, filepath.ToSlash(c))
				}
				if _, err := repo.AddAndCommit(relPaths, "chore: initialise kaos-control project", "kaos-control", "noreply@kaos-control.local"); err != nil {
					slog.Warn("init: git commit failed", "project", name, "err", err)
				}
			}
		}
	} else if len(created) > 0 {
		// Git already exists — return the commands the user should run.
		addArgs := ""
		for _, c := range created {
			addArgs += " " + filepath.ToSlash(c)
		}
		gitCommands = []string{
			fmt.Sprintf("git -C %s add%s", projectPath, addArgs),
			fmt.Sprintf(`git -C %s commit -m "chore: initialise kaos-control project"`, projectPath),
		}
	}

	// Re-open the project so it picks up the new lifecycle/config.yaml and
	// starts watching the newly created directories.
	entry := p.Entry
	if err := s.UnregisterProject(name); err != nil {
		slog.Warn("init: failed to unregister project before re-open", "project", name, "err", err)
	}
	if err := s.RegisterProject(entry); err != nil {
		slog.Warn("init: failed to re-register project after init", "project", name, "err", err)
	}

	resp := map[string]any{
		"created":       created,
		"git_initialised": gitInitialised,
	}
	if gitCommands != nil {
		resp["git_commands"] = gitCommands
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleDeleteProject unloads a project from the server and removes its registry file.
// No on-disk project files are deleted.
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	if _, ok := s.getProject(name); !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}

	// Remove the registry YAML first so that if Close() takes a while the project
	// is already gone from disk and won't be re-loaded on next restart.
	if err := config.DeleteProjectEntry(s.projectsDir, name); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("delete_failed", "removing registry entry: "+err.Error()))
		return
	}

	if err := s.UnregisterProject(name); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("unregister_failed", "unregistering project: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleUpdateProject updates mutable project fields (description, owner, path).
// name is immutable; if included in the body it is ignored.
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}

	var body struct {
		Description *string `json:"description"`
		Owner       *string `json:"owner"`
		Path        *string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
		return
	}

	// Build updated entry from existing values.
	entry := &config.ProjectEntry{
		Name:        p.Entry.Name,
		Path:        p.Entry.Path,
		Description: p.Entry.Description,
		Owner:       p.Entry.Owner,
	}
	if body.Description != nil {
		entry.Description = *body.Description
	}
	if body.Owner != nil {
		entry.Owner = *body.Owner
	}

	pathChanged := false
	if body.Path != nil && *body.Path != p.Entry.Path {
		resolved, err := config.ValidatePath(*body.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
			return
		}
		entry.Path = resolved
		pathChanged = true
	}

	// Persist to disk atomically.
	if err := config.SaveProjectEntry(s.projectsDir, entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("save_failed", "saving project entry: "+err.Error()))
		return
	}

	if pathChanged {
		// Re-initialise project runtime for the new path.
		if err := s.UnregisterProject(name); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("unregister_failed", "unregistering project: "+err.Error()))
			return
		}
		if err := s.RegisterProject(entry); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("register_failed", "re-registering project at new path: "+err.Error()))
			return
		}
		p, _ = s.getProject(name)
	} else {
		// In-place update of non-path fields.
		s.projectsMu.RLock()
		p.Entry.Description = entry.Description
		p.Entry.Owner = entry.Owner
		s.projectsMu.RUnlock()
	}

	writeJSON(w, http.StatusOK, projectToSummary(p))
}

// handleCreateProject registers a new project and persists it to the registry.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Description string `json:"description"`
		Owner       string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
		return
	}

	if err := config.ValidateProjectName(body.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_name", err.Error()))
		return
	}

	if _, exists := s.getProject(body.Name); exists {
		writeJSON(w, http.StatusConflict, apiError("conflict", "project already exists: "+body.Name))
		return
	}

	if body.Path == "" {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", "path must not be empty"))
		return
	}
	resolved, err := config.ValidatePath(body.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	entry := &config.ProjectEntry{
		Name:        body.Name,
		Path:        resolved,
		Description: body.Description,
		Owner:       body.Owner,
	}

	if err := config.SaveProjectEntry(s.projectsDir, entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("save_failed", "saving project entry: "+err.Error()))
		return
	}

	if err := s.RegisterProject(entry); err != nil {
		// Roll back: remove the saved YAML file since registration failed.
		_ = config.DeleteProjectEntry(s.projectsDir, entry.Name)
		writeJSON(w, http.StatusInternalServerError, apiError("register_failed", "registering project: "+err.Error()))
		return
	}

	p, _ := s.getProject(entry.Name)
	writeJSON(w, http.StatusCreated, projectToSummary(p))
}
