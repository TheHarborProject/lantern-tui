package main

import (
	"strings"
	"testing"
)

func newTestController() (*AuditController, *ListPanel, *ListPanel, *ListPanel, *TextPanel) {
	components := NewListPanel("components", "Components", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	controller := NewAuditController(fixtureComponents(), components, states, checks, evidence)
	return controller, components, states, checks, evidence
}

func TestComponentSelectionRefreshesDependentPanels(t *testing.T) {
	controller, components, states, checks, evidence := newTestController()

	components.SetSelectedIndex(1)

	if got := controller.currentComponent().CanonicalID; got != "ui/label" {
		t.Fatalf("selected component = %q, want ui/label", got)
	}
	if len(states.items) != 1 || states.items[0].Object.(*State).ID != "label-default" {
		t.Fatalf("states were not refreshed for Label: %#v", states.items)
	}
	if len(checks.items) != 1 || checks.items[0].Object.(*Check).ID != "chk-label-association" {
		t.Fatalf("checks were not refreshed for Label: %#v", checks.items)
	}
	if !strings.Contains(evidence.content, "lantern/label-association") {
		t.Fatalf("evidence was not refreshed for Label check: %q", evidence.content)
	}
}

func TestStateAndCheckSelectionRefreshEvidence(t *testing.T) {
	controller, _, states, checks, evidence := newTestController()

	states.SetSelectedIndex(1)
	if got := controller.currentState().ID; got != "button-icon" {
		t.Fatalf("selected state = %q, want button-icon", got)
	}
	if !strings.Contains(evidence.content, "lantern/accessible-name") {
		t.Fatalf("evidence did not follow state selection: %q", evidence.content)
	}

	checks.SetSelectedIndex(1)
	if got := controller.currentCheck().ID; got != "chk-icon-focus" {
		t.Fatalf("selected check = %q, want chk-icon-focus", got)
	}
	if !strings.Contains(evidence.content, "lantern/focus-visible") {
		t.Fatalf("evidence did not follow check selection: %q", evidence.content)
	}
}

func TestEmptyChildrenClearDependentPanels(t *testing.T) {
	_, components, states, checks, evidence := newTestController()

	components.SetSelectedIndex(3)
	if len(states.items) != 0 || len(checks.items) != 0 {
		t.Fatalf("empty component retained child data: states=%d checks=%d", len(states.items), len(checks.items))
	}
	if !strings.Contains(evidence.content, "No evidence") {
		t.Fatalf("empty component retained evidence: %q", evidence.content)
	}
}
