package main

import "github.com/jesseduffield/gocui"

// ============================================================================
// ListPanel - Displays selectable list items
// ============================================================================

type ListItem[T any] struct {
	DisplayText string
	Object      T
}

type ListPanel struct {
	*Panel
	items              []ListItem[any]
	selectedIndex      int
	emptyMessage       string
	onSelectionChanged func(int)
}

func NewListPanel(id, title, subtitle, footer string, visible bool) *ListPanel {
	return &ListPanel{
		Panel:         NewPanel(id, title, subtitle, footer, visible),
		items:         []ListItem[any]{},
		selectedIndex: 0,
		emptyMessage:  "No items",
	}
}

// Render implements UIComponent interface.
func (l *ListPanel) Render(g *gocui.Gui, dim Dimension) error {
	if !l.visible {
		if v, err := g.View(l.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	v, err := g.SetView(l.id, dim.X0, dim.Y0, dim.X1, dim.Y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	l.SetView(v)

	// Apply panel style
	l.applyStyle(v)

	// Enable highlight for selection
	v.Highlight = true
	v.SelBgColor = gocui.ColorBlue

	// Clear and write items
	v.Clear()
	if len(l.items) == 0 {
		if _, err := v.Write([]byte(l.emptyMessage + "\n")); err != nil {
			return err
		}
	} else {
		for _, item := range l.items {
			if _, err := v.Write([]byte(item.DisplayText + "\n")); err != nil {
				return err
			}
		}
	}

	// Adjust origin if needed (e.g., after terminal resize)
	l.Panel.AdjustOrigin(v)

	// Set origin
	v.SetOrigin(0, l.Panel.originY)

	// Update cursor position
	_, h := v.Size()
	innerHeight := h - 2 // Exclude frame

	// Set cursor position relative to origin
	cy := l.selectedIndex - l.Panel.originY
	if cy >= 0 && cy < innerHeight && cy < len(l.items) {
		v.SetCursor(0, cy)
	} else {
		v.SetCursor(0, -1) // Hide cursor if out of view
	}

	return nil
}

// SetItems updates the list items.
func (l *ListPanel) SetItems(items []ListItem[any]) {
	l.items = items

	// Adjust selectedIndex if out of bounds
	if len(items) == 0 {
		l.selectedIndex = 0
	} else if l.selectedIndex >= len(items) {
		// Keep selection at the last valid index instead of resetting to 0
		l.selectedIndex = len(items) - 1
		logDebugf("ListPanel %s: selectedIndex adjusted to last item (%d)", l.ID(), l.selectedIndex)
	}
}

func (l *ListPanel) SetEmptyMessage(message string) {
	l.emptyMessage = message
}

func (l *ListPanel) SetOnSelectionChanged(handler func(int)) {
	l.onSelectionChanged = handler
}

func (l *ListPanel) notifySelectionChanged(previousIndex int) {
	if l.selectedIndex != previousIndex && l.onSelectionChanged != nil {
		l.onSelectionChanged(l.selectedIndex)
	}
}

// GetContentLines returns the number of items (lines) in the list
func (l *ListPanel) GetContentLines() int {
	return len(l.items)
}

// adjustOriginAfterSetItems adjusts scroll origin after items are set
func (l *ListPanel) adjustOriginAfterSetItems() {
	// Adjust origin if needed (only if view is already created)
	// This may be called before app.Run(), so GetView might return nil
	v := l.GetView(nil)
	if v != nil {
		_, h := v.Size()
		innerHeight := h - 2

		// Calculate maxOrigin
		maxOrigin := len(l.items) - innerHeight
		if maxOrigin < 0 {
			maxOrigin = 0
		}

		// Adjust origin if it exceeds maxOrigin
		if l.Panel.originY > maxOrigin {
			l.Panel.originY = maxOrigin
			logDebugf("ListPanel %s: originY adjusted to %d (maxOrigin)", l.ID(), l.Panel.originY)
		}

		// Ensure selected item is visible
		if l.selectedIndex < l.Panel.originY {
			l.Panel.originY = l.selectedIndex
			logDebugf("ListPanel %s: originY adjusted to %d (selected item above view)", l.ID(), l.Panel.originY)
		} else if l.selectedIndex >= l.Panel.originY+innerHeight {
			l.Panel.originY = l.selectedIndex - innerHeight + 1
			logDebugf("ListPanel %s: originY adjusted to %d (selected item below view)", l.ID(), l.Panel.originY)
		}
	}
	// If view is nil, origin adjustments will happen in first Render()
}

// Selectable interface implementation
func (l *ListPanel) GetSelectedIndex() int {
	return l.selectedIndex
}

func (l *ListPanel) SetSelectedIndex(index int) {
	if index >= 0 && index < len(l.items) {
		previousIndex := l.selectedIndex
		l.selectedIndex = index
		l.notifySelectionChanged(previousIndex)
	}
}

func (l *ListPanel) SelectNext() {
	previousIndex := l.selectedIndex
	if l.selectedIndex < len(l.items)-1 {
		l.selectedIndex++

		// Auto-scroll if needed
		v := l.GetView(nil)
		if v != nil {
			_, h := v.Size()
			innerHeight := h - 2
			if l.selectedIndex-l.Panel.originY >= innerHeight {
				l.Panel.originY++
			}
		}
	}
	l.notifySelectionChanged(previousIndex)
}

func (l *ListPanel) SelectPrev() {
	previousIndex := l.selectedIndex
	if l.selectedIndex > 0 {
		l.selectedIndex--

		// Auto-scroll if needed
		if l.selectedIndex < l.Panel.originY {
			l.Panel.originY--
		}
	}
	l.notifySelectionChanged(previousIndex)
}

// RegisterBindings overrides Panel's RegisterBindings to add list-specific bindings
func (l *ListPanel) RegisterBindings(g *gocui.Gui, app *App) error {
	// Call parent RegisterBindings for mouse wheel
	if err := l.Panel.RegisterBindings(g, app); err != nil {
		return err
	}

	// Arrow up: select previous item
	if err := app.RegisterViewKeybinding(l.ID(), Keybinding{
		Key:      gocui.KeyArrowUp,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			l.SelectPrev()
			return nil
		},
		DisplayName: "↑",
		Description: "prev item",
		Displayable: false, // Show in StatusBar for testing
	}); err != nil {
		return err
	}

	// Arrow down: select next item
	if err := app.RegisterViewKeybinding(l.ID(), Keybinding{
		Key:      gocui.KeyArrowDown,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			l.SelectNext()
			return nil
		},
		DisplayName: "↓",
		Description: "next item",
		Displayable: false, // Show in StatusBar for testing
	}); err != nil {
		return err
	}

	// Add mouse left click handler for item selection + focus
	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: l.ID(),
		Key:      gocui.MouseLeft,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			return l.handleClick(g, app, opts)
		},
	})

	logDebugf("Mouse click binding registered for ListPanel: %s", l.ID())
	return nil
}

