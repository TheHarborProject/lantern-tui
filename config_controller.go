package main

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"github.com/jesseduffield/gocui"
)

type ConfigController struct {
	config           *AuthoredConfig
	panel            *ListPanel
	quickActions     *ListPanel
	input            *InputModal
	selectModal      *ChoiceModal
	multiSelectModal *ChoiceModal
	store            *ConfigStore
	reloadConfirm    *YesNoModal
	loadError        error
	statusMessage    string
}

type ConfigActionID string

const (
	ConfigActionSave     ConfigActionID = "save"
	ConfigActionReload   ConfigActionID = "reload"
	ConfigActionReset    ConfigActionID = "reset"
	ConfigActionDefaults ConfigActionID = "defaults"
)

type ConfigQuickAction struct {
	ID          ConfigActionID
	Name        string
	Description string
}

var configQuickActions = []ConfigQuickAction{
	{ID: ConfigActionSave, Name: "Save", Description: "Write changes to lantern.config.json"},
	{ID: ConfigActionReload, Name: "Reload", Description: "Reload configuration from disk"},
	{ID: ConfigActionReset, Name: "Reset", Description: "Restore last saved configuration"},
	{ID: ConfigActionDefaults, Name: "Defaults", Description: "Restore Lantern defaults"},
}

func NewConfigController(config *AuthoredConfig, panel *ListPanel, input *InputModal, selectModal, multiSelectModal *ChoiceModal) *ConfigController {
	c := &ConfigController{config: config, panel: panel, input: input, selectModal: selectModal, multiSelectModal: multiSelectModal}
	panel.SetTitle("Config · Settings | Value")
	panel.SetEmptyMessage("No authored Lantern configuration loaded.")
	panel.SetRowFormatter(formatConfigTableItem)
	c.Refresh()
	return c
}

func (c *ConfigController) Refresh() {
	c.refreshChrome()
	c.refreshQuickActions()
	if c.loadError != nil {
		c.panel.SetItems(nil)
		c.panel.SetEmptyMessage(c.loadError.Error())
		return
	}
	settings := c.config.Flatten()
	items := make([]ListItem[any], 0, len(settings))
	for i := range settings {
		setting := settings[i]
		items = append(items, ListItem[any]{DisplayText: formatConfigTableRow(setting, 80), Object: setting})
	}
	c.panel.SetItems(items)
}

func (c *ConfigController) SetQuickActionsPanel(panel *ListPanel) {
	c.quickActions = panel
	panel.SetTitle("Quick Actions")
	panel.SetEmptyMessage("No configuration actions available.")
	c.refreshQuickActions()
}

func (c *ConfigController) refreshQuickActions() {
	if c.quickActions == nil {
		return
	}
	items := make([]ListItem[any], 0, len(configQuickActions))
	for _, action := range configQuickActions {
		items = append(items, ListItem[any]{DisplayText: fmt.Sprintf("%-10s %s", action.Name, action.Description), Object: action})
	}
	c.quickActions.SetItems(items)
	c.quickActions.subtitle = c.statusMessage
}

func (c *ConfigController) refreshChrome() {
	title := "Config · Settings | Value"
	if c.config != nil && c.config.IsDirty() {
		title = "Config * · Settings | Value"
	}
	c.panel.SetTitle(title)
	c.panel.subtitle = c.statusMessage
}

func (c *ConfigController) SetPersistence(store *ConfigStore, loadError error, reloadConfirm *YesNoModal) {
	c.store = store
	c.loadError = loadError
	c.reloadConfirm = reloadConfirm
	if loadError != nil {
		c.statusMessage = loadError.Error()
	}
	c.Refresh()
}

func formatConfigTableItem(item ListItem[any], width int) string {
	setting, ok := item.Object.(AuthoredConfigSetting)
	if !ok {
		return item.DisplayText
	}
	return formatConfigTableRow(setting, width)
}

func formatConfigTableRow(setting AuthoredConfigSetting, width int) string {
	if width < 20 {
		return truncateText(setting.Path+" | "+formatAuthoredValue(setting), width)
	}
	leftWidth := width * 45 / 100
	if leftWidth < 28 {
		leftWidth = 28
	}
	if leftWidth > width-10 {
		leftWidth = width - 10
	}
	valueWidth := width - leftWidth - 3
	path := truncateText(setting.Path, leftWidth)
	value := truncateText(formatAuthoredValue(setting), valueWidth)
	return fmt.Sprintf("%-*s │ %s", leftWidth, path, value)
}

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}

func (c *ConfigController) selectedSetting() *AuthoredConfigSetting {
	index := c.panel.GetSelectedIndex()
	if index < 0 || index >= len(c.panel.items) {
		return nil
	}
	setting, ok := c.panel.items[index].Object.(AuthoredConfigSetting)
	if !ok {
		return nil
	}
	return &setting
}

