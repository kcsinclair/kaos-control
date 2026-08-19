// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/directives"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/initcmd"
	"github.com/kaos-control/kaos-control/internal/project"
)

// projectSummary is the JSON representation of a registered project.
type projectSummary struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Initialised bool   `json:"initialised"`
	// DirectivesMigrationAvailable is true when the project is still on the
	// legacy single-CLAUDE.md layout and could be upgraded to the
	// AGENTS.md-primary directive set (see directives.NeedsMigration) — the
	// frontend uses this to surface a migration-offer banner.
	DirectivesMigrationAvailable bool `json:"directivesMigrationAvailable"`
}

func entryToSummary(e *config.ProjectEntry) projectSummary {
	migrationAvailable, err := directives.NeedsMigration(e.Path)
	if err != nil {
		slog.Warn("checking directives migration availability", "project", e.Name, "err", err)
	}
	return projectSummary{
		Name:                         e.Name,
		Path:                         e.Path,
		Description:                  e.Description,
		Owner:                        e.Owner,
		Initialised:                  config.IsInitialised(e.Path),
		DirectivesMigrationAvailable: migrationAvailable,
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
// project's path. Delegates to initcmd.ScaffoldProject so the GUI path
// produces the same layout as `kaos-control init` (full agent config,
// CLAUDE.md, .claude/settings.json, .gitignore, devops/sample.yaml,
// lifecycle/docs/, etc.). The logged-in session user is auto-populated
// as the project owner in the rendered config.yaml's users: section.
// Operation is idempotent: existing files and directories are skipped.
func (s *Server) handleInitProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}

	// requireAuth covers /api/* already, but be explicit — the session
	// user's email is critical (it becomes the project owner) so a
	// nil here is a programmer error worth surfacing loudly.
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "init requires an authenticated session"))
		return
	}

	projectPath := p.Entry.Path

	res, err := initcmd.ScaffoldProject(initcmd.ScaffoldOptions{
		ProjectRoot: projectPath,
		ProjectName: name,
		OwnerEmail:  user.Email,
		// Force left zero — idempotent.
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("init_failed", err.Error()))
		return
	}

	// Flatten Dirs + Files into the single 'created' list the
	// InitProjectModal renders, preserving the directories-first ordering
	// the CLI uses in its summary.
	var created []string
	for _, r := range res.Dirs {
		if r.Created {
			created = append(created, r.Path)
		}
	}
	for _, r := range res.Files {
		if r.Created {
			created = append(created, r.Path)
		}
	}
	// Include the architecture catalog scaffold (README, architectures/*,
	// tech-stacks/*, decisions/ + standards/ .gitkeep) so a fresh project's
	// git tracks everything kaos-control created, not just the top-level
	// scaffold.
	for _, r := range res.Architecture {
		if r.Created {
			created = append(created, r.Path)
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
		// Mark this repo as kaos-control-created so later scaffolding
		// (directive generation) auto-commits rather than handing back
		// commands, per the new-folder policy.
		if err := kgit.MarkManaged(projectPath); err != nil {
			slog.Warn("init: failed to mark repo as kaos-control-managed", "project", name, "err", err)
		}

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
		"created":         created,
		"git_initialised": gitInitialised,
	}
	if gitCommands != nil {
		resp["git_commands"] = gitCommands
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleMigrateDirectives handles POST /api/projects/{project}/migrate-directives:
// upgrades a project from the legacy single-CLAUDE.md layout to the
// AGENTS.md-primary directive set (directives.Migrate). Role-gated to
// product-owner (project-admin), mirroring handleInitProject's scope.
func (s *Server) handleMigrateDirectives(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}
	if !requireRole(w, r, p, RolesAdminOnly...) {
		return
	}

	var body struct {
		Force bool `json:"force"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
			return
		}
	}

	res, err := directives.Migrate(p.Entry.Path, directives.MigrateOptions{Force: body.Force})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("migrate_failed", err.Error()))
		return
	}
	res.Committed, res.GitCommands = trackDirectiveFiles(p.Entry.Path, res.Files, "kaos-control: migrate agent directives")

	writeJSON(w, http.StatusOK, res)
}

// trackDirectiveFiles gets the root directive files a migrate/refresh run wrote
// under git per the new-folder policy: for a repo kaos-control created it
// auto-commits them and returns committed=true; for a pre-existing user repo it
// returns the git add/commit commands for the user to run (never touching their
// history). Directive files (AGENTS.md/CLAUDE.md/GEMINI.md) live at the project
// root, outside the index and fsnotify watch, and generation never touches git
// itself — so without this they'd be written but left untracked (FR-17).
func trackDirectiveFiles(projectPath string, files []directives.FileWrite, commitMsg string) (bool, []string) {
	var paths []string
	for _, f := range files {
		// Only files actually written this run — skip pending-diff (withheld)
		// and skipped entries.
		if (f.Created || f.Changed) && f.Diff == "" && !f.Skipped {
			paths = append(paths, filepath.ToSlash(f.Path))
		}
	}
	committed, cmds, err := kgit.TrackGenerated(projectPath, paths, commitMsg)
	if err != nil {
		slog.Warn("directives: git tracking failed", "project", projectPath, "err", err)
	}
	return committed, cmds
}

// handleRefreshDirectives handles POST /api/p/{project}/directives/refresh:
// regenerates AGENTS.md/CLAUDE.md/GEMINI.md and re-patches the six standard
// agents from the project's promoted stack (directives.Generate). Role-gated
// to product-owner (project-admin). Reloads the project's live config
// afterward so a config.yaml patch (new allowed_write_paths, disabled
// agents) takes effect without a server restart.
func (s *Server) handleRefreshDirectives(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RolesAdminOnly...) {
		return
	}

	var body struct {
		Force bool `json:"force"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
			return
		}
	}

	res, err := directives.Generate(p.Entry.Path, directives.GenerateOptions{Force: body.Force})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("refresh_failed", err.Error()))
		return
	}
	res.Committed, res.GitCommands = trackDirectiveFiles(p.Entry.Path, res.Files, "kaos-control: refresh agent directives")

	if err := p.ReloadConfig(); err != nil {
		slog.Warn("directives refresh: failed to reload project config", "project", p.Entry.Name, "err", err)
	}

	writeJSON(w, http.StatusOK, res)
}

