package main

import (
	"testing"

	"github.com/jesseduffield/gocui"
)

type auditFocusFixture struct {
	app        *App
	controller *AuditController
	components *ListPanel
	states     *ListPanel
	checks     *ListPanel
	evidence   *TextPanel
}

func newAuditFocusFixture() auditFocusFixture {
	components := NewListPanel("components", "Components", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	controller := NewAuditController(mockComponents(), components, states, checks, evidence)
	app := &App{
		components:      []UIComponent{components, checks, states, evidence},
		statusBar:       NewStatusBar("statusbar", AppConfig{}),
		focusModePanels: make(map[string]FocusModeCapability),
	}
	for _, capability := range controller.FocusModeCapabilities() {
		app.RegisterFocusModeCapability(capability)
	}
	return auditFocusFixture{app, controller, components, states, checks, evidence}
}

func TestAllAuditPanelsEnterAndRestoreFocusMode(t *testing.T) {
	fixture := newAuditFocusFixture()
	for index, panelID := range []string{"components", "checks", "states", "evidence"} {
		fixture.app.currentFocusIdx = index
		if !fixture.app.EnterFocusMode() {
			t.Fatalf("%s did not enter focus mode", panelID)
		}
		if fixture.app.focusMode.PanelID != panelID {
			t.Fatalf("focused panel = %q, want %q", fixture.app.focusMode.PanelID, panelID)
		}
		if err := fixture.app.handleEscape(); err != nil {
			t.Fatalf("Escape from %s returned %v", panelID, err)
		}
		if fixture.app.focusMode.Active {
			t.Fatalf("%s remained in focus mode after Escape", panelID)
		}
		if got := fixture.app.getFocusableComponents()[fixture.app.currentFocusIdx].ID(); got != panelID {
			t.Fatalf("focus restored to %q, want %q", got, panelID)
		}
	}
}

func TestComponentsQueueTogglePersists(t *testing.T) {
	fixture := newAuditFocusFixture()
	fixture.app.currentFocusIdx = 0
	fixture.components.SetSelectedIndex(0)
	fixture.app.EnterFocusMode()
	if got := fixture.components.items[0].DisplayText; got[:3] != "[x]" {
		t.Fatalf("initial focused component row = %q, want queued marker", got)
	}
	space := fixture.app.focusModeSpaceKeybinding()
	if space.Key != gocui.KeySpace {
		t.Fatalf("Space binding key = %#v, want gocui.KeySpace", space.Key)
	}
	if err := space.Handler(nil, nil); err != nil {
		t.Fatalf("Space handler returned %v", err)
	}
	if fixture.controller.auditQueue["ui/button"] {
		t.Fatal("Button remained in audit queue after toggle")
	}
	if got := fixture.components.items[0].DisplayText; got[:3] != "[ ]" {
		t.Fatalf("marker after Space = %q, want unqueued marker immediately", got)
	}
	if got := fixture.components.GetSelectedIndex(); got != 0 {
		t.Fatalf("selection after Space = %d, want 0", got)
	}
	fixture.app.ExitFocusMode()
	if fixture.controller.auditQueue["ui/button"] {
		t.Fatal("Escape did not preserve queue state")
	}
	fixture.app.EnterFocusMode()
	if got := fixture.components.items[0].DisplayText; got[:3] != "[ ]" {
		t.Fatalf("queue marker after re-entry = %q, want persisted unqueued marker", got)
	}
}

func TestFocusedStateSelectionPersists(t *testing.T) {
	fixture := newAuditFocusFixture()
	fixture.app.currentFocusIdx = 2
	fixture.app.EnterFocusMode()
	fixture.states.SetSelectedIndex(1)
	fixture.app.ExitFocusMode()
	if fixture.controller.selectedState != 1 || fixture.states.GetSelectedIndex() != 1 {
		t.Fatalf("state selection = controller %d panel %d, want 1", fixture.controller.selectedState, fixture.states.GetSelectedIndex())
	}
}

func TestFocusedCheckSelectionPersistsAndSynchronizesEvidence(t *testing.T) {
	fixture := newAuditFocusFixture()
	fixture.app.currentFocusIdx = 1
	fixture.app.EnterFocusMode()
	fixture.checks.SetSelectedIndex(1)
	if fixture.controller.currentCheck().ID != "chk-button-name" {
		t.Fatalf("current check = %q, want chk-button-name", fixture.controller.currentCheck().ID)
	}
	if fixture.evidence.content == "" || fixture.controller.selectedCheck != 1 {
		t.Fatal("Evidence did not synchronize with focused check selection")
	}
	fixture.app.ExitFocusMode()
	if fixture.checks.GetSelectedIndex() != 1 {
		t.Fatalf("check selection after exit = %d, want 1", fixture.checks.GetSelectedIndex())
	}
}

func TestEvidenceFocusBehaviorRemainsUnchanged(t *testing.T) {
	fixture := newAuditFocusFixture()
	originalContent := fixture.evidence.content
	fixture.evidence.SetOrigin(0, 4)
	fixture.app.currentFocusIdx = 3
	fixture.app.EnterFocusMode()
	fixture.app.ExitFocusMode()
	if fixture.evidence.content != originalContent {
		t.Fatal("Evidence content changed while entering or leaving focus mode")
	}
	if _, origin := fixture.evidence.GetOrigin(); origin != 4 {
		t.Fatalf("Evidence origin = %d, want 4", origin)
	}
}

func TestFocusedHintsAndKeybindingsArePanelSpecific(t *testing.T) {
	fixture := newAuditFocusFixture()
	space := fixture.app.focusModeSpaceKeybinding()
	wants := []string{
		"q: quit | Esc: back | ↑/↓: navigate | Space: toggle",
		"q: quit | Esc: back | ↑/↓: navigate",
		"q: quit | Esc: back | ↑/↓: navigate",
		"q: quit | Esc: back | ↑/↓: scroll",
	}
	for index, want := range wants {
		fixture.app.currentFocusIdx = index
		fixture.app.EnterFocusMode()
		if got := fixture.app.formatKeybindingsForStatusBar(); got != want {
			t.Errorf("panel %d hints = %q, want %q", index, got, want)
		}
		queueBefore := fixture.controller.auditQueue["ui/button"]
		if err := space.Handler(nil, nil); err != nil {
			t.Errorf("Space on focused panel %d returned %v", index, err)
		}
		if index != 0 && fixture.controller.auditQueue["ui/button"] != queueBefore {
			t.Errorf("Space leaked into focused panel %d", index)
		}
		fixture.app.ExitFocusMode()
	}
}
