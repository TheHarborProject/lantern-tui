package main

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

type workspaceFixture struct {
	app        *App
	controller *AuditController
	components *ListPanel
	states     *ListPanel
	checks     *ListPanel
	runs       *TextPanel
	queue      *TextPanel
	config     *TextPanel
}

func newWorkspaceFixture() workspaceFixture {
	components := NewListPanel("components", "Components", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	runs := NewTextPanel("runs", "Runs", "", "", true)
	queue := NewTextPanel("queue", "Queue", "", "", true)
	config := NewTextPanel("config", "Config", "", "", true)
	runs.SetContent("No audit runs loaded.\n")
	queue.SetContent("Audit queue workspace is not implemented yet.\n")
	config.SetContent("No Lantern configuration loaded.\n")
	controller := NewAuditController(fixtureComponents(), components, states, checks, evidence)
	workspaces := []Workspace{
		{ID: "audit", Name: "Audit", Components: []UIComponent{components, checks, states, evidence}, Layout: &boxlayout.Box{Children: []*boxlayout.Box{{Window: "components"}, {Window: "checks"}, {Window: "states"}, {Window: "evidence"}}}},
		{ID: "runs", Name: "Runs", Components: []UIComponent{runs}, Layout: &boxlayout.Box{Window: "runs"}},
		{ID: "queue", Name: "Queue", Components: []UIComponent{queue}, Layout: &boxlayout.Box{Window: "queue"}},
		{ID: "config", Name: "Config", Components: []UIComponent{config}, Layout: &boxlayout.Box{Window: "config"}},
	}
	header := NewAppHeader("appheader", "lantern-tui")
	header.SetWorkspaces(workspaces, 0)
	app := &App{
		components: []UIComponent{components, checks, states, evidence, runs, queue, config},
		statusBar:  NewStatusBar("statusbar", AppConfig{}), header: header,
		workspaces: workspaces, focusModePanels: make(map[string]FocusModeCapability),
	}
	for _, capability := range controller.FocusModeCapabilities() {
		app.RegisterFocusModeCapability(capability)
	}
	return workspaceFixture{app, controller, components, states, checks, runs, queue, config}
}

func TestNumberKeysSwitchWorkspaces(t *testing.T) {
	fixture := newWorkspaceFixture()
	for index, key := range []rune{'1', '2', '3', '4'} {
		binding := fixture.app.workspaceKeybinding(index, key)
		if binding.Key != key {
			t.Fatalf("workspace %d binding key = %#v, want %q", index, binding.Key, key)
		}
		if err := binding.Handler(nil, nil); err != nil {
			t.Fatalf("workspace %d handler returned %v", index, err)
		}
		if fixture.app.activeWorkspace != index {
			t.Fatalf("active workspace = %d, want %d", fixture.app.activeWorkspace, index)
		}
	}
}

func TestAuditStateSurvivesWorkspaceRoundTrip(t *testing.T) {
	fixture := newWorkspaceFixture()
	fixture.components.SetSelectedIndex(0)
	fixture.states.SetSelectedIndex(1)
	fixture.checks.SetSelectedIndex(1)
	fixture.controller.auditQueue["ui/button"] = false

	fixture.app.SwitchWorkspace(1)
	fixture.app.SwitchWorkspace(0)

	if fixture.controller.selectedComponent != 0 || fixture.controller.selectedState != 1 || fixture.controller.selectedCheck != 1 {
		t.Fatalf("audit selections changed: component=%d state=%d check=%d", fixture.controller.selectedComponent, fixture.controller.selectedState, fixture.controller.selectedCheck)
	}
	if fixture.controller.auditQueue["ui/button"] {
		t.Fatal("audit queue state changed during workspace round trip")
	}
}

func TestPanelFocusNavigationDoesNotSwitchWorkspace(t *testing.T) {
	fixture := newWorkspaceFixture()
	fixture.app.currentFocusIdx = 0
	fixture.app.FocusNext()
	if fixture.app.activeWorkspace != 0 || fixture.app.currentFocusIdx != 1 {
		t.Fatalf("FocusNext changed workspace=%d focus=%d", fixture.app.activeWorkspace, fixture.app.currentFocusIdx)
	}
	fixture.app.FocusPrev()
	if fixture.app.activeWorkspace != 0 || fixture.app.currentFocusIdx != 0 {
		t.Fatalf("FocusPrev changed workspace=%d focus=%d", fixture.app.activeWorkspace, fixture.app.currentFocusIdx)
	}
}

func TestWorkspaceSwitchIsIgnoredDuringFocusMode(t *testing.T) {
	fixture := newWorkspaceFixture()
	fixture.app.currentFocusIdx = 0
	fixture.app.EnterFocusMode()
	if fixture.app.SwitchWorkspace(1) || fixture.app.activeWorkspace != 0 {
		t.Fatal("workspace switched while Focus Mode was active")
	}
	fixture.app.ExitFocusMode()
	if !fixture.app.SwitchWorkspace(1) || fixture.app.activeWorkspace != 1 {
		t.Fatal("workspace did not switch after leaving Focus Mode")
	}
}

func TestPlaceholderWorkspacesAndHeader(t *testing.T) {
	fixture := newWorkspaceFixture()
	if fixture.runs.content != "No audit runs loaded.\n" {
		t.Fatalf("Runs placeholder = %q", fixture.runs.content)
	}
	if fixture.config.content != "No Lantern configuration loaded.\n" {
		t.Fatalf("Config placeholder = %q", fixture.config.content)
	}
	fixture.app.SwitchWorkspace(2)
	ids := layoutWindowIDs(fixture.app.buildLayoutTree())
	if strings.Join(ids, ",") != "appheader,queue,statusbar" {
		t.Fatalf("Queue layout windows = %v", ids)
	}
	header := fixture.app.header.Content()
	if !strings.Contains(header, "lantern-tui    Accessibility audit explorer") || !strings.Contains(header, "[3 Queue]") {
		t.Fatalf("header did not show title and active Queue tab: %q", header)
	}
}

func TestNormalStatusHintsIncludeWorkspaceShortcuts(t *testing.T) {
	fixture := newWorkspaceFixture()
	if got := fixture.app.formatKeybindingsForStatusBar(); !strings.Contains(got, "1-4: tabs") {
		t.Fatalf("status hints = %q, want workspace shortcuts", got)
	}
}