// checkDirectoryRequest is the mode-aware body for POST /projects/check-directory.
// Existing mode uses Path; new mode uses Parent + Name.
type checkDirectoryRequest struct {
	Mode   string `json:"mode"`
	Path   string `json:"path,omitempty"`
	Parent string `json:"parent,omitempty"`
	Name   string `json:"name,omitempty"`
}

// checkDirectoryResult is the mode-aware response for POST /projects/check-directory.
// Existing mode populates Exists/IsDir/Writable/Initialised; new mode populates
// ParentExists/ParentWritable/NameValid/TargetExists. ResolvedPath is always
// populated (FR9) and Reason is set when NameValid is false.
type checkDirectoryResult struct {
	// Existing-mode fields.
	Exists      bool `json:"exists"`
	IsDir       bool `json:"isDir"`
	Writable    bool `json:"writable"`
	Initialised bool `json:"initialised"`

	// New-mode fields.
	ParentExists   bool `json:"parentExists"`
	ParentWritable bool `json:"parentWritable"`
	NameValid      bool `json:"nameValid"`
	TargetExists   bool `json:"targetExists"`

	ResolvedPath string `json:"resolvedPath"`
	Reason       string `json:"reason,omitempty"`
}

// handleCheckDirectory validates a filesystem path before form submission.
// Does not require the project to be registered. Serves both the "existing"
// and "new" directory modes (FR2/FR3); the resolved path is always returned
// so the UI can show exactly what will be written (FR9).
func (s *Server) handleCheckDirectory(w http.ResponseWriter, r *http.Request) {
	var body checkDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
		return
	}

	switch body.Mode {
	case "", "existing":
		s.checkExistingDirectory(w, body.Path)
	case "new":
		s.checkNewDirectory(w, body.Parent, body.Name)
	default:
		writeJSON(w, http.StatusBadRequest, apiError("invalid_mode", `mode must be "existing" or "new"`))
	}
}

// checkExistingDirectory implements the "existing" mode of handleCheckDirectory.
func (s *Server) checkExistingDirectory(w http.ResponseWriter, path string) {
	normalized := config.NormalizePath(path)

	if err := config.ValidatePathFormat(normalized); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	info, statErr := os.Stat(normalized)
	exists := statErr == nil
	isDir := exists && info.IsDir()
	writable := isDir && isWritable(normalized)

	writeJSON(w, http.StatusOK, checkDirectoryResult{
		Exists:       exists,
		IsDir:        isDir,
		Writable:     writable,
		Initialised:  isDir && config.IsInitialised(normalized),
		ResolvedPath: normalized,
	})
}

