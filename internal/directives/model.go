// SPDX-License-Identifier: AGPL-3.0-or-later

// Package directives generates the AGENTS.md-primary directive set
// (AGENTS.md canonical, CLAUDE.md/GEMINI.md pointers) and patches the six
// standard agent prompt templates in lifecycle/config.yaml, tuned to a
// project's promoted tech-stack.
package directives

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/architecture"
)

// DirectiveModel is the single shared content model rendered into AGENTS.md
// (canonical, FR-4/FR-5) and used to patch the standard agents' config.yaml
// entries (FR-6/FR-7) — one source of truth for both (FR-1).
type DirectiveModel struct {
	ProjectName string
	Language    string
	Stack       architecture.StackProfile
	StackTitle  string
	RepoLayout  []architecture.RepoLayoutEntry
	// ArchitecturePointer is true once a stack has actually been promoted,
	// so the rendered directive can point at the concrete chosen
	// architecture/stack rather than the generic pre-wizard message.
	ArchitecturePointer bool
}

// BuildModel assembles a DirectiveModel for projectRoot from its promoted
// tech-stack (via architecture.LoadPromotedStackProfile). If no stack has
// been promoted yet it returns architecture.ErrNoPromotedStack (wrapped, so
// errors.Is still matches) — callers fall back to the generic pre-wizard
// directive (see GenericAgents).
func BuildModel(projectRoot string) (DirectiveModel, error) {
	profile, title, err := architecture.LoadPromotedStackProfile(projectRoot)
	if err != nil {
		if errors.Is(err, architecture.ErrNoPromotedStack) {
			return DirectiveModel{}, err
		}
		return DirectiveModel{}, fmt.Errorf("loading promoted stack profile: %w", err)
	}

	return DirectiveModel{
		ProjectName:         filepath.Base(filepath.Clean(projectRoot)),
		Stack:               profile,
		StackTitle:          title,
		RepoLayout:          profile.RepoLayout,
		ArchitecturePointer: true,
	}, nil
}
