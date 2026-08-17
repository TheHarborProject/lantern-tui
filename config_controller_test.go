package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jesseduffield/gocui"
)

type configFixture struct {
	app        *App
	config     *AuthoredConfig
	panel      *ListPanel
	quick      *ListPanel
	input      *InputModal
	selectOne  *ChoiceModal
	selectMany *ChoiceModal
	controller *ConfigController
}

func newConfigFixture() configFixture {
	app := &App{}
	panel := NewListPanel("config", "Config", "", "", true)
	quick := NewListPanel("config-quick-actions", "Quick Actions", "", "", true)
	input := NewInputModal("config-input", "Edit setting", ModalTypeInfo, 0, app)
	selectOne := NewChoiceModal("config-select", "Select value", false, app)
	selectMany := NewChoiceModal("config-multi-select", "Select values", true, app)
	config := defaultAuthoredConfig()
	controller := NewConfigController(config, panel, input, selectOne, selectMany)
	controller.SetQuickActionsPanel(quick)
	return configFixture{app, config, panel, quick, input, selectOne, selectMany, controller}
}

func enterConfigRow(t *testing.T, fixture configFixture, index int) {
	t.Helper()
	fixture.panel.SetSelectedIndex(index)
	binding := fixture.controller.Keybinding()
	if binding.Key != gocui.KeyEnter {
		t.Fatalf("Config edit key = %#v, want Enter", binding.Key)
	}
	if err := binding.Handler(nil, nil); err != nil {
		t.Fatalf("Config Enter returned %v", err)
	}
}

func TestConfigRowsRenderAsTwoColumns(t *testing.T) {
	fixture := newConfigFixture()
	row := formatConfigTableItem(fixture.panel.items[0], 80)
	if !strings.Contains(row, "project.root") || !strings.Contains(row, "│ .") {
		t.Fatalf("Config row is not a two-column rendering: %q", row)
	}
	narrow := formatConfigTableItem(fixture.panel.items[4], 40)
	if !strings.Contains(narrow, "│") || len([]rune(narrow)) > 40 {
		t.Fatalf("narrow Config row did not resize cleanly: %q", narrow)
	}
}

func TestConfigBooleanTogglesImmediatelyWithoutModal(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 1)
	if fixture.config.Runtime.Enabled {
		t.Fatal("runtime.enabled did not toggle")
	}
	if fixture.app.IsModalOpen || fixture.input.IsVisible() || fixture.selectOne.IsVisible() || fixture.selectMany.IsVisible() {
		t.Fatal("boolean toggle opened a modal")
	}
	if fixture.panel.GetSelectedIndex() != 1 {
		t.Fatalf("selection = %d, want 1", fixture.panel.GetSelectedIndex())
	}
	setting := fixture.panel.items[1].Object.(AuthoredConfigSetting)
	if value, ok := setting.Value.(bool); !ok || value {
		t.Fatalf("refreshed row value = %#v, want boolean false", setting.Value)
	}
	if !strings.Contains(fixture.panel.items[1].DisplayText, "false") {
		t.Fatalf("refreshed row did not display false: %q", fixture.panel.items[1].DisplayText)
	}
}

func TestConfigStringUsesPrefilledInputModalAndSaves(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 0)
	if !fixture.input.IsVisible() || !fixture.app.IsModalOpen {
		t.Fatal("string setting did not open InputModal")
	}
	if fixture.input.Panel.title != "Edit project.root" || fixture.input.InitialValue() != "." {
		t.Fatalf("InputModal title/value = %q/%q", fixture.input.Panel.title, fixture.input.InitialValue())
	}
	if !fixture.input.SubmitAndClose("/tmp/project") {
		t.Fatal("valid string edit was rejected")
	}
	if fixture.config.Project.Root != "/tmp/project" {
		t.Fatalf("project.root = %q", fixture.config.Project.Root)
	}
	if fixture.panel.GetSelectedIndex() != 0 || !strings.Contains(fixture.panel.items[0].DisplayText, "/tmp/project") {
		t.Fatal("string save did not refresh the row while preserving selection")
	}
}

func TestConfigNumberUsesValidatedInputModal(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 2)
	if !fixture.input.IsVisible() || fixture.input.InitialValue() != "5000" {
		t.Fatal("number setting did not open a prefilled InputModal")
	}
	if fixture.input.SubmitAndClose("ten seconds") {
		t.Fatal("invalid number was accepted")
	}
	if fixture.config.Runtime.Timeout != 5000 || !fixture.input.IsVisible() || !strings.Contains(fixture.input.Panel.subtitle, "whole number") {
		t.Fatal("invalid number did not remain open with an error and preserve config")
	}
	if !fixture.input.SubmitAndClose("10000") {
		t.Fatal("valid number was rejected")
	}
	if fixture.config.Runtime.Timeout != 10000 {
		t.Fatalf("runtime.timeout = %d", fixture.config.Runtime.Timeout)
	}
	if fixture.panel.GetSelectedIndex() != 2 {
		t.Fatalf("selection = %d, want 2", fixture.panel.GetSelectedIndex())
	}
}