// checkNewDirectory implements the "new" mode of handleCheckDirectory.
func (s *Server) checkNewDirectory(w http.ResponseWriter, parent, name string) {
	normalizedParent := config.NormalizePath(parent)

	if err := config.ValidatePathFormat(normalizedParent); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	parentInfo, statErr := os.Stat(normalizedParent)
	parentExists := statErr == nil && parentInfo.IsDir()
	parentWritable := parentExists && isWritable(normalizedParent)

	result := checkDirectoryResult{
		ParentExists:   parentExists,
		ParentWritable: parentWritable,
		ResolvedPath:   normalizedParent,
	}

	target, err := config.ResolveNewTarget(parent, name)
	if err != nil {
		result.Reason = err.Error()
		writeJSON(w, http.StatusOK, result)
		return
	}
	// Re-assert the config-dir guard on the resolved target (NFR1).
	if err := config.ValidatePathFormat(target); err != nil {
		result.Reason = err.Error()
		writeJSON(w, http.StatusOK, result)
		return
	}

	result.NameValid = true
	result.ResolvedPath = target
	if _, err := os.Stat(target); err == nil {
		result.TargetExists = true
	}

	writeJSON(w, http.StatusOK, result)
}

// isWritable reports whether the directory at path is writable by the current process.
func isWritable(path string) bool {
	probe := filepath.Join(path, ".kaos-write-probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(probe)
	return true
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

// createProjectRequest is the mode-aware body for POST /projects. Existing
// mode uses Path; new mode uses Parent + DirName (the target directory name,
// distinct from the project Name).
type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Mode        string `json:"mode"`
	Path        string `json:"path,omitempty"`
	Parent      string `json:"parent,omitempty"`
	DirName     string `json:"dirName,omitempty"`
}

// createProjectResult is the response for POST /projects: the registered
// project summary plus onboarding metadata the frontend needs to render
// FR7/FR8/resolved-partial-tree feedback.
type createProjectResult struct {
	projectSummary
	ResolvedPath       string   `json:"resolvedPath"`
	Created            []string `json:"created,omitempty"`
	AlreadyInitialised bool     `json:"alreadyInitialised"`
	PartialCompletion  bool     `json:"partialCompletion"`
}

// handleCreateProject registers a new project and persists it to the
// registry. Mode-aware (FR1): "existing" scaffolds into a user-chosen
// pre-existing directory (FR2, FR5); "new" creates the target directory
// itself, then scaffolds it (FR3, FR6). Both modes converge on the same
// scaffold-and-register path so a project created either way is
// indistinguishable at rest from a CLI-`init` project (NFR3).
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body createProjectRequest
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

	// The session user's email becomes lifecycle/config.yaml's owner (as for
	// handleInitProject) — required now that this path always scaffolds.
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "create requires an authenticated session"))
		return
	}

	mode := body.Mode
	if mode == "" {
		mode = "existing"
	}

	var target string
	switch mode {
	case "existing":
		t, apiErr := resolveExistingTarget(body.Path)
		if apiErr != nil {
			writeJSON(w, http.StatusBadRequest, apiErr)
			return
		}
		target = t
	case "new":
		t, apiErr := resolveNewTargetForCreate(body.Parent, body.DirName)
		if apiErr != nil {
			writeJSON(w, http.StatusBadRequest, apiErr)
			return
		}
		target = t
	default:
		writeJSON(w, http.StatusBadRequest, apiError("invalid_mode", `mode must be "existing" or "new"`))
		return
	}

	// Reject an already-initialised target rather than re-scaffolding (FR4/NFR2).
	if config.IsInitialised(target) {
		writeJSON(w, http.StatusOK, createProjectResult{
			ResolvedPath:       target,
			AlreadyInitialised: true,
		})
		return
	}

	createdDir := false
	if mode == "new" {
		// Create only the target itself, not missing parents (FR6).
		if err := os.Mkdir(target, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("mkdir_failed", "creating target directory: "+err.Error()))
			return
		}
		createdDir = true
	}
	rollbackDir := func() {
		if createdDir {
			_ = os.RemoveAll(target)
		}
	}

	res, err := initcmd.ScaffoldProject(initcmd.ScaffoldOptions{
		ProjectRoot: target,
		ProjectName: body.Name,
		OwnerEmail:  user.Email,
		// Force left zero — idempotent, so FR5 (non-destructive) holds for free.
	})
	if err != nil {
		rollbackDir()
		writeJSON(w, http.StatusInternalServerError, apiError("scaffold_failed", err.Error()))
		return
	}

	var created []string
	dirsCreated := 0
	for _, d := range res.Dirs {
		if d.Created {
			created = append(created, d.Path)
			dirsCreated++
		}
	}
	for _, f := range res.Files {
		if f.Created {
			created = append(created, f.Path)
		}
	}
	// A partial pre-existing lifecycle/ tree is one where some (not all)
	// stage directories already existed and the rest were just filled in.
	partialCompletion := mode == "existing" && dirsCreated > 0 && dirsCreated < len(res.Dirs)

	entry := &config.ProjectEntry{
		Name:        body.Name,
		Path:        target,
		Description: body.Description,
		Owner:       body.Owner,
	}

	if err := config.SaveProjectEntry(s.projectsDir, entry); err != nil {
		rollbackDir()
		writeJSON(w, http.StatusInternalServerError, apiError("save_failed", "saving project entry: "+err.Error()))
		return
	}

	if err := s.RegisterProject(entry); err != nil {
		// Roll back: remove the saved YAML file, and the directory we created (FR8).
		_ = config.DeleteProjectEntry(s.projectsDir, entry.Name)
		rollbackDir()
		writeJSON(w, http.StatusInternalServerError, apiError("register_failed", "registering project: "+err.Error()))
		return
	}

	p, _ := s.getProject(entry.Name)
	writeJSON(w, http.StatusCreated, createProjectResult{
		projectSummary:     projectToSummary(p),
		ResolvedPath:       target,
		Created:            created,
		AlreadyInitialised: false,
		PartialCompletion:  partialCompletion,
	})
}

