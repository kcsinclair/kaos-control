// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// projectRuntimeDir returns the per-project runtime state directory (the
// same base directory the SQLite index and scheduler runs live under),
// which the wizard uses for scratch, resumable state — always outside
// lifecycle/, so it never counts as a write under lifecycle/architecture/
// (NFR-1).
func (s *Server) projectRuntimeDir(p *project.Project) string {
	return filepath.Join(s.dataDir, p.Entry.Name)
}

// priorRunInfo reports whether the Architecture Wizard has already run in
// this project (FR-2), by scanning lifecycle/architecture/ for a promoted
// architecture/tech-stack, architecture-summary.md, and adr-0001-*.md.
type priorRunInfo struct {
	Detected     bool   `json:"detected"`
	Architecture string `json:"architecture,omitempty"`
	TechStack    string `json:"tech_stack,omitempty"`
	ADRPath      string `json:"adr_path,omitempty"`
	SummaryPath  string `json:"summary_path,omitempty"`
}

func detectPriorRun(projectRoot string) (priorRunInfo, error) {
	var pr priorRunInfo

	archDirAbs := filepath.Join(projectRoot, "lifecycle", "architecture")
	entries, err := os.ReadDir(archDirAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return pr, nil
		}
		return pr, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "architecture-summary.md" {
			pr.SummaryPath = path.Join("lifecycle/architecture", e.Name())
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(archDirAbs, e.Name()))
		if rerr != nil {
			continue
		}
		relPath := path.Join("lifecycle/architecture", e.Name())
		a := artifact.Parse(raw, relPath, time.Time{})
		switch a.FM.Type {
		case "architecture":
			pr.Architecture = relPath
		case "tech-stack":
			pr.TechStack = relPath
		}
	}

	if decisions, derr := os.ReadDir(filepath.Join(archDirAbs, "decisions")); derr == nil {
		for _, e := range decisions {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "adr-0001-") && strings.HasSuffix(e.Name(), ".md") {
				pr.ADRPath = path.Join("lifecycle/architecture/decisions", e.Name())
				break
			}
		}
	}

	pr.Detected = pr.Architecture != "" || pr.TechStack != "" || pr.SummaryPath != "" || pr.ADRPath != ""
	return pr, nil
}

// handleGetArchitectureWizard handles GET /api/p/{project}/architecture/wizard
func (s *Server) handleGetArchitectureWizard(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	prior, err := detectPriorRun(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	var resumable *architecture.WizardState
	if st, found, serr := architecture.LoadWizardState(s.projectRuntimeDir(p), user.Email); serr != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", serr.Error()))
		return
	} else if found {
		resumable = &st
	}

	cfg := p.Config().ArchitectureWizard
	writeJSON(w, http.StatusOK, map[string]any{
		"questions":            cfg.Questions,
		"default_architecture": cfg.DefaultArchitecture,
		"prior_run":            prior,
		"resumable_state":      resumable,
	})
}

// handleRecommendArchitecture handles POST /api/p/{project}/architecture/wizard/recommend
func (s *Server) handleRecommendArchitecture(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		Answers []architecture.Answer `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	arches, _, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	recs, dropped, err := architecture.Recommend(arches, p.Config().ArchitectureWizard, req.Answers)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("recommend_error", err.Error()))
		return
	}

	// Return empty arrays rather than JSON null (nil slices) so clients can
	// safely read .length without a null guard.
	if recs == nil {
		recs = []architecture.Recommendation{}
	}
	if dropped == nil {
		dropped = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recommendations":     recs,
		"dropped_constraints": dropped,
	})
}

// handleListWizardStacks handles GET /api/p/{project}/architecture/wizard/stacks
func (s *Server) handleListWizardStacks(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	archSlug := r.URL.Query().Get("architecture")
	if archSlug == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "architecture query parameter is required"))
		return
	}
	language := r.URL.Query().Get("language")

	arches, stacks, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	var chosen *architecture.CatalogItem
	for i := range arches {
		if arches[i].Slug == archSlug {
			chosen = &arches[i]
			break
		}
	}
	if chosen == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "architecture not found in catalog: "+archSlug))
		return
	}

	ranked := architecture.RankStacks(*chosen, stacks, language)
	writeJSON(w, http.StatusOK, map[string]any{"stacks": ranked})
}

// handleListWizardCatalog handles GET /architecture/wizard/catalog: the full
// candidate catalog — every architecture and tech-stack with title, summary,
// labels, related_to, and pros/cons — so the Browse step can render cards and
// the comparison table before any architecture is chosen (FR-5, resolves
// FE-plan OQ-6). Unlike wizard/recommend (needs answers, returns top 3) and
// wizard/stacks (needs a chosen architecture), this takes no inputs. pros/cons
// live only as `## Pros`/`## Cons` markdown bodies parsed by LoadCatalog, so
// this endpoint is the single HTTP source for them — the frontend never
// re-parses catalog markdown.
func (s *Server) handleListWizardCatalog(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	arches, stacks, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"architectures": arches, "tech_stacks": stacks})
}

