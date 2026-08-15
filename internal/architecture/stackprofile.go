// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"gopkg.in/yaml.v3"
)

// ErrNoStackProfile is returned by ParseStackProfile when the markdown has
// no fenced `stack_profile:` YAML block. A stack without a profile is a
// hard error for directive/config generation.
var ErrNoStackProfile = errors.New("no stack_profile block found")

// ErrNoPromotedStack is returned by LoadPromotedStackProfile when no
// tech-stack has been promoted to the lifecycle/architecture/ root yet.
var ErrNoPromotedStack = errors.New("no tech-stack has been promoted yet")

// RoleProfile is the stack-specific configuration for one standard agent
// role, as embedded in a tech-stack catalog artefact's stack_profile block.
type RoleProfile struct {
	// Required, when explicitly false, means this role has no place in the
	// stack (e.g. backend-developer for a static site) and should be
	// disabled rather than configured. Nil (the common case) means true —
	// see IsRequired.
	Required   *bool    `yaml:"required"`
	WritePaths []string `yaml:"write_paths"`
	Build      string   `yaml:"build"`
	Lint       string   `yaml:"lint"`
	Test       string   `yaml:"test"`
	Note       string   `yaml:"note,omitempty"`
}

// IsRequired reports whether this role is required for the stack. Absent
// (nil) is treated as required — a role only opts out with `required: false`.
func (r RoleProfile) IsRequired() bool {
	return r.Required == nil || *r.Required
}

// RepoLayoutEntry is one row of the stack's repo layout table.
type RepoLayoutEntry struct {
	Path string `yaml:"path"`
	Note string `yaml:"note"`
}

// StackProfile is the machine-readable profile embedded at the end of a
// tech-stack catalog artefact (see e.g.
// lifecycle/architecture/tech-stacks/go-vue.md), consumed by kaos-control to
// tune generated directive files and the standard agents' config.yaml
// entries.
type StackProfile struct {
	Run        string                 `yaml:"run"`
	RepoLayout []RepoLayoutEntry      `yaml:"repo_layout"`
	Roles      map[string]RoleProfile `yaml:"roles"`
}

// yamlFenceRe matches a fenced ```yaml / ```yml code block and captures its
// interior.
var yamlFenceRe = regexp.MustCompile("(?s)```ya?ml\\s*\\n(.*?)\\n```")

// stackProfileProbe decodes only the stack_profile key, so a document can be
// checked for its presence without assuming any fenced yaml block found is
// the right one.
type stackProfileProbe struct {
	StackProfile *StackProfile `yaml:"stack_profile"`
}

// ParseStackProfile extracts the fenced ```yaml block whose top-level key is
// `stack_profile:` from a tech-stack catalog artefact's markdown bytes and
// unmarshals it. Inline `# comments` are tolerated (yaml.v3 handles them
// natively). Returns ErrNoStackProfile if no such block is found.
func ParseStackProfile(mdBytes []byte) (StackProfile, error) {
	for _, m := range yamlFenceRe.FindAllSubmatch(mdBytes, -1) {
		var probe stackProfileProbe
		if err := yaml.Unmarshal(m[1], &probe); err != nil {
			continue
		}
		if probe.StackProfile != nil {
			return *probe.StackProfile, nil
		}
	}
	return StackProfile{}, ErrNoStackProfile
}

// LoadPromotedStackProfile locates the promoted tech-stack root copy under
// projectRoot's lifecycle/architecture/ (a root-level `type: tech-stack`
// file — see Promote), reads and parses its stack_profile block, and returns
// the profile plus the stack's title. Returns ErrNoPromotedStack if nothing
// is promoted yet (generation is gated on promotion).
func LoadPromotedStackProfile(projectRoot string) (profile StackProfile, title string, err error) {
	promoted, err := currentlyPromoted(projectRoot, "tech-stack")
	if err != nil {
		return StackProfile{}, "", fmt.Errorf("scanning promoted tech-stack copies: %w", err)
	}
	if len(promoted) == 0 {
		return StackProfile{}, "", ErrNoPromotedStack
	}

	relPath := promoted[0]
	absPath := filepath.Join(projectRoot, filepath.FromSlash(relPath))
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return StackProfile{}, "", fmt.Errorf("reading promoted tech-stack %q: %w", relPath, err)
	}

	profile, err = ParseStackProfile(raw)
	if err != nil {
		return StackProfile{}, "", fmt.Errorf("parsing stack_profile from %q: %w", relPath, err)
	}

	a := artifact.Parse(raw, relPath, time.Time{})
	return profile, a.FM.Title, nil
}
