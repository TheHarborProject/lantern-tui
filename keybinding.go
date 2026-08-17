package main

import "github.com/jesseduffield/gocui"

// Keybinding represents a keyboard shortcut with metadata for display.
type Keybinding struct {
	Key         interface{}                         // gocui.Key or rune
	Modifier    gocui.Modifier                      // Modifier keys (Ctrl, Alt, etc.)
	Handler     func(*gocui.Gui, *gocui.View) error // Callback function
	DisplayName string                              // Short name for StatusBar (e.g., "q", "Enter", "←")
	Description string                              // Description (e.g., "quit", "confirm", "prev panel")
	Displayable bool                                // Whether to show in StatusBar
}

// RegisterGlobalKeybinding registers a keybinding that works across all views.
func (a *App) RegisterGlobalKeybinding(kb Keybinding) error {
	err := a.g.SetKeybinding("", kb.Key, kb.Modifier, kb.Handler)
	if err != nil {
		return err
	}

	// Store for StatusBar display
	if kb.Displayable {
		a.GlobalKeybindings = append(a.GlobalKeybindings, kb)
	}

	return nil
}

// RegisterViewKeybinding registers a keybinding for a specific view.
func (a *App) RegisterViewKeybinding(viewID string, kb Keybinding) error {
	err := a.g.SetKeybinding(viewID, kb.Key, kb.Modifier, kb.Handler)
	if err != nil {
		return err
	}

	// Store for StatusBar display
	if kb.Displayable {
		if a.viewKeybindings == nil {
			a.viewKeybindings = make(map[string][]Keybinding)
		}
		a.viewKeybindings[viewID] = append(a.viewKeybindings[viewID], kb)
	}

	return nil
}

// formatKeybindingsForStatusBar formats keybindings for display in StatusBar.
// Format: "key: description | key: description | ..."
func (a *App) formatKeybindingsForStatusBar() string {
	if a.focusMode.Active {
		return a.focusModeStatusHints()
	}
	if len(a.workspaces) > 0 {
		if workspace := a.activeWorkspaceDefinition(); workspace != nil && workspace.ID == "queue" {
			return "Space:toggle a:all n:none Enter:view r:run 1-4:tabs q:quit"
		}
		return "1-4: tabs | Tab/⇧Tab: panel | Enter: focus | q: quit"
	}
	var parts []string

	// Add global keybindings first
	globalCount := 0
	for _, kb := range a.GlobalKeybindings {
		if kb.Displayable {
			parts = append(parts, kb.DisplayName+": "+kb.Description)
			globalCount++
		}
	}
	logDebugf("formatKeybindingsForStatusBar: Global keybindings count=%d", globalCount)

	// Add current focused view's keybindings
	focusables := a.getFocusableComponents()
	if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(focusables) {
		currentView := focusables[a.currentFocusIdx]
		currentViewID := currentView.ID()
		logDebugf("formatKeybindingsForStatusBar: Current focus ID=%s", currentViewID)

		// If current view is a TabPanel, use active child's ID for keybinding lookup
		if tabPanel, ok := currentView.(*TabPanel); ok {
			activeChild := tabPanel.GetActiveChild()
			if activeChild != nil {
				currentViewID = activeChild.ID()
				logDebugf("formatKeybindingsForStatusBar: TabPanel detected, active child ID=%s", currentViewID)
			} else {
				logDebugf("formatKeybindingsForStatusBar: TabPanel has no active child!")
			}
		}

		if viewKbs, ok := a.viewKeybindings[currentViewID]; ok {
			viewKbCount := 0
			for _, kb := range viewKbs {
				if kb.Displayable {
					parts = append(parts, kb.DisplayName+": "+kb.Description)
					viewKbCount++
				}
			}
			logDebugf("formatKeybindingsForStatusBar: Found view keybindings for ID=%s, count=%d", currentViewID, viewKbCount)
		} else {
			logDebugf("formatKeybindingsForStatusBar: No view keybindings found for ID=%s", currentViewID)
		}
	} else {
		logDebugf("formatKeybindingsForStatusBar: No focused component (currentFocusIdx=%d, focusables=%d)", a.currentFocusIdx, len(focusables))
	}

	// Join with " | "
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " | "
		}
		result += part
	}

	logDebugf("formatKeybindingsForStatusBar: Final result (len=%d): %s", len(result), result)
	return result
}