// handleClick handles mouse click on list items
func (l *ListPanel) handleClick(g *gocui.Gui, app *App, opts gocui.ViewMouseBindingOpts) error {
	// Ignore if modal is open
	if app.IsModalOpen {
		return nil
	}

	v := l.GetView(g)
	if v == nil {
		return nil
	}

	// opts.Y is already content-relative index (including origin)
	clickedIndex := opts.Y

	logDebugf("ListPanel click: clickedIndex=%d, originY=%d, itemsLen=%d",
		clickedIndex, l.Panel.originY, len(l.items))

	// Validate index
	if clickedIndex >= 0 && clickedIndex < len(l.items) {
		previousIndex := l.selectedIndex
		l.selectedIndex = clickedIndex
		l.notifySelectionChanged(previousIndex)
		logDebugf("ListPanel: selected item %d (%s)", clickedIndex, l.items[clickedIndex].DisplayText)
	}

	// Handle focus switch (same logic as App.handlePanelClick)
	focusables := app.getFocusableComponents()
	for i, f := range focusables {
		if f.ID() == l.ID() {
			// Already focused, do nothing
			if i == app.currentFocusIdx {
				return nil
			}

			// Blur current focus
			if app.currentFocusIdx >= 0 && app.currentFocusIdx < len(focusables) {
				focusables[app.currentFocusIdx].OnBlur(g)
			}

			// Update focus
			app.currentFocusIdx = i
			focusables[i].OnFocus(g)
			g.SetCurrentView(l.ID())

			logDebugf("Focus switched to ListPanel: %s (index: %d)", l.ID(), i)
			return nil
		}
	}

	return nil
}

// Scrollable interface: fully inherited from Panel (no override needed)
// Panel's unified ScrollDown uses ViewBufferLines(), which works for both text and list content
