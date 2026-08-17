package main

// FocusMode describes the single-panel layout currently active, if any.
type FocusMode struct {
	Active  bool
	PanelID string
}

// FocusModeCapability supplies the small amount of panel-specific behavior
// needed by the generic focused layout.
type FocusModeCapability struct {
	PanelID     string
	StatusHints string
	OnEnter     func()
	OnExit      func()
	OnKey       func(key interface{}) bool
}

func (a *App) RegisterFocusModeCapability(capability FocusModeCapability) {
	if a.focusModePanels == nil {
		a.focusModePanels = make(map[string]FocusModeCapability)
	}
	a.focusModePanels[capability.PanelID] = capability
}

func (a *App) EnterFocusMode() bool {
	if a.focusMode.Active || a.IsModalOpen {
		return false
	}
	focusables := a.getFocusableComponents()
	if a.currentFocusIdx < 0 || a.currentFocusIdx >= len(focusables) {
		return false
	}
	panelID := focusables[a.currentFocusIdx].ID()
	capability, supported := a.focusModePanels[panelID]
	if !supported {
		return false
	}
	a.focusMode = FocusMode{Active: true, PanelID: panelID}
	if capability.OnEnter != nil {
		capability.OnEnter()
	}
	return true
}

func (a *App) ExitFocusMode() bool {
	if !a.focusMode.Active {
		return false
	}
	panelID := a.focusMode.PanelID
	capability := a.focusModePanels[panelID]
	a.focusMode = FocusMode{}
	if capability.OnExit != nil {
		capability.OnExit()
	}
	a.focusPanel(panelID)
	return true
}

func (a *App) HandleFocusModeKey(key interface{}) bool {
	if !a.focusMode.Active {
		return false
	}
	capability := a.focusModePanels[a.focusMode.PanelID]
	return capability.OnKey != nil && capability.OnKey(key)
}

func (a *App) focusModeStatusHints() string {
	if !a.focusMode.Active {
		return ""
	}
	return a.focusModePanels[a.focusMode.PanelID].StatusHints
}

func (a *App) focusPanel(panelID string) {
	focusables := a.getFocusableComponents()
	if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(focusables) {
		focusables[a.currentFocusIdx].OnBlur(a.g)
	}
	for i, panel := range focusables {
		if panel.ID() != panelID {
			continue
		}
		a.currentFocusIdx = i
		panel.OnFocus(a.g)
		if a.g != nil {
			a.g.SetCurrentView(panelID)
		}
		return
	}
}
