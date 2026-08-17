package main

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

// layoutManager is the main layout function called by gocui on each render cycle.
// It calculates dimensions using lazycore and renders all components.
func (a *App) layoutManager(g *gocui.Gui) error {
	width, height := g.Size()

	// Set initial focus on first render
	if a.currentFocusIdx == 0 {
		focusables := a.getFocusableComponents()
		if len(focusables) > 0 {
			// Check if first component is actually focused
			if !focusables[0].IsFocused() {
				focusables[0].OnFocus(g)
				g.SetCurrentView(focusables[0].ID())
				logDebugf("Initial focus set to: %s", focusables[0].ID())
			}
		}
	}

	// Build layout tree for panels (StatusBar + other focusable panels)
	layoutTree := a.buildLayoutTree()

	// Calculate dimensions using lazycore
	dimensionMap := boxlayout.ArrangeWindows(layoutTree, 0, 0, width, height)

	// Update StatusBar keybindings display
	a.statusBar.SetLeftContent(a.formatKeybindingsForStatusBar())

	// Render all panels using calculated dimensions
	if err := a.renderPanels(g, dimensionMap); err != nil {
		return err
	}

	// Render current modal if one is open (overlay, not part of lazycore layout)
	if a.currentModal != nil {
		if err := a.currentModal.Render(g, Dimension{}); err != nil {
			return err
		}

		// Set focus to modal
		if cv := g.CurrentView(); cv == nil || cv.Name() != a.currentModal.ID() {
			g.SetCurrentView(a.currentModal.ID())
			logDebugf("layoutManager: Set focus to modal %s", a.currentModal.ID())
		}
	}

	// Render spinner if task is running (overlay on StatusBar)
	if a.IsTaskRunning && a.spinner != nil {
		// Use StatusBar dimensions for spinner overlay
		if statusBarDim, ok := dimensionMap[a.statusBar.ID()]; ok {
			dim := Dimension{
				X0: statusBarDim.X0,
				Y0: statusBarDim.Y0,
				X1: statusBarDim.X1,
				Y1: statusBarDim.Y1,
			}
			if err := a.spinner.Render(g, dim); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildLayoutTree constructs the lazycore boxlayout tree.
// Layout structure:
//
//	ROW
//	  ├─ [User-defined layout strategy]  (e.g., COLUMN with panels)
//	  └─ StatusBar (1 line at bottom)
func (a *App) buildLayoutTree() *boxlayout.Box {
	// Get current layout strategy (user panels)
	var userLayout *boxlayout.Box

	if a.focusMode.Active {
		userLayout = &boxlayout.Box{Window: a.focusMode.PanelID, Weight: 1}
	} else if workspace := a.activeWorkspaceDefinition(); workspace != nil {
		userLayout = workspace.Layout
	} else if len(a.layoutStrategies) > 0 && a.currentLayoutStrategyIdx < len(a.layoutStrategies) {
		userLayout = a.layoutStrategies[a.currentLayoutStrategyIdx]
	} else {
		// Default: empty layout
		userLayout = &boxlayout.Box{
			Direction: boxlayout.COLUMN,
			Children:  []*boxlayout.Box{},
		}
	}

	// Root layout: user panels + status bar
	children := make([]*boxlayout.Box, 0, 3)
	if a.header != nil {
		children = append(children, &boxlayout.Box{Window: a.header.ID(), Size: 2})
	}
	children = append(children, userLayout, &boxlayout.Box{Window: a.statusBar.ID(), Size: 1})
	root := &boxlayout.Box{Direction: boxlayout.ROW, Children: children}

	return root
}

// renderPanels renders all panels (including StatusBar) using calculated dimensions.
func (a *App) renderPanels(g *gocui.Gui, dimensionMap map[string]boxlayout.Dimensions) error {
	if a.header != nil {
		headerDim, ok := dimensionMap[a.header.ID()]
		if ok {
			if err := a.header.Render(g, Dimension{X0: headerDim.X0, Y0: headerDim.Y0, X1: headerDim.X1, Y1: headerDim.Y1}); err != nil {
				return err
			}
		}
	}

	// Render StatusBar first
	if statusBarDim, ok := dimensionMap[a.statusBar.ID()]; ok {
		dim := Dimension{
			X0: statusBarDim.X0,
			Y0: statusBarDim.Y0,
			X1: statusBarDim.X1,
			Y1: statusBarDim.Y1,
		}
		if err := a.statusBar.Render(g, dim); err != nil {
			return err
		}
	}

	// Render all other components that are panels (not modals)
	for _, component := range a.components {
		// Skip modals (they're rendered separately as overlays)
		if _, isModal := component.(*BaseModal); isModal {
			continue
		}

		// Views outside the active layout are hidden without changing the
		// component's semantic visibility or its panel state.
		if _, ok := dimensionMap[component.ID()]; !ok {
			if view := componentView(component, g); view != nil {
				view.Visible = false
			}
			continue
		}

		// Render component if it has a dimension mapping
		if dim, ok := dimensionMap[component.ID()]; ok {
			if view := componentView(component, g); view != nil {
				view.Visible = true
			}
			dimension := Dimension{
				X0: dim.X0,
				Y0: dim.Y0,
				X1: dim.X1,
				Y1: dim.Y1,
			}
			if err := component.Render(g, dimension); err != nil {
				return err
			}
		}
	}

	return nil
}

func componentView(component UIComponent, g *gocui.Gui) *gocui.View {
	if base, ok := component.(interface{ GetView(*gocui.Gui) *gocui.View }); ok {
		return base.GetView(g)
	}
	return nil
}

// SetLayoutStrategy sets the active layout strategy.
// Users can define multiple layout strategies and switch between them.
func (a *App) SetLayoutStrategy(strategy *boxlayout.Box) {
	strategy.Weight = 1 // Default weight
	a.layoutStrategies = append(a.layoutStrategies, strategy)
}

// SwitchLayoutStrategy switches to the next layout strategy.
func (a *App) SwitchLayoutStrategy() {
	if len(a.layoutStrategies) == 0 {
		return
	}
	a.currentLayoutStrategyIdx = (a.currentLayoutStrategyIdx + 1) % len(a.layoutStrategies)
}