// handlePutWizardState handles PUT /api/p/{project}/architecture/wizard/state
func (s *Server) handlePutWizardState(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var st architecture.WizardState
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	if err := architecture.SaveWizardState(s.projectRuntimeDir(p), user.Email, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

// handleDeleteWizardState handles DELETE /api/p/{project}/architecture/wizard/state
func (s *Server) handleDeleteWizardState(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	if err := architecture.ClearWizardState(s.projectRuntimeDir(p), user.Email); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true})
}

// handleCommitArchitectureWizard handles POST /api/p/{project}/architecture/wizard/commit
// — the single write entry point (FR-13, FR-14, FR-16, NFR-1, NFR-2). It is
// gated to product-owner (OQ-5): everything up to and including validating
// the requested architecture/tech-stack against the catalog happens before
// any write, so a rejected or invalid request leaves the project unchanged.
func (s *Server) handleCommitArchitectureWizard(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RoleProductOwner) {
		return
	}

	var req struct {
		ArchitecturePath     string                     `json:"architecture_path"`
		TechStackPath        string                     `json:"tech_stack_path"`
		Answers              []architecture.Answer      `json:"answers"`
		BreakingRequirements []architecture.BreakingReq `json:"breaking_requirements"`
		QA                   []architecture.QAPair      `json:"qa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.ArchitecturePath == "" || req.TechStackPath == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "architecture_path and tech_stack_path are required"))
		return
	}

	arches, stacks, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	archItem, ok := findCatalogItem(arches, req.ArchitecturePath)
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown architecture: "+req.ArchitecturePath))
		return
	}
	stackItem, ok := findCatalogItem(stacks, req.TechStackPath)
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown tech stack: "+req.TechStackPath))
		return
	}

	prior, err := detectPriorRun(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	isFirstRun := prior.ADRPath == ""

	promoteReq := architecture.PromotionRequest{
		ArchitectureCatalogPath: req.ArchitecturePath,
		TechStackCatalogPath:    req.TechStackPath,
	}
	changed, err := architecture.SelectionChanged(p.Entry.Path, promoteReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	result, err := architecture.Promote(p.Entry.Path, promoteReq)
	if err != nil {
		if errors.Is(err, sandbox.ErrPathTraversal) || errors.Is(err, sandbox.ErrAbsolutePath) || errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", err.Error()))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	rejected := rejectedAlternatives(arches, p.Config().ArchitectureWizard, req.Answers, archItem.Slug)
	qaTrail := renderQATrail(req.QA)

	var adrPath, supersededPath string
	switch {
	case isFirstRun:
		adrPath, err = architecture.WriteADR0001(p.Entry.Path, archItem.Title, stackItem.Title, qaTrail, rejected)
	case changed:
		body := qaTrail
		if len(rejected) > 0 {
			body += "\n\n## Rejected alternatives\n\n"
			for _, r := range rejected {
				body += "- " + r + "\n"
			}
		}
		if prior.ADRPath != "" {
			body += "\n\nSupersedes: [" + filepath.Base(prior.ADRPath) + "](" + filepath.Base(prior.ADRPath) + ")\n"
		}
		adrPath, err = architecture.CreateADR(p.Entry.Path, architecture.ADRRequest{
			Slug:   "readopt-" + archItem.Slug,
			Title:  fmt.Sprintf("Re-adopt %s with %s", archItem.Title, stackItem.Title),
			Status: "approved",
			Body:   body,
		})
		if err == nil && prior.ADRPath != "" {
			supersededPath = prior.ADRPath
			err = architecture.Supersede(p.Entry.Path, prior.ADRPath, adrPath)
		}
	default:
		// Same selection, not first run: idempotent re-author of ADR-0001, no new ADR (NFR-2).
		adrPath, err = architecture.WriteADR0001(p.Entry.Path, archItem.Title, stackItem.Title, qaTrail, rejected)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	summaryPath, err := architecture.WriteSummary(p.Entry.Path, architecture.SummaryInput{
		Architecture:         result.PromotedArchitecture,
		TechStack:            result.PromotedTechStack,
		BreakingRequirements: req.BreakingRequirements,
		QA:                   req.QA,
		ADRPaths:             []string{adrPath},
		StandardPaths:        listStandards(p.Entry.Path),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	// Synchronous re-index of everything this commit touched (NFR-2), mirroring
	// the existing promote handler.
	reindexPath(p, result.PromotedArchitecture)
	reindexPath(p, result.PromotedTechStack)
	for _, archivedPath := range result.Archived {
		reindexPath(p, archivedPath)
	}
	reindexPath(p, adrPath)
	if supersededPath != "" {
		reindexPath(p, supersededPath)
	}
	reindexPath(p, summaryPath)

	if user := userFromCtx(r.Context()); user != nil {
		_ = architecture.ClearWizardState(s.projectRuntimeDir(p), user.Email)
	}

	p.Hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"action":                "architecture_wizard_commit",
			"promoted_architecture": result.PromotedArchitecture,
			"promoted_tech_stack":   result.PromotedTechStack,
			"archived":              result.Archived,
			"adr_path":              adrPath,
			"superseded_adr_path":   supersededPath,
			"summary_path":          summaryPath,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"promoted_architecture": result.PromotedArchitecture,
		"promoted_tech_stack":   result.PromotedTechStack,
		"archived":              result.Archived,
		"adr_path":              adrPath,
		"superseded_adr_path":   supersededPath,
		"summary_path":          summaryPath,
	})
}

// findCatalogItem finds the catalog entry whose repo-relative path matches
// catalogRelPath (relative to lifecycle/architecture/, e.g.
// "architectures/modular-monolith.md").
func findCatalogItem(items []architecture.CatalogItem, catalogRelPath string) (architecture.CatalogItem, bool) {
	want := path.Join("lifecycle/architecture", catalogRelPath)
	for _, it := range items {
		if it.Path == want {
			return it, true
		}
	}
	return architecture.CatalogItem{}, false
}

// rejectedAlternatives re-runs Recommend over the submitted answers and
// returns the titles of every candidate other than the chosen architecture
// (FR-14's "ranked alternatives that were rejected"). A Recommend error is
// swallowed — the rejected-alternatives list is supplementary context for
// the ADR body, never a reason to fail the commit.
func rejectedAlternatives(arches []architecture.CatalogItem, wizardCfg config.ArchitectureWizardConfig, answers []architecture.Answer, chosenSlug string) []string {
	recs, _, err := architecture.Recommend(arches, wizardCfg, answers)
	if err != nil {
		return nil
	}
	var rejected []string
	for _, rec := range recs {
		if rec.Item.Slug != chosenSlug {
			rejected = append(rejected, rec.Item.Title)
		}
	}
	return rejected
}

// renderQATrail renders the wizard's Q&A trail as a markdown section for
// embedding in an ADR body and architecture-summary.md (FR-15).
func renderQATrail(qa []architecture.QAPair) string {
	if len(qa) == 0 {
		return "## Selection Q&A\n\nNo questions were answered.\n"
	}
	var sb strings.Builder
	sb.WriteString("## Selection Q&A\n\n")
	for _, pair := range qa {
		sb.WriteString("- **Q:** " + pair.Question + "\n")
		sb.WriteString("  **A:** " + pair.Answer + "\n")
	}
	return sb.String()
}

// listStandards returns the repo-relative paths of any seeded standards
// under lifecycle/architecture/standards/ — empty until
// [[architecture-templates]]'s standards seed set is built (a recorded
// cross-lineage dependency, not a blocker for this milestone).
func listStandards(projectRoot string) []string {
	dir := filepath.Join(projectRoot, "lifecycle", "architecture", "standards")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		out = append(out, path.Join("lifecycle/architecture/standards", e.Name()))
	}
	sort.Strings(out)
	return out
}

// scaffoldNotAvailableMessage is returned whenever no Scaffolder is
// registered — i.e. always, until [[architecture-templates]] §4 /
// [[agent-directives-generation]] land. The core wizard flow (M6) never
// depends on this seam (FR-17/FR-18).
const scaffoldNotAvailableMessage = "scaffolding is not yet available — see lifecycle/requirements/agent-directives-generation.md"

// handleGetWizardScaffold handles GET /api/p/{project}/architecture/wizard/scaffold
func (s *Server) handleGetWizardScaffold(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	scaffolder := architecture.ActiveScaffolder()
	if scaffolder == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "message": scaffoldNotAvailableMessage})
		return
	}

	archSlug := r.URL.Query().Get("architecture")
	stackSlug := r.URL.Query().Get("tech_stack")
	steps, ok := scaffolder.Available(archSlug, stackSlug)
	writeJSON(w, http.StatusOK, map[string]any{"available": ok, "steps": steps})
}

// handleRunWizardScaffold handles POST /api/p/{project}/architecture/wizard/scaffold
func (s *Server) handleRunWizardScaffold(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RoleProductOwner) {
		return
	}

	var req struct {
		ArchitectureSlug string                        `json:"architecture"`
		TechStackSlug    string                        `json:"tech_stack"`
		Choices          []architecture.ScaffoldChoice `json:"choices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	scaffolder := architecture.ActiveScaffolder()
	if scaffolder == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "message": scaffoldNotAvailableMessage})
		return
	}

	result, err := scaffolder.Run(p.Entry.Path, req.ArchitectureSlug, req.TechStackSlug, req.Choices)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("scaffold_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": true, "result": result})
}