// setupDefaultKeybindings registers default keybindings for the app.
func (a *App) setupDefaultKeybindings() error {
	// Global: Quit on 'q' or Ctrl+C
	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:         'q',
		Modifier:    gocui.ModNone,
		Handler:     func(g *gocui.Gui, v *gocui.View) error { return a.handleQuit() },
		DisplayName: "q",
		Description: "quit",
		Displayable: true,
	}); err != nil {
		return err
	}

	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:         gocui.KeyCtrlC,
		Modifier:    gocui.ModNone,
		Handler:     func(g *gocui.Gui, v *gocui.View) error { return a.handleQuit() },
		DisplayName: "Ctrl+C",
		Description: "quit",
		Displayable: false, // Don't show redundant quit binding
	}); err != nil {
		return err
	}

	// Global: Focus navigation with arrow keys (Left/Right)
	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyArrowRight,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.FocusNext()
			return nil
		},
		DisplayName: "→",
		Description: "next panel",
		Displayable: false,
	}); err != nil {
		return err
	}

	// Global: Focus navigation in visual reading order with Tab/Shift+Tab.
	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyTab,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.FocusNext()
			return nil
		},
		DisplayName: "Tab",
		Description: "next panel",
		Displayable: true,
	}); err != nil {
		return err
	}

	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyBacktab,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.FocusPrev()
			return nil
		},
		DisplayName: "Shift+Tab",
		Description: "previous panel",
		Displayable: true,
	}); err != nil {
		return err
	}

	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyEnter,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.EnterFocusMode()
			return nil
		},
		DisplayName: "Enter",
		Description: "focus",
		Displayable: true,
	}); err != nil {
		return err
	}

	if err := a.RegisterGlobalKeybinding(a.focusModeSpaceKeybinding()); err != nil {
		return err
	}

	for index, key := range []rune{'1', '2', '3', '4'} {
		if err := a.RegisterGlobalKeybinding(a.workspaceKeybinding(index, key)); err != nil {
			return err
		}
	}

	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyArrowLeft,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.FocusPrev()
			return nil
		},
		DisplayName: "←",
		Description: "prev panel",
		Displayable: false,
	}); err != nil {
		return err
	}

	// Global: ESC to close modal
	if err := a.RegisterGlobalKeybinding(Keybinding{
		Key:      gocui.KeyEsc,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			return a.handleEscape()
		},
		DisplayName: "Esc",
		Description: "close modal",
		Displayable: false, // Only show when modal is open (TODO: dynamic display)
	}); err != nil {
		return err
	}

	return nil
}

func (a *App) workspaceKeybinding(index int, key rune) Keybinding {
	return Keybinding{
		Key:      key,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.SwitchWorkspace(index)
			return nil
		},
		Displayable: false,
	}
}

func (a *App) focusModeSpaceKeybinding() Keybinding {
	return Keybinding{
		Key:      gocui.KeySpace,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			a.HandleFocusModeKey(' ')
			return nil
		},
		DisplayName: "Space",
		Description: "toggle",
		Displayable: false,
	}
}

// handleEscape handles ESC key press (close modal, restore focus).
func (a *App) handleEscape() error {
	if a.ExitFocusMode() {
		return nil
	}
	if !a.IsModalOpen {
		return nil
	}

	// Hide current modal (modal.Hide() handles state cleanup)
	a.currentModal.Hide()

	return nil
}

func (a *App) handleQuit() error {
	if a.IsModalOpen {
		return nil
	}
	return gocui.ErrQuit
}
