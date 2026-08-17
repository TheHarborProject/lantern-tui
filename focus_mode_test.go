package main

import (
	"strings"
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

func newFocusModeTestApp() (*App, *TextPanel) {
	components := NewListPanel("components", "Components", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	app := &App{
		components:      []UIComponent{components, checks, states, evidence},
		currentFocusIdx: 3,
		statusBar:       NewStatusBar("statusbar", AppConfig{}),
		focusModePanels: map[string]FocusModeCapability{
			"evidence": {PanelID: "evidence", StatusHints: "q: quit | Esc: back | ↑/↓: scroll"},
		},
		layoutStrategies: []*boxlayout.Box{{
			Direction: boxlayout.COLUMN,
			Children: []*boxlayout.Box{
				{Window: "components"}, {Window: "checks"},
				{Window: "states"}, {Window: "evidence"},
			},
		}},
		GlobalKeybindings: []Keybinding{
			{DisplayName: "q", Description: "quit", Displayable: true},
			{DisplayName: "Tab", Description: "next panel", Displayable: true},
			{DisplayName: "Shift+Tab", Description: "previous panel", Displayable: true},
			{DisplayName: "Enter", Description: "focus", Displayable: true},
		},
	}
	evidence.OnFocus(nil)
	return app, evidence
}

func TestEvidenceEntersFocusMode(t *testing.T) {
	app, _ := newFocusModeTestApp()
	if !app.EnterFocusMode() {
		t.Fatal("EnterFocusMode returned false for Evidence")
	}
	if !app.focusMode.Active || app.focusMode.PanelID != "evidence" {
		t.Fatalf("focus mode = %#v, want active Evidence", app.focusMode)
	}
	if app.EnterFocusMode() {
		t.Fatal("entering an already active focus mode should be a no-op")
	}
}

func TestUnsupportedPanelsDoNotEnterFocusMode(t *testing.T) {
	app, _ := newFocusModeTestApp()
	app.currentFocusIdx = 0
	if app.EnterFocusMode() || app.focusMode.Active {
		t.Fatal("Components must not enter focus mode yet")
	}
}

func TestFocusedLayoutContainsOnlyEvidenceAndStatusBar(t *testing.T) {
	app, _ := newFocusModeTestApp()
	app.EnterFocusMode()

	ids := layoutWindowIDs(app.buildLayoutTree())
	if strings.Join(ids, ",") != "evidence,statusbar" {
		t.Fatalf("focused layout windows = %v, want [evidence statusbar]", ids)
	}
}

func TestEscapeRestoresLayoutFocusAndScroll(t *testing.T) {
	app, evidence := newFocusModeTestApp()
	evidence.SetOrigin(0, 7)
	app.EnterFocusMode()

	if err := app.handleEscape(); err != nil {
		t.Fatalf("handleEscape returned %v", err)
	}
	if app.focusMode.Active {
		t.Fatal("focus mode remained active after Escape")
	}
	if got := app.getFocusableComponents()[app.currentFocusIdx].ID(); got != "evidence" {
		t.Fatalf("focus returned to %q, want evidence", got)
	}
	if !evidence.IsFocused() {
		t.Fatal("Evidence is not focused after leaving focus mode")
	}
	if _, y := evidence.GetOrigin(); y != 7 {
		t.Fatalf("Evidence scroll origin = %d, want 7", y)
	}
	ids := layoutWindowIDs(app.buildLayoutTree())
	if len(ids) != 5 {
		t.Fatalf("restored layout windows = %v, want four panels and status bar", ids)
	}
}

func TestFocusModeQuitAndStatusHints(t *testing.T) {
	app, _ := newFocusModeTestApp()
	if normal := app.formatKeybindingsForStatusBar(); !strings.Contains(normal, "Enter: focus") {
		t.Fatalf("normal hints = %q, want Enter focus hint", normal)
	}
	app.EnterFocusMode()
	if got, want := app.formatKeybindingsForStatusBar(), "q: quit | Esc: back | ↑/↓: scroll"; got != want {
		t.Fatalf("focus hints = %q, want %q", got, want)
	}
	if err := app.handleQuit(); err != gocui.ErrQuit {
		t.Fatalf("q handler returned %v, want ErrQuit", err)
	}
}

func layoutWindowIDs(box *boxlayout.Box) []string {
	if box == nil {
		return nil
	}
	var ids []string
	if box.Window != "" {
		ids = append(ids, box.Window)
	}
	for _, child := range box.Children {
		ids = append(ids, layoutWindowIDs(child)...)
	}
	return ids
}
