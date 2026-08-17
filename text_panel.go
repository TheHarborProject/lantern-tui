package main

import (
	"strings"

	"github.com/jesseduffield/gocui"
)

// ============================================================================
// TextPanel - Displays scrollable text content
// ============================================================================

type TextPanel struct {
	*Panel
	content string
}

func NewTextPanel(id, title, subtitle, footer string, visible bool) *TextPanel {
	return &TextPanel{
		Panel:   NewPanel(id, title, subtitle, footer, visible),
		content: "",
	}
}

// Render implements UIComponent interface.
func (t *TextPanel) Render(g *gocui.Gui, dim Dimension) error {
	if !t.visible {
		// Hide the view if it exists
		if v, err := g.View(t.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	v, err := g.SetView(t.id, dim.X0, dim.Y0, dim.X1, dim.Y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	t.SetView(v)

	// Apply panel style
	t.applyStyle(v)

	// Clear and write content
	v.Clear()
	if _, err := v.Write([]byte(t.content)); err != nil {
		return err
	}

	// Adjust origin if needed (e.g., after terminal resize)
	t.Panel.AdjustOrigin(v)

	// Restore scroll position (use Panel's originY)
	v.SetOrigin(0, t.Panel.originY)

	return nil
}

// SetContent updates the text content.
func (t *TextPanel) SetContent(content string) {
	t.content = content
}

// GetContentLines returns the number of lines in the content
func (t *TextPanel) GetContentLines() int {
	if t.content == "" {
		return 0
	}
	return strings.Count(t.content, "\n") + 1
}

// RegisterBindings adds text panel specific keybindings
func (t *TextPanel) RegisterBindings(g *gocui.Gui, app *App) error {
	// Call parent RegisterBindings for mouse wheel
	if err := t.Panel.RegisterBindings(g, app); err != nil {
		return err
	}

	// Arrow up: scroll up
	if err := app.RegisterViewKeybinding(t.ID(), Keybinding{
		Key:      gocui.KeyArrowUp,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			t.ScrollUp()
			return nil
		},
		DisplayName: "↑",
		Description: "scroll up",
		Displayable: false, // Don't show in StatusBar (default binding)
	}); err != nil {
		return err
	}

	// Arrow down: scroll down
	if err := app.RegisterViewKeybinding(t.ID(), Keybinding{
		Key:      gocui.KeyArrowDown,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			t.ScrollDown()
			return nil
		},
		DisplayName: "↓",
		Description: "scroll down",
		Displayable: false, // Don't show in StatusBar (default binding)
	}); err != nil {
		return err
	}

	return nil
}

// Scrollable interface: fully inherited from Panel (no override needed)
