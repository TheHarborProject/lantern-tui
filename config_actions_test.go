package main

import (
	"os"
	"strings"
	"testing"

	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

func persistentConfigFixture(t *testing.T, contents string) (configFixture, *ConfigStore, *YesNoModal, string) {
	t.Helper()
	directory := t.TempDir()
	path := writeTestConfig(t, directory, contents)
	store := NewConfigStore(path)
	config, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	fixture := newConfigFixture()
	fixture.config = config
	fixture.controller.config = config
	confirm := NewYesNoModal("config-confirm", "Confirm", ModalTypeWarning, fixture.app)
	fixture.controller.SetPersistence(store, nil, confirm)
	return fixture, store, confirm, path
}

func confirmYes(modal *YesNoModal) {
	modal.onYes()
	modal.Hide()
}

func TestQuickActionsPanelRendersActions(t *testing.T) {
	fixture := newConfigFixture()
	if fixture.quick.title != "Quick Actions" || len(fixture.quick.items) != 4 {
		t.Fatalf("Quick Actions panel = title %q, rows %d", fixture.quick.title, len(fixture.quick.items))
	}
	for index, name := range []string{"Save", "Reload", "Reset", "Defaults"} {
		if !strings.Contains(fixture.quick.items[index].DisplayText, name) {
			t.Fatalf("action row %d = %q", index, fixture.quick.items[index].DisplayText)
		}
	}
}

func TestQuickActionsParticipatesInFocusNavigationAndPersistsFocus(t *testing.T) {
	fixture := newConfigFixture()
	other := NewTextPanel("other", "Other", "", "", true)
	workspaces := []Workspace{
		{ID: "config", Components: []UIComponent{fixture.panel, fixture.quick}, Layout: &boxlayout.Box{Direction: boxlayout.ROW, Children: []*boxlayout.Box{{Window: fixture.panel.ID(), Weight: 3}, {Window: fixture.quick.ID(), Weight: 1}}}},
		{ID: "other", Components: []UIComponent{other}, Layout: &boxlayout.Box{Window: other.ID()}},
	}
	app := &App{components: []UIComponent{fixture.panel, fixture.quick, other}, workspaces: workspaces}
	app.FocusNext()
	if app.currentFocusIdx != 1 || !fixture.quick.IsFocused() {
		t.Fatal("Tab-style focus navigation did not reach Quick Actions")
	}
	fixture.quick.SetSelectedIndex(2)
	app.SwitchWorkspace(1)
	app.SwitchWorkspace(0)
	if !fixture.quick.IsFocused() || fixture.quick.GetSelectedIndex() != 2 {
		t.Fatal("Quick Actions focus/selection did not survive workspace switching")
	}
}

func TestQuickActionSaveDelegatesToConfigSave(t *testing.T) {
	fixture, _, _, path := persistentConfigFixture(t, `{"project":{"root":"disk"}}`)
	if err := fixture.config.Edit("project.root", "saved-by-action"); err != nil {
		t.Fatal(err)
	}
	fixture.quick.SetSelectedIndex(0)
	fixture.controller.ExecuteSelectedQuickAction()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "saved-by-action") || fixture.config.IsDirty() {
		t.Fatal("Save action did not delegate to save behavior")
	}
	if fixture.quick.GetSelectedIndex() != 0 {
		t.Fatal("Save action reset Quick Actions selection")
	}
}

func TestQuickActionReloadDelegatesAndConfirmsWhenDirty(t *testing.T) {
	fixture, _, confirm, _ := persistentConfigFixture(t, `{"project":{"root":"disk"}}`)
	if err := fixture.config.Edit("project.root", "memory"); err != nil {
		t.Fatal(err)
	}
	fixture.quick.SetSelectedIndex(1)
	fixture.controller.ExecuteSelectedQuickAction()
	if !confirm.IsVisible() || fixture.controller.config.Project.Root != "memory" {
		t.Fatal("dirty Reload action did not request confirmation")
	}
	confirmYes(confirm)
	if fixture.controller.config.Project.Root != "disk" || fixture.quick.GetSelectedIndex() != 1 {
		t.Fatal("confirmed Reload action did not reload or preserve action selection")
	}
}

func TestQuickActionResetRestoresBaselineAndClearsDirty(t *testing.T) {
	fixture, _, confirm, _ := persistentConfigFixture(t, `{"project":{"root":"saved"},"future":{"keep":true}}`)
	if err := fixture.config.Edit("project.root", "changed"); err != nil {
		t.Fatal(err)
	}
	fixture.panel.SetSelectedIndex(2)
	fixture.quick.SetSelectedIndex(2)
	fixture.controller.ExecuteSelectedQuickAction()
	if !confirm.IsVisible() {
		t.Fatal("dirty Reset action did not request confirmation")
	}
	confirmYes(confirm)
	if fixture.config.Project.Root != "saved" || fixture.config.IsDirty() {
		t.Fatal("Reset did not restore the saved baseline and clear dirty")
	}
	if fixture.panel.GetSelectedIndex() != 2 || fixture.quick.GetSelectedIndex() != 2 {
		t.Fatal("Reset did not preserve selections")
	}
}

func TestQuickActionDefaultsIsInMemoryOnlyAndSetsDirty(t *testing.T) {
	fixture, _, confirm, path := persistentConfigFixture(t, `{"project":{"root":"custom"},"runtime":{"enabled":false,"timeout":9000}}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fixture.quick.SetSelectedIndex(3)
	fixture.controller.ExecuteSelectedQuickAction()
	if !confirm.IsVisible() {
		t.Fatal("Defaults did not request confirmation before replacing config")
	}
	confirmYes(confirm)
	if fixture.config.Project.Root != "." || !fixture.config.Runtime.Enabled || fixture.config.Runtime.Timeout != 5000 {
		t.Fatal("Defaults did not install the starter config in memory")
	}
	if !fixture.config.IsDirty() {
		t.Fatal("Defaults did not mark a differing config dirty")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("Defaults wrote to disk")
	}
}

func TestConfigShortcutsStillDelegateToSaveAndReload(t *testing.T) {
	fixture, _, confirm, path := persistentConfigFixture(t, `{"project":{"root":"disk"}}`)
	bindings := fixture.controller.shortcutKeybindings()
	if len(bindings) != 2 || bindings[0].Key != 's' || bindings[1].Key != 'r' {
		t.Fatalf("shortcut bindings = %#v", bindings)
	}
	if err := fixture.config.Edit("project.root", "shortcut-save"); err != nil {
		t.Fatal(err)
	}
	if err := bindings[0].Handler(nil, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "shortcut-save") {
		t.Fatal("s shortcut did not save")
	}
	if err := os.WriteFile(path, []byte(`{"project":{"root":"shortcut-reload"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := bindings[1].Handler(nil, nil); err != nil {
		t.Fatal(err)
	}
	if confirm.IsVisible() || fixture.controller.config.Project.Root != "shortcut-reload" {
		t.Fatal("r shortcut did not reload clean config")
	}
}

func TestConfigWorkspaceLayoutUsesThreeToOneVerticalSplit(t *testing.T) {
	settings := NewListPanel("config", "Config", "", "", true)
	actions := NewListPanel("config-actions", "Quick Actions", "", "", true)
	layout := &boxlayout.Box{Direction: boxlayout.ROW, Children: []*boxlayout.Box{{Window: settings.ID(), Weight: 3}, {Window: actions.ID(), Weight: 1}}}
	if layout.Direction != boxlayout.ROW || layout.Children[0].Weight != 3 || layout.Children[1].Weight != 1 {
		t.Fatal("Config workspace is not a 3:1 vertical split")
	}
}
