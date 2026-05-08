// Package workflow implements the artifact state machine from §6 of the spec.
package workflow

import (
	"github.com/kaos-control/kaos-control/internal/artifact"
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
	types []string // empty means "any artifact type"; non-empty restricts the rule to listed types
}

// defaultRules implement the spec §6.2 transition matrix.
var defaultRules = []rule{
	{from: "draft", to: "clarifying", roles: []string{"product-owner", "analyst"}},
	{from: "clarifying", to: "planning", roles: []string{"product-owner", "reviewer", "analyst"}},
	{from: "planning", to: "in-development", roles: []string{"approver"}},
	{from: "in-development", to: "in-qa", roles: []string{"backend-developer", "frontend-developer", "test-developer"}},
	{from: "in-qa", to: "approved", roles: []string{"qa"}},
	{from: "approved", to: "done", roles: []string{"approver"}},
	// Terminal fallbacks: clarifying ↔ draft (so product-owner / analyst can retract)
	{from: "clarifying", to: "draft", roles: []string{"product-owner", "analyst"}},
	// any → rejected / abandoned
	{from: "", to: "rejected", roles: []string{"reviewer"}},
	{from: "", to: "abandoned", roles: []string{"product-owner", "approver"}},
	// Block-on-questions: any agent role can self-block when stuck on missing input.
	{from: "", to: "blocked", roles: []string{"analyst", "backend-developer", "frontend-developer", "test-developer", "qa"}},
	// Unblock: product-owner (and analyst, who can also re-scope) sends it back to draft after answering.
	{from: "blocked", to: "draft", roles: []string{"product-owner", "analyst"}},
	// System actor: machine-initiated block/unblock transitions (auto-block on open questions).
	{from: "", to: "blocked", roles: []string{"system"}},
	{from: "blocked", to: "draft", roles: []string{"system"}},
	// Test artifact lifecycle: approved → in-qa (qa initiates) and in-qa → approved (system on success).
	{from: "approved", to: "in-qa", roles: []string{"qa"}, types: []string{"test"}},
	{from: "in-qa", to: "approved", roles: []string{"system"}, types: []string{"test"}},
}

// New builds an Engine, overlaying project-level overrides on the default matrix.
func New(transitions []config.Transition) *Engine {
	rules := make([]rule, len(defaultRules))
	copy(rules, defaultRules)

	for _, t := range transitions {
		matched := false
		// A project-level transition overrides a default rule only when both
		// (from, to) and the types restriction match exactly. Type-scoped
		// transitions are always appended as new rules when no exact match exists.
		for i, r := range rules {
			if r.from == t.From && r.to == t.To && typeSlicesEqual(r.types, t.Types) {
				rules[i].roles = t.Roles
				matched = true
				break
			}
		}
		if !matched {
			rules = append(rules, rule{from: t.From, to: t.To, roles: t.Roles, types: t.Types})
		}
	}
	return &Engine{rules: rules}
}

// typeSlicesEqual reports whether two type slices contain the same elements
// (order-insensitive, nil and empty are considered equal).
func typeSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if !set[v] {
			return false
		}
	}
	return true
}

// HasProductOwner reports whether the role list contains the product-owner role.
// Product-owner is the project superuser and is exempt from transition rules and
// plan-readiness gates so they can recover from data drift or smooth over edge cases.
func HasProductOwner(roles []string) bool {
	for _, r := range roles {
		if r == "product-owner" {
			return true
		}
	}
	return false
}

// ruleMatchesType reports whether a rule applies to the given artifact type.
// Rules with an empty types list apply to all artifact types.
// Rules with a non-empty types list only apply when artifactType is in the list.
// When artifactType is empty (caller does not know the type), type-restricted
// rules are never matched.
func ruleMatchesType(r rule, artifactType string) bool {
	if len(r.types) == 0 {
		return true // type-agnostic rule — applies to everything
	}
	if artifactType == "" {
		return false // type-restricted rule but caller has no type context
	}
	for _, t := range r.types {
		if t == artifactType {
			return true
		}
	}
	return false
}

// CanTransition reports whether a holder of any of the given roles may advance
// an artifact whose current status is 'from' to status 'to'.
// artifactType is the type field of the artifact (e.g. "test", "requirement").
// Pass an empty string when the type is unknown; in that case type-restricted
// rules are not considered.
func (e *Engine) CanTransition(from, to string, userRoles []string, artifactType string) bool {
	// Self-transitions are no-ops and never permitted, even for product-owner.
	// (Both for correctness — a transition that doesn't change anything is a
	// bug — and to keep the API behaviour sensible: the handler would write
	// the same value back to disk and emit a confusing "X → X" event.)
	if from == to {
		return false
	}
	// Targets outside the documented status vocabulary are never permitted —
	// product-owner included. This stops typos and stray automation from
	// putting an artifact into a status the rest of the system can't reason
	// about.
	if !artifact.KnownStatuses[to] {
		return false
	}
	if HasProductOwner(userRoles) {
		return true
	}
	for _, r := range e.rules {
		if r.to != to {
			continue
		}
		// Empty 'from' means the rule applies to any source status.
		if r.from != "" && r.from != from {
			continue
		}
		if !ruleMatchesType(r, artifactType) {
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
// artifactType filters type-restricted rules; pass an empty string to receive
// only type-agnostic allowed targets.
func (e *Engine) AllowedTargets(from string, userRoles []string, artifactType string) []string {
	seen := map[string]bool{}
	var out []string
	if HasProductOwner(userRoles) {
		for _, r := range e.rules {
			if !seen[r.to] {
				seen[r.to] = true
				out = append(out, r.to)
			}
		}
		return out
	}
	for _, r := range e.rules {
		if r.from != "" && r.from != from {
			continue
		}
		if !ruleMatchesType(r, artifactType) {
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
