// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/agent-directives-generation-5-test.md —
// Milestone 1 (FR-2, FR-5, OQ-5): every shipped tech-stack's stack_profile
// parses without error, and required: false roles are modelled as disabled.
//
// Exact field-level parsing (go-vue, static-html-js, the no-profile error
// case, LoadPromotedStackProfile) is already covered as package-internal
// unit tests in internal/architecture/stackprofile_test.go, per this test
// plan's own "package-internal unit tests are owned by the backend plan"
// note. What isn't covered anywhere yet is the specific acceptance
// criterion "every shipped tech-stack's stack_profile parses without error
// (table-driven over all 14 files)" — added here.

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/architecture/catalogfs"
)

func TestStackProfile_AllShippedTechStacks_ParseWithoutError(t *testing.T) {
	entries, err := catalogfs.FS.ReadDir("tech-stacks")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 14 {
		t.Fatalf("expected 14 shipped tech-stacks (per the test plan's acceptance criterion), found %d: %v", len(entries), entries)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			raw, err := catalogfs.FS.ReadFile("tech-stacks/" + name)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := architecture.ParseStackProfile(raw)
			if err != nil {
				t.Fatalf("ParseStackProfile(%s): %v", name, err)
			}
			if len(profile.Roles) == 0 {
				t.Errorf("%s: stack_profile has no roles", name)
			}
		})
	}
}

func TestStackProfile_StaticHTMLJS_BackendDeveloperDisabled(t *testing.T) {
	profile, err := architecture.ParseStackProfile([]byte(catalogStackContent(t, "static-html-js.md")))
	if err != nil {
		t.Fatalf("ParseStackProfile: %v", err)
	}
	be, ok := profile.Roles["backend-developer"]
	if !ok {
		t.Fatal("missing backend-developer role")
	}
	if be.IsRequired() {
		t.Error("expected backend-developer.IsRequired() == false for static-html-js")
	}
}
