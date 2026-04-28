//go:build integration

package integration

import (
	"testing"
)

// rolesOnlyCfgYAML is a project config with roles but no users.
// Used by TestGetRoles_EmptyUsers to verify the API returns an empty array.
const rolesOnlyCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestGetRoles_ReturnsConfiguredRoles verifies that GET /api/p/:project/roles returns
// the roles and user bindings configured in lifecycle/config.yaml.
// Covers Milestone 1, scenario 1.
func TestGetRoles_ReturnsConfiguredRoles(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/roles", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Verify roles array matches the seeded config.
	roles, ok := data["roles"].([]any)
	if !ok {
		t.Fatalf("expected roles array in response, got %T", data["roles"])
	}
	wantRoles := []string{
		"product-owner", "analyst", "backend-developer", "frontend-developer",
		"test-developer", "qa", "reviewer", "approver",
	}
	if len(roles) != len(wantRoles) {
		t.Errorf("expected %d roles, got %d: %v", len(wantRoles), len(roles), roles)
	}
	roleSet := map[string]bool{}
	for _, r := range roles {
		if s, ok := r.(string); ok {
			roleSet[s] = true
		}
	}
	for _, want := range wantRoles {
		if !roleSet[want] {
			t.Errorf("expected role %q in response roles", want)
		}
	}

	// Verify users array contains expected email/role bindings.
	users, ok := data["users"].([]any)
	if !ok {
		t.Fatalf("expected users array in response, got %T", data["users"])
	}
	// Default config has 3 users.
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}

	// Build a map of email → roles from the response.
	userRoles := map[string][]string{}
	for _, raw := range users {
		u, _ := raw.(map[string]any)
		email, _ := u["email"].(string)
		rawRoles, _ := u["roles"].([]any)
		for _, r := range rawRoles {
			if s, ok := r.(string); ok {
				userRoles[email] = append(userRoles[email], s)
			}
		}
	}

	// Check admin@test.local has product-owner role.
	adminRoles := userRoles["admin@test.local"]
	if len(adminRoles) == 0 {
		t.Error("expected admin@test.local in users response")
	}
	adminRoleSet := map[string]bool{}
	for _, r := range adminRoles {
		adminRoleSet[r] = true
	}
	for _, want := range []string{"product-owner", "analyst", "reviewer", "approver"} {
		if !adminRoleSet[want] {
			t.Errorf("expected admin@test.local to have role %q", want)
		}
	}

	// Check dev@test.local has backend-developer role.
	devRoleSet := map[string]bool{}
	for _, r := range userRoles["dev@test.local"] {
		devRoleSet[r] = true
	}
	if !devRoleSet["backend-developer"] {
		t.Error("expected dev@test.local to have role backend-developer")
	}

	// Check qa@test.local has qa role.
	qaRoleSet := map[string]bool{}
	for _, r := range userRoles["qa@test.local"] {
		qaRoleSet[r] = true
	}
	if !qaRoleSet["qa"] {
		t.Error("expected qa@test.local to have role qa")
	}
}

// TestGetRoles_EmptyUsers verifies that when lifecycle/config.yaml has roles but no
// users list, the API returns a populated roles array and an empty (non-null) users array.
// Covers Milestone 1, scenario 2.
func TestGetRoles_EmptyUsers(t *testing.T) {
	// Use a config with roles but no users section.
	env := newTestEnvWithCfgYAML(t, nil, rolesOnlyCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/roles", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// roles should be populated.
	roles, ok := data["roles"].([]any)
	if !ok {
		t.Fatalf("expected roles array in response, got %T", data["roles"])
	}
	if len(roles) == 0 {
		t.Error("expected non-empty roles array when config has roles but no users")
	}

	// users must be an empty array, not null.
	rawUsers, exists := data["users"]
	if !exists {
		t.Fatal("expected users key in response")
	}
	users, ok := rawUsers.([]any)
	if !ok {
		t.Fatalf("expected users to be an array, got %T (value: %v)", rawUsers, rawUsers)
	}
	if len(users) != 0 {
		t.Errorf("expected empty users array, got %d entries", len(users))
	}
}

// TestGetRoles_Unauthenticated verifies that GET /roles without a valid session
// returns 401.
// Covers Milestone 1, scenario 3.
func TestGetRoles_Unauthenticated(t *testing.T) {
	env := newTestEnv(t, nil)
	// Deliberately do NOT call env.login() — no session cookie is set.

	resp := env.doRequest("GET", "/api/p/testproject/roles", nil)
	requireStatus(t, resp, 401)
	resp.Body.Close()
}
