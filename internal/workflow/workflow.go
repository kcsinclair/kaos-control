// Package workflow implements the artifact state machine from §6 of the spec.
package workflow

import (
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/index"
)

// Engine is an immutable state machine for one project.
type Engine struct {
	rules []rule
}

type rule struct {
	from  string   // empty means "any non-terminal status"
	to    string
	roles []string
}

// defaultRules implement the spec §6.2 transition matrix.
var defaultRules = []rule{
	{from: "draft", to: "clarifying", roles: []string{"product-owner"}},
	{from: "clarifying", to: "planning", roles: []string{"product-owner", "reviewer"}},
	{from: "planning", to: "in-development", roles: []string{"approver"}},
	{from: "in-development", to: "in-qa", roles: []string{"developer"}},
	{from: "in-qa", to: "approved", roles: []string{"qa"}},
	{from: "approved", to: "done", roles: []string{"approver"}},
	// Terminal fallbacks: clarifying ↔ draft (so product-owner can retract)
	{from: "clarifying", to: "draft", roles: []string{"product-owner"}},
	// any → rejected / abandoned
	{from: "", to: "rejected", roles: []string{"reviewer"}},
	{from: "", to: "abandoned", roles: []string{"product-owner", "approver"}},
}

// New builds an Engine, overlaying project-level overrides on the default matrix.
func New(transitions []config.Transition) *Engine {
	rules := make([]rule, len(defaultRules))
	copy(rules, defaultRules)

	for _, t := range transitions {
		matched := false
		for i, r := range rules {
			if r.from == t.From && r.to == t.To {
				rules[i].roles = t.Roles
				matched = true
				break
			}
		}
		if !matched {
			rules = append(rules, rule{from: t.From, to: t.To, roles: t.Roles})
		}
	}
	return &Engine{rules: rules}
}

// CanTransition reports whether a holder of any of the given roles may advance
// an artifact whose current status is 'from' to status 'to'.
func (e *Engine) CanTransition(from, to string, userRoles []string) bool {
	for _, r := range e.rules {
		if r.to != to {
			continue
		}
		// Empty 'from' means the rule applies to any source status.
		if r.from != "" && r.from != from {
			continue
		}
		for _, allowed := range r.roles {
			for _, ur := range userRoles {
				if ur == allowed {
					return true
				}
			}
		}
	}
	return false
}

// AllowedTargets returns every status the caller may transition 'from' to.
func (e *Engine) AllowedTargets(from string, userRoles []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range e.rules {
		if r.from != "" && r.from != from {
			continue
		}
		for _, allowed := range r.roles {
			for _, ur := range userRoles {
				if ur == allowed && !seen[r.to] {
					seen[r.to] = true
					out = append(out, r.to)
				}
			}
		}
	}
	return out
}

// GateReady checks whether a lineage is ready to leave the 'planning' state.
// required is the list of artifact types that must each have at least one 'approved' member.
// Returns (ready, missingTypes, error).
func GateReady(idx *index.Index, lineage string, required []string) (bool, []string, error) {
	if len(required) == 0 {
		return true, nil, nil
	}
	rows, _, err := idx.List(index.Filter{Lineage: lineage, Limit: 500})
	if err != nil {
		return false, nil, err
	}
	approved := map[string]bool{}
	for _, r := range rows {
		if r.Status == "approved" {
			approved[r.Type] = true
		}
	}
	var missing []string
	for _, req := range required {
		if !approved[req] {
			missing = append(missing, req)
		}
	}
	return len(missing) == 0, missing, nil
}