func (c *ConfigController) OpenSelected() {
	setting := c.selectedSetting()
	if setting == nil {
		return
	}
	selectedIndex := c.panel.GetSelectedIndex()
	refresh := func() { c.Refresh(); c.panel.SetSelectedIndex(selectedIndex) }
	switch setting.EditorKind {
	case ConfigEditorToggle:
		if c.config.Toggle(setting.Path) == nil {
			refresh()
		}
	case ConfigEditorText, ConfigEditorNumber:
		c.input.SetTitle("Edit " + setting.Path)
		c.input.SetInitialValue(formatAuthoredValue(*setting))
		c.input.SetOnValidateSubmit(func(input string) error {
			if err := c.config.Edit(setting.Path, input); err != nil {
				return err
			}
			refresh()
			return nil
		})
		c.input.Show(string(setting.EditorKind))
	case ConfigEditorSelect:
		c.selectModal.SetTitle("Edit " + setting.Path)
		c.selectModal.ShowOptions(setting.Options, []string{formatAuthoredValue(*setting)}, func(values []string) {
			if len(values) == 1 && c.config.SetRuleSeverity(setting.Path, values[0]) == nil {
				refresh()
			}
		})
	case ConfigEditorMultiSelect:
		current, _ := setting.Value.([]string)
		c.multiSelectModal.SetTitle("Edit " + setting.Path)
		c.multiSelectModal.ShowOptions(setting.Options, current, func(values []string) {
			if c.config.SetExtends(values) == nil {
				refresh()
			}
		})
	}
}

func (c *ConfigController) Save() {
	if c.store == nil {
		return
	}
	if c.loadError != nil {
		c.statusMessage = "Failed to save " + lanternConfigFilename + ": resolve or reload the invalid file first"
		c.Refresh()
		return
	}
	if err := c.store.Save(c.config); err != nil {
		c.statusMessage = "Failed to save " + lanternConfigFilename + ": " + err.Error()
	} else {
		c.statusMessage = "Saved " + lanternConfigFilename
	}
	c.Refresh()
}

func (c *ConfigController) Reload() {
	if c.store == nil {
		return
	}
	if c.config != nil && c.config.IsDirty() && c.reloadConfirm != nil {
		c.reloadConfirm.SetTitle("Reload Config")
		c.reloadConfirm.SetOnYes(c.reloadNow)
		c.reloadConfirm.Show("Discard unsaved config changes and reload?")
		return
	}
	c.reloadNow()
}

func (c *ConfigController) Reset() {
	if c.config == nil {
		return
	}
	if c.config.IsDirty() && c.reloadConfirm != nil {
		c.reloadConfirm.SetTitle("Reset Config")
		c.reloadConfirm.SetOnYes(c.resetNow)
		c.reloadConfirm.Show("Discard unsaved config changes and restore the last saved configuration?")
		return
	}
	c.resetNow()
}

func (c *ConfigController) resetNow() {
	selected := c.panel.GetSelectedIndex()
	if err := c.config.ResetToBaseline(); err != nil {
		c.statusMessage = "Failed to reset configuration: " + err.Error()
	} else {
		c.loadError = nil
		c.statusMessage = "Restored last saved configuration"
	}
	c.Refresh()
	c.panel.SetSelectedIndex(selected)
}

func (c *ConfigController) Defaults() {
	if c.config == nil {
		return
	}
	defaults := defaultAuthoredConfig()
	if c.loadError == nil && bytes.Equal(c.config.canonical(), defaults.canonical()) {
		c.statusMessage = "Lantern defaults already active"
		c.Refresh()
		return
	}
	if c.reloadConfirm != nil {
		c.reloadConfirm.SetTitle("Restore Defaults")
		c.reloadConfirm.SetOnYes(c.defaultsNow)
		c.reloadConfirm.Show("Replace the in-memory configuration with Lantern defaults?")
		return
	}
	c.defaultsNow()
}

func (c *ConfigController) defaultsNow() {
	selected := c.panel.GetSelectedIndex()
	c.config.ApplyDefaults()
	c.loadError = nil
	c.statusMessage = "Restored Lantern defaults (not saved)"
	c.Refresh()
	c.panel.SetSelectedIndex(selected)
}

func (c *ConfigController) ExecuteSelectedQuickAction() {
	if c.quickActions == nil {
		return
	}
	index := c.quickActions.GetSelectedIndex()
	if index < 0 || index >= len(c.quickActions.items) {
		return
	}
	action, ok := c.quickActions.items[index].Object.(ConfigQuickAction)
	if !ok {
		return
	}
	switch action.ID {
	case ConfigActionSave:
		c.Save()
	case ConfigActionReload:
		c.Reload()
	case ConfigActionReset:
		c.Reset()
	case ConfigActionDefaults:
		c.Defaults()
	}
}

func (c *ConfigController) reloadNow() {
	config, _, err := c.store.Load()
	if err != nil {
		c.loadError = err
		c.statusMessage = err.Error()
		c.Refresh()
		return
	}
	selected := c.panel.GetSelectedIndex()
	c.config = config
	c.loadError = nil
	c.statusMessage = "Reloaded " + lanternConfigFilename
	c.Refresh()
	c.panel.SetSelectedIndex(selected)
}

func (c *ConfigController) Keybinding() Keybinding {
	return Keybinding{Key: gocui.KeyEnter, Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.OpenSelected(); return nil }}
}

func (c *ConfigController) shortcutKeybindings() []Keybinding {
	return []Keybinding{
		{Key: 's', Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.Save(); return nil }},
		{Key: 'r', Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.Reload(); return nil }},
	}
}

func (c *ConfigController) RegisterBindings(app *App) error {
	settingsBindings := append([]Keybinding{c.Keybinding()}, c.shortcutKeybindings()...)
	for _, binding := range settingsBindings {
		if err := app.RegisterViewKeybinding(c.panel.ID(), binding); err != nil {
			return err
		}
	}
	if c.quickActions != nil {
		quickBindings := append([]Keybinding{{Key: gocui.KeyEnter, Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.ExecuteSelectedQuickAction(); return nil }}}, c.shortcutKeybindings()...)
		for _, binding := range quickBindings {
			if err := app.RegisterViewKeybinding(c.quickActions.ID(), binding); err != nil {
				return err
			}
		}
	}
	return nil
}
