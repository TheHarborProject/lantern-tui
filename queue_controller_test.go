package main

import (
	"strings"
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

type queueFixture struct {
	app        *App
	audit      *AuditController
	queue      *QueueController
	queuePanel *ListPanel
	components *ListPanel
	states     *ListPanel
	checks     *ListPanel
	evidence   *TextPanel
}

func newQueueFixture() queueFixture {
	components := NewListPanel("components", "Components", "", "", true)
	states := NewListPanel("states", "States", "", "", true)
	checks := NewListPanel("checks", "Checks", "", "", true)
	evidence := NewTextPanel("evidence", "Evidence", "", "", true)
	queuePanel := NewListPanel("queue", "Queue", "", "", true)
	audit := NewAuditController(fixtureComponents(), components, states, checks, evidence)
	workspaces := []Workspace{
		{ID: "audit", Name: "Audit", Components: []UIComponent{components, checks, states, evidence}, Layout: &boxlayout.Box{Window: "components"}},
		{ID: "runs", Name: "Runs", Components: []UIComponent{NewTextPanel("runs", "Runs", "", "", true)}, Layout: &boxlayout.Box{Window: "runs"}},
		{ID: "queue", Name: "Queue", Components: []UIComponent{queuePanel}, Layout: &boxlayout.Box{Window: "queue"}},
		{ID: "config", Name: "Config", Components: []UIComponent{NewTextPanel("config", "Config", "", "", true)}, Layout: &boxlayout.Box{Window: "config"}},
	}
	app := &App{
		components: []UIComponent{components, checks, states, evidence, workspaces[1].Components[0], queuePanel, workspaces[3].Components[0]},
		workspaces: workspaces, activeWorkspace: 2, currentFocusIdx: 0,
		statusBar: NewStatusBar("statusbar", AppConfig{}), focusModePanels: make(map[string]FocusModeCapability),
	}
	for _, capability := range audit.FocusModeCapabilities() {
		app.RegisterFocusModeCapability(capability)
	}
	queue := NewQueueController(audit, app, queuePanel)
	return queueFixture{app, audit, queue, queuePanel, components, states, checks, evidence}
}

func invokeQueueKey(t *testing.T, controller *QueueController, key interface{}) {
	t.Helper()
	for _, binding := range controller.Keybindings() {
		if binding.Key == key {
			if err := binding.Handler(nil, nil); err != nil {
				t.Fatalf("key %#v returned %v", key, err)
			}
			return
		}
	}
	t.Fatalf("queue key %#v is not registered", key)
}

func TestQueueRendersAllComponentsAndTogglesWithoutMovingCursor(t *testing.T) {
	fixture := newQueueFixture()
	if len(fixture.queuePanel.items) != 4 {
		t.Fatalf("Queue rows = %d, want 4", len(fixture.queuePanel.items))
	}
	component, ok := fixture.queuePanel.items[0].Object.(*Component)
	if !ok || component.CanonicalID != "ui/button" || component.SourceFile == "" {
		t.Fatalf("Queue row did not retain component domain object: %#v", fixture.queuePanel.items[0].Object)
	}
	fixture.queuePanel.SetSelectedIndex(1)
	invokeQueueKey(t, fixture.queue, gocui.KeySpace)
	if !fixture.audit.IsQueued("ui/label") {
		t.Fatal("Space did not add Label to the queue")
	}
	if !strings.HasPrefix(fixture.queuePanel.items[1].DisplayText, "[x]") {
		t.Fatalf("Label marker did not update immediately: %q", fixture.queuePanel.items[1].DisplayText)
	}
	if fixture.queuePanel.GetSelectedIndex() != 1 {
		t.Fatalf("Queue cursor moved to %d, want 1", fixture.queuePanel.GetSelectedIndex())
	}
	if !strings.Contains(fixture.queuePanel.footer, "Selected 3 / 4") {
		t.Fatalf("Queue summary = %q", fixture.queuePanel.footer)
	}
}

func TestQueueSelectAllNoneAndSummary(t *testing.T) {
	fixture := newQueueFixture()
	invokeQueueKey(t, fixture.queue, 'a')
	if len(fixture.audit.QueuedComponents()) != 4 || !strings.Contains(fixture.queuePanel.footer, "Selected 4 / 4") {
		t.Fatalf("select all failed: queued=%d footer=%q", len(fixture.audit.QueuedComponents()), fixture.queuePanel.footer)
	}
	invokeQueueKey(t, fixture.queue, 'n')
	if len(fixture.audit.QueuedComponents()) != 0 || !strings.Contains(fixture.queuePanel.footer, "Selected 0 / 4") {
		t.Fatalf("select none failed: queued=%d footer=%q", len(fixture.audit.QueuedComponents()), fixture.queuePanel.footer)
	}
}

func TestAuditComponentsFocusAndQueueStaySynchronized(t *testing.T) {
	fixture := newQueueFixture()
	fixture.app.activeWorkspace = 0
	fixture.app.currentFocusIdx = 0
	fixture.app.EnterFocusMode()
	fixture.app.HandleFocusModeKey(' ')
	if fixture.audit.IsQueued("ui/button") || !strings.HasPrefix(fixture.queuePanel.items[0].DisplayText, "[ ]") {
		t.Fatal("Components Focus toggle did not update Queue")
	}
	fixture.app.ExitFocusMode()
	fixture.app.SwitchWorkspaceByID("queue")
	fixture.queuePanel.SetSelectedIndex(2)
	invokeQueueKey(t, fixture.queue, gocui.KeySpace)
	fixture.app.SwitchWorkspaceByID("audit")
	fixture.app.currentFocusIdx = 0
	fixture.app.EnterFocusMode()
	if !strings.HasPrefix(fixture.components.items[2].DisplayText, "[ ]") {
		t.Fatalf("Queue toggle did not update Components Focus row: %q", fixture.components.items[2].DisplayText)
	}
}

func TestQueueEnterInspectsCanonicalComponentAndPreservesCursor(t *testing.T) {
	fixture := newQueueFixture()
	fixture.queuePanel.SetSelectedIndex(2)
	invokeQueueKey(t, fixture.queue, gocui.KeyEnter)
	if fixture.app.activeWorkspace != 0 {
		t.Fatalf("Enter opened workspace %d, want Audit", fixture.app.activeWorkspace)
	}
	if got := fixture.audit.currentComponent().CanonicalID; got != "ui/calendar" {
		t.Fatalf("Enter selected %q, want ui/calendar", got)
	}
	if fixture.states.items[0].Object.(*State).ID != "calendar-single" || fixture.checks.items[0].Object.(*Check).ID != "chk-calendar-grid" {
		t.Fatal("Enter did not refresh Calendar states and checks")
	}
	if !strings.Contains(fixture.evidence.content, "lantern/grid-navigation") {
		t.Fatal("Enter did not refresh Calendar evidence")
	}
	if got := fixture.app.getFocusableComponents()[fixture.app.currentFocusIdx].ID(); got != "components" {
		t.Fatalf("Enter focused %q, want components", got)
	}
	fixture.app.SwitchWorkspaceByID("queue")
	if fixture.queuePanel.GetSelectedIndex() != 2 {
		t.Fatalf("Queue cursor after return = %d, want 2", fixture.queuePanel.GetSelectedIndex())
	}
}

func TestQueueMockRunMessages(t *testing.T) {
	fixture := newQueueFixture()
	invokeQueueKey(t, fixture.queue, 'r')
	if !strings.Contains(fixture.queuePanel.footer, "Queued audit: 2 components") {
		t.Fatalf("mock run message = %q", fixture.queuePanel.footer)
	}
	invokeQueueKey(t, fixture.queue, 'n')
	invokeQueueKey(t, fixture.queue, 'r')
	if !strings.Contains(fixture.queuePanel.footer, "Nothing selected to audit.") {
		t.Fatalf("empty run message = %q", fixture.queuePanel.footer)
	}
}

func TestQueueStateSurvivesWorkspaceRoundTripAndHasContextHints(t *testing.T) {
	fixture := newQueueFixture()
	fixture.queuePanel.SetSelectedIndex(3)
	invokeQueueKey(t, fixture.queue, gocui.KeySpace)
	fixture.app.SwitchWorkspaceByID("audit")
	fixture.app.SwitchWorkspaceByID("queue")
	if !fixture.audit.IsQueued("ui/separator") || fixture.queuePanel.GetSelectedIndex() != 3 {
		t.Fatal("Queue state or cursor did not survive workspace round trip")
	}
	if hints := fixture.app.formatKeybindingsForStatusBar(); !strings.Contains(hints, "Space:toggle") || !strings.Contains(hints, "r:run") {
		t.Fatalf("Queue status hints = %q", hints)
	}
}