func TestConfigExtendsUsesMultiSelectAndSavesStringSlice(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 3)
	if !fixture.selectMany.IsVisible() || fixture.selectOne.IsVisible() || fixture.input.IsVisible() {
		t.Fatal("extends did not exclusively open MultiSelectModal")
	}
	if !reflect.DeepEqual(fixture.selectMany.options, knownLanternPresets) {
		t.Fatalf("preset options = %#v", fixture.selectMany.options)
	}
	if got := fixture.selectMany.SelectedValues(); !reflect.DeepEqual(got, []string{"lantern:recommended"}) {
		t.Fatalf("preselected presets = %#v", got)
	}
	fixture.selectMany.SelectNext()
	fixture.selectMany.ToggleSelected()
	fixture.selectMany.SaveAndClose()
	want := []string{"lantern:recommended", "lantern:strict"}
	if !reflect.DeepEqual(fixture.config.Extends, want) {
		t.Fatalf("extends = %#v, want %#v", fixture.config.Extends, want)
	}
	setting := fixture.panel.items[3].Object.(AuthoredConfigSetting)
	if _, ok := setting.Value.([]string); !ok {
		t.Fatalf("extends row has type %T, want []string", setting.Value)
	}
	if fixture.panel.GetSelectedIndex() != 3 {
		t.Fatalf("selection = %d, want 3", fixture.panel.GetSelectedIndex())
	}
}

func TestConfigRuleUsesRestrictedSingleSelect(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 4)
	if !fixture.selectOne.IsVisible() || fixture.selectMany.IsVisible() || fixture.input.IsVisible() {
		t.Fatal("rule did not exclusively open SelectModal")
	}
	wantOptions := []string{"off", "warn", "error"}
	if !reflect.DeepEqual(fixture.selectOne.options, wantOptions) {
		t.Fatalf("rule options = %#v, want %#v", fixture.selectOne.options, wantOptions)
	}
	if got := fixture.selectOne.SelectedValues(); !reflect.DeepEqual(got, []string{"error"}) {
		t.Fatalf("preselected severity = %#v", got)
	}
	fixture.selectOne.SelectPrev()
	fixture.selectOne.SaveAndClose()
	if got := fixture.config.Rules["lantern/keyboard-access"]; got != ConfigSeverityWarn {
		t.Fatalf("saved severity = %q, want warn", got)
	}
	if fixture.panel.GetSelectedIndex() != 4 {
		t.Fatalf("selection = %d, want 4", fixture.panel.GetSelectedIndex())
	}
	if err := fixture.config.SetRuleSeverity("rules.lantern/keyboard-access", "serious"); err == nil {
		t.Fatal("rule editor accepted value outside off/warn/error")
	}
}

func TestConfigEditorCancelDoesNotMutate(t *testing.T) {
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 0)
	fixture.input.Hide()
	if fixture.config.Project.Root != "." {
		t.Fatal("InputModal cancel mutated config")
	}

	enterConfigRow(t, fixture, 3)
	fixture.selectMany.SelectNext()
	fixture.selectMany.ToggleSelected()
	fixture.selectMany.Hide()
	if !reflect.DeepEqual(fixture.config.Extends, []string{"lantern:recommended"}) {
		t.Fatal("MultiSelectModal cancel mutated config")
	}

	enterConfigRow(t, fixture, 4)
	fixture.selectOne.SelectPrev()
	fixture.selectOne.Hide()
	if fixture.config.Rules["lantern/keyboard-access"] != ConfigSeverityError {
		t.Fatal("SelectModal cancel mutated config")
	}
}

func TestAuthoredEditsDoNotMutateResolvedWireSnapshot(t *testing.T) {
	wire := lanternWireFixtures()[0]
	before, err := json.Marshal(wire.Config)
	if err != nil {
		t.Fatal(err)
	}
	fixture := newConfigFixture()
	enterConfigRow(t, fixture, 1)
	enterConfigRow(t, fixture, 4)
	fixture.selectOne.SelectPrev()
	fixture.selectOne.SaveAndClose()
	after, err := json.Marshal(wire.Config)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("authored edit mutated resolved snapshot:\nbefore %s\nafter  %s", before, after)
	}
}
