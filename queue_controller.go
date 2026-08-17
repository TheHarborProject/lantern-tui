package main

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

type QueueController struct {
	audit *AuditController
	app   *App
	panel *ListPanel

	message string
}

func NewQueueController(audit *AuditController, app *App, panel *ListPanel) *QueueController {
	c := &QueueController{audit: audit, app: app, panel: panel}
	panel.SetEmptyMessage("No components available")
	audit.OnQueueChanged(c.Refresh)
	c.Refresh()
	return c
}

func (c *QueueController) Refresh() {
	components := c.audit.Components()
	items := make([]ListItem[any], 0, len(components))
	for i := range components {
		component := &components[i]
		marker := "[ ]"
		if c.audit.IsQueued(component.CanonicalID) {
			marker = "[x]"
		}
		items = append(items, ListItem[any]{
			DisplayText: fmt.Sprintf("%s %-14s  %-18s  %s", marker, component.DisplayName, component.CanonicalID, component.SourceFile),
			Object:      component,
		})
	}
	c.panel.SetItems(items)
	c.updateFooter()
}

func (c *QueueController) updateFooter() {
	footer := fmt.Sprintf("Selected %d / %d", len(c.audit.QueuedComponents()), len(c.audit.Components()))
	if c.message != "" {
		footer += " · " + c.message
	}
	c.panel.SetFooter(footer)
}

func (c *QueueController) selectedComponent() *Component {
	index := c.panel.GetSelectedIndex()
	components := c.audit.Components()
	if index < 0 || index >= len(components) {
		return nil
	}
	return &components[index]
}

func (c *QueueController) ToggleSelected() {
	component := c.selectedComponent()
	if component == nil {
		return
	}
	c.message = ""
	c.audit.ToggleQueued(component.CanonicalID)
}

func (c *QueueController) SelectAll() {
	c.message = ""
	c.audit.SelectAllQueued()
}

func (c *QueueController) SelectNone() {
	c.message = ""
	c.audit.SelectNoQueued()
}

func (c *QueueController) InspectSelected() {
	component := c.selectedComponent()
	if component == nil || !c.app.SwitchWorkspaceByID("audit") {
		return
	}
	c.audit.SelectComponentByCanonicalID(component.CanonicalID)
	c.app.FocusPanel(c.audit.componentsPanel.ID())
}

func (c *QueueController) MockRun() {
	count := len(c.audit.QueuedComponents())
	if count == 0 {
		c.message = "Nothing selected to audit."
	} else {
		c.message = fmt.Sprintf("Queued audit: %d components", count)
	}
	c.updateFooter()
}

func (c *QueueController) Keybindings() []Keybinding {
	return []Keybinding{
		{Key: gocui.KeySpace, Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.ToggleSelected(); return nil }},
		{Key: 'a', Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.SelectAll(); return nil }},
		{Key: 'n', Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.SelectNone(); return nil }},
		{Key: gocui.KeyEnter, Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.InspectSelected(); return nil }},
		{Key: 'r', Modifier: gocui.ModNone, Handler: func(*gocui.Gui, *gocui.View) error { c.MockRun(); return nil }},
	}
}

func (c *QueueController) RegisterBindings() error {
	for _, binding := range c.Keybindings() {
		if err := c.app.RegisterViewKeybinding(c.panel.ID(), binding); err != nil {
			return err
		}
	}
	return nil
}
