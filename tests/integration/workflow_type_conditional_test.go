//go:build integration

package integration

// Tests for the type-aware CanTransition and AllowedTargets functions added in
// the test-artifact-status-lifecycle backend plan (Milestone 1).
//
// These tests call the workflow engine directly; no HTTP server or filesystem
// I/O is required so they run under `go test ./... -short` alongside unit tests.
//
// Test plan: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md §Milestone 1

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/workflow"
)

// TC1: approved → in-qa allowed for qa role on test type.
func TestWorkflowTypeConditional_ApprovedToInQA_QAOnTest(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("approved", "in-qa", []string{"qa"}, "test") {
		t.Error("CanTransition('approved','in-qa',['qa'],'test') must be true: type-conditional rule exists for qa on test")
	}
}

// TC2: approved → in-qa denied for qa role on non-test type (rule is type-restricted to "test").
func TestWorkflowTypeConditional_ApprovedToInQA_QAOnRequirement(t *testing.T) {
	e := workflow.New(nil)
	if e.CanTransition("approved", "in-qa", []string{"qa"}, "requirement") {
		t.Error("CanTransition('approved','in-qa',['qa'],'requirement') must be false: rule is type-restricted to 'test'")
	}
}

// TC3: approved → in-qa denied for non-qa role on test type (wrong role).
func TestWorkflowTypeConditional_ApprovedToInQA_BackendDevOnTest(t *testing.T) {
	e := workflow.New(nil)
	if e.CanTransition("approved", "in-qa", []string{"backend-developer"}, "test") {
		t.Error("CanTransition('approved','in-qa',['backend-developer'],'test') must be false: only qa role is permitted")
	}
}

// TC4: in-qa → approved allowed for system role on test type.
func TestWorkflowTypeConditional_InQAToApproved_SystemOnTest(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("in-qa", "approved", []string{"system"}, "test") {
		t.Error("CanTransition('in-qa','approved',['system'],'test') must be true: post-run reset rule exists")
	}
}

// TC5: in-qa → approved denied for system role on non-test type (rule is type-restricted to "test").
func TestWorkflowTypeConditional_InQAToApproved_SystemOnRequirement(t *testing.T) {
	e := workflow.New(nil)
	if e.CanTransition("in-qa", "approved", []string{"system"}, "requirement") {
		t.Error("CanTransition('in-qa','approved',['system'],'requirement') must be false: system rule restricted to 'test' type")
	}
}

// TC6: in-qa → approved for qa role on non-test types is unchanged (the qa rule has no type restriction).
func TestWorkflowTypeConditional_InQAToApproved_QAOnRequirement(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("in-qa", "approved", []string{"qa"}, "requirement") {
		t.Error("CanTransition('in-qa','approved',['qa'],'requirement') must be true: existing qa rule has no type restriction")
	}
}

// TC7: in-development → in-qa for backend-developer is unchanged (existing rule, no type restriction).
func TestWorkflowTypeConditional_InDevelopmentToInQA_BackendDev(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("in-development", "in-qa", []string{"backend-developer"}, "requirement") {
		t.Error("CanTransition('in-development','in-qa',['backend-developer'],'requirement') must be true: existing rule unchanged")
	}
}

// TC8: AllowedTargets includes in-qa for test artifacts in approved status with qa role.
func TestWorkflowTypeConditional_AllowedTargets_InQAForTestWithQA(t *testing.T) {
	e := workflow.New(nil)
	targets := e.AllowedTargets("approved", []string{"qa"}, "test")
	for _, target := range targets {
		if target == "in-qa" {
			return // found — test passes
		}
	}
	t.Errorf("AllowedTargets('approved',['qa'],'test') must include 'in-qa'; got: %v", targets)
}

// TC9: AllowedTargets excludes in-qa for non-test artifacts in approved status with qa role.
func TestWorkflowTypeConditional_AllowedTargets_NoInQAForRequirementWithQA(t *testing.T) {
	e := workflow.New(nil)
	targets := e.AllowedTargets("approved", []string{"qa"}, "requirement")
	for _, target := range targets {
		if target == "in-qa" {
			t.Errorf("AllowedTargets('approved',['qa'],'requirement') must NOT include 'in-qa'; got: %v", targets)
			return
		}
	}
	// in-qa absent — test passes
}

// TC10: product-owner bypasses all type restrictions.
func TestWorkflowTypeConditional_ProductOwnerBypasses(t *testing.T) {
	e := workflow.New(nil)
	if !e.CanTransition("approved", "in-qa", []string{"product-owner"}, "requirement") {
		t.Error("CanTransition('approved','in-qa',['product-owner'],'requirement') must be true: product-owner is superuser")
	}
}