// resolveExistingTarget validates an "existing" mode path (FR4) and returns
// the normalised, existence/writability-checked target, or a distinct error
// map identifying the failure (FR8).
func resolveExistingTarget(path string) (string, map[string]any) {
	if path == "" {
		return "", apiError("path_missing", "path must not be empty")
	}
	normalized := config.NormalizePath(path)
	if err := config.ValidatePathFormat(normalized); err != nil {
		return "", apiError("invalid_path", err.Error())
	}
	info, statErr := os.Stat(normalized)
	if statErr != nil {
		return "", apiError("path_missing", "path does not exist: "+normalized)
	}
	if !info.IsDir() {
		return "", apiError("not_a_directory", "path is not a directory: "+normalized)
	}
	if !isWritable(normalized) {
		return "", apiError("not_writable", "path is not writable: "+normalized)
	}
	return normalized, nil
}

// resolveNewTargetForCreate validates a "new" mode parent + directory name
// (FR3/FR4) and returns the resolved target, or a distinct error map
// identifying the failure (FR8). The target is routed through
// config.ResolveNewTarget so a crafted name cannot escape parent (NFR1); the
// config-dir guard is re-asserted on the resolved target.
func resolveNewTargetForCreate(parent, dirName string) (string, map[string]any) {
	if parent == "" {
		return "", apiError("parent_missing", "parent must not be empty")
	}
	normalizedParent := config.NormalizePath(parent)
	if err := config.ValidatePathFormat(normalizedParent); err != nil {
		return "", apiError("invalid_path", err.Error())
	}
	parentInfo, statErr := os.Stat(normalizedParent)
	if statErr != nil || !parentInfo.IsDir() {
		return "", apiError("parent_missing", "parent does not exist: "+normalizedParent)
	}
	if !isWritable(normalizedParent) {
		return "", apiError("parent_not_writable", "parent is not writable: "+normalizedParent)
	}

	target, err := config.ResolveNewTarget(parent, dirName)
	if err != nil {
		return "", apiError("invalid_name", err.Error())
	}
	// Re-assert the config-dir guard on the resolved target (NFR1).
	if err := config.ValidatePathFormat(target); err != nil {
		return "", apiError("invalid_name", err.Error())
	}
	if _, err := os.Stat(target); err == nil {
		return "", apiError("target_exists", "target already exists: "+target)
	}
	return target, nil
}
