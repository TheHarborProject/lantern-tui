package main

import "github.com/jesseduffield/lazycore/pkg/boxlayout"

type Workspace struct {
	ID         string
	Name       string
	Components []UIComponent
	Layout     *boxlayout.Box
	focusID    string
}

func (a *App) RegisterWorkspace(workspace Workspace) {
	if workspace.Layout != nil {
		workspace.Layout.Weight = 1
	}
	a.workspaces = append(a.workspaces, workspace)
	for _, component := range workspace.Components {
		a.RegisterComponent(component)
	}
	if len(a.workspaces) == 1 {
		a.activeWorkspace = 0
	}
	if a.header != nil {
		a.header.SetWorkspaces(a.workspaces, a.activeWorkspace)
	}
}

func (a *App) SwitchWorkspace(index int) bool {
	if a.focusMode.Active || index < 0 || index >= len(a.workspaces) || index == a.activeWorkspace {
		return false
	}

	current := a.getFocusableComponents()
	if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(current) {
		a.workspaces[a.activeWorkspace].focusID = current[a.currentFocusIdx].ID()
		current[a.currentFocusIdx].OnBlur(a.g)
	}

	a.activeWorkspace = index
	if a.header != nil {
		a.header.SetWorkspaces(a.workspaces, index)
	}
	targetID := a.workspaces[index].focusID
	targets := a.getFocusableComponents()
	if len(targets) == 0 {
		a.currentFocusIdx = 0
		return true
	}
	if targetID == "" {
		targetID = targets[0].ID()
	}
	a.currentFocusIdx = 0
	a.focusPanel(targetID)
	return true
}

func (a *App) SwitchWorkspaceByID(workspaceID string) bool {
	for i := range a.workspaces {
		if a.workspaces[i].ID == workspaceID {
			return a.SwitchWorkspace(i)
		}
	}
	return false
}

func (a *App) FocusPanel(panelID string) bool {
	for _, panel := range a.getFocusableComponents() {
		if panel.ID() == panelID {
			a.focusPanel(panelID)
			return true
		}
	}
	return false
}

func (a *App) activeWorkspaceDefinition() *Workspace {
	if a.activeWorkspace < 0 || a.activeWorkspace >= len(a.workspaces) {
		return nil
	}
	return &a.workspaces[a.activeWorkspace]
}

func (a *App) componentInActiveWorkspace(componentID string) bool {
	workspace := a.activeWorkspaceDefinition()
	if workspace == nil {
		return true
	}
	for _, component := range workspace.Components {
		if component.ID() == componentID {
			return true
		}
	}
	return false
}
