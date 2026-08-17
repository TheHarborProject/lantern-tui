package main

import "github.com/jesseduffield/gocui"

// ============================================================================
// TabPanel - Container for multiple child views with tab navigation
// ============================================================================

// TabItem represents a single tab with its content and keybindings
type TabItem struct {
	Title       string       // Tab title displayed in tab bar
	Component   UIComponent  // Child component (holds state like scroll position, selection)
	Keybindings []Keybinding // Keybindings active when this tab is focused
}

type TabPanel struct {
	*Panel
	tabs        []TabItem // Tabs with their components and keybindings
	activeIndex int       // Currently selected tab index
	app         *App      // Reference to App for StatusBar keybinding updates
}

func NewTabPanel(id, subtitle, footer string, visible bool) *TabPanel {
	return &TabPanel{
		Panel:       NewPanel(id, "", subtitle, footer, visible), // Title handled by tabs
		tabs:        []TabItem{},
		activeIndex: 0,
	}
}

// Render implements UIComponent interface.
func (t *TabPanel) Render(g *gocui.Gui, dim Dimension) error {
	if !t.visible {
		if v, err := g.View(t.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	// Create tab panel view
	v, err := g.SetView(t.id, dim.X0, dim.Y0, dim.X1, dim.Y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	t.SetView(v)

	// Apply panel style
	t.applyStyle(v)

	// Set up tabs using TabItem titles
	var tabTitles []string
	for _, tab := range t.tabs {
		tabTitles = append(tabTitles, tab.Title)
	}
	v.Tabs = tabTitles
	v.TabIndex = t.activeIndex

	// Set tab colors based on focus state
	if t.focused {
		v.SelFgColor = gocui.ColorGreen | gocui.AttrBold // Active tab when focused
	} else {
		v.SelFgColor = gocui.ColorGreen | gocui.AttrNone // Active tab when not focused
	}
	v.TitleColor = gocui.ColorWhite | gocui.AttrNone // Inactive tabs

	// Clear content before rendering active tab
	v.Clear()

	// Render active tab's content using type assertion
	if len(t.tabs) == 0 {
		return nil
	}

	activeTab := t.tabs[t.activeIndex]
	return t.renderTabContent(v, activeTab.Component)
}

// renderTabContent renders the active tab's content using type assertion
func (t *TabPanel) renderTabContent(v *gocui.View, component UIComponent) error {
	switch child := component.(type) {
	case *ListPanel:
		logDebugf("ListPanel %s: Rendering ListPanel in TabPanel %s", child.ID(), t.ID())
		return t.renderListPanel(v, child)
	case *TextPanel:
		logDebugf("TextPanel %s: Rendering TextPanel in TabPanel %s", child.ID(), t.ID())
		return t.renderTextPanel(v, child)
	default:
		logDebugf("TabPanel %s: Unknown component type for rendering: %T", t.ID(), component)
		return nil
	}
}

// renderListPanel renders a ListPanel's content into the TabPanel view
func (t *TabPanel) renderListPanel(v *gocui.View, lp *ListPanel) error {
	// Enable highlight for selection
	v.Highlight = true
	v.SelBgColor = gocui.ColorBlue

	// Calculate visible area
	_, viewHeight := v.Size()
	innerHeight := viewHeight - 2 // Exclude frame

	// Render items first
	for _, item := range lp.items {
		v.Write([]byte(item.DisplayText + "\n"))
	}

	// Adjust origin after rendering content
	lp.Panel.AdjustOrigin(v)

	// Set origin for scrolling
	v.SetOrigin(0, lp.Panel.originY)

	// Set cursor position relative to origin
	cy := lp.selectedIndex - lp.Panel.originY
	if cy >= 0 && cy < innerHeight && cy < len(lp.items) {
		v.SetCursor(0, cy)
	} else {
		v.SetCursor(0, -1) // Hide cursor if out of view
	}

	return nil
}

// renderTextPanel renders a TextPanel's content into the TabPanel view
func (t *TabPanel) renderTextPanel(v *gocui.View, tp *TextPanel) error {
	// Disable highlight and hide cursor for text panels
	v.Highlight = false
	v.SelBgColor = gocui.ColorDefault
	v.SetCursor(0, -1)

	// Write content first
	v.Write([]byte(tp.content))

	// Adjust origin after rendering content
	tp.Panel.AdjustOrigin(v)

	// Set origin for scrolling
	v.SetOrigin(0, tp.Panel.originY)

	return nil
}

// ScrollUp delegates scrolling to the active tab's component
func (t *TabPanel) ScrollUp() {
	child := t.GetActiveChild()
	if child == nil {
		return
	}

	// Get child's Panel and scroll up
	panel := t.getChildPanel(child)
	if panel != nil && panel.originY > 0 {
		panel.originY--
	}
}

// ScrollDown delegates scrolling to the active tab's component
func (t *TabPanel) ScrollDown() {
	child := t.GetActiveChild()
	if child == nil {
		return
	}

	// Get TabPanel's view to calculate maxOrigin
	v := t.GetView(nil)
	if v == nil {
		return
	}

	// Get child's Panel and scroll down
	panel := t.getChildPanel(child)
	if panel == nil {
		return
	}

	// Calculate maxOrigin from rendered content
	contentLines := len(v.ViewBufferLines())
	_, h := v.Size()
	innerHeight := h - 2

	maxOrigin := contentLines - innerHeight
	if maxOrigin < 0 {
		maxOrigin = 0
	}

	// Only scroll if we haven't reached the bottom
	if panel.originY < maxOrigin {
		panel.originY++
	}
}

// ScrollToTop delegates scrolling to the active tab's component
func (t *TabPanel) ScrollToTop() {
	child := t.GetActiveChild()
	if child == nil {
		return
	}

	panel := t.getChildPanel(child)
	if panel == nil {
		return
	}

	panel.ScrollToTop()
	logDebugf("TabPanel %s: ScrollToTop delegated to child %s", t.ID(), child.ID())
}

// ScrollToBottom delegates scrolling to the active tab's component
// If contentLines is provided (> 0), use it; otherwise use ViewBufferLines
func (t *TabPanel) ScrollToBottom(contentLines int) {
	child := t.GetActiveChild()
	if child == nil {
		return
	}

	// Get TabPanel's view
	v := t.GetView(nil)
	if v == nil {
		return
	}

	panel := t.getChildPanel(child)
	if panel == nil {
		return
	}

	// Use provided contentLines if available, otherwise get from rendered view buffer
	if contentLines <= 0 {
		contentLines = len(v.ViewBufferLines())
	}

	_, viewHeight := v.Size()
	innerHeight := viewHeight - 2 // Exclude frame (top + bottom)

	// Calculate maxOrigin
	maxOrigin := contentLines - innerHeight
	if maxOrigin < 0 {
		maxOrigin = 0
	}

	// Set origin to bottom
	panel.originY = maxOrigin

	logDebugf("TabPanel %s: ScrollToBottom delegated to child %s (contentLines=%d, maxOrigin=%d)",
		t.ID(), child.ID(), contentLines, maxOrigin)
}

// getChildPanel extracts the Panel from a child component
func (t *TabPanel) getChildPanel(child UIComponent) *Panel {
	switch c := child.(type) {
	case *ListPanel:
		return c.Panel
	case *TextPanel:
		return c.Panel
	default:
		return nil
	}
}

// Container interface implementation
func (t *TabPanel) Children() []UIComponent {
	children := make([]UIComponent, len(t.tabs))
	for i, tab := range t.tabs {
		children[i] = tab.Component
	}
	return children
}

func (t *TabPanel) AddChild(child UIComponent, keybindings []Keybinding) {
	// Extract title from child component
	var title string
	var defaultBindings []Keybinding

	switch c := child.(type) {
	case *ListPanel:
		title = c.Panel.title
		// Add default keybindings for ListPanel
		defaultBindings = []Keybinding{
			{
				Key:      gocui.KeyArrowUp,
				Modifier: gocui.ModNone,
				Handler: func(g *gocui.Gui, v *gocui.View) error {
					// Manual implementation: SelectPrev + auto-scroll
					if c.selectedIndex > 0 {
						c.selectedIndex--

						// Auto-scroll up if selected item moves above visible area
						if c.selectedIndex < c.Panel.originY {
							c.Panel.originY = c.selectedIndex
							logDebugf("SelectPrev: SCROLLED UP! new originY=%d", c.Panel.originY)
						}

						logDebugf("SelectPrev: selectedIndex=%d, originY=%d", c.selectedIndex, c.Panel.originY)
					}
					return nil
				},
				DisplayName: "↑",
				Description: "prev item",
				Displayable: false,
			},
			{
				Key:      gocui.KeyArrowDown,
				Modifier: gocui.ModNone,
				Handler: func(g *gocui.Gui, v *gocui.View) error {
					// Manual implementation: SelectNext + auto-scroll
					if c.selectedIndex < len(c.items)-1 {
						c.selectedIndex++

						// Auto-scroll using TabPanel's view
						if v != nil {
							_, h := v.Size()
							innerHeight := h - 2 // Account for frame (top + bottom)
							// Scroll down if selected item moves below visible area
							// cy is the cursor Y position relative to origin (0-based)
							cy := c.selectedIndex - c.Panel.originY

							logDebugf("SelectNext: selectedIndex=%d, originY=%d, cy=%d, innerHeight=%d, h=%d",
								c.selectedIndex, c.Panel.originY, cy, innerHeight, h)

							if cy >= innerHeight {
								c.Panel.originY = c.selectedIndex - innerHeight + 1
								logDebugf("SelectNext: SCROLLED! new originY=%d", c.Panel.originY)
							}
						} else {
							logDebugf("SelectNext: v is nil!")
						}
					}
					return nil
				},
				DisplayName: "↓",
				Description: "next item",
				Displayable: false,
			},
		}
	case *TextPanel:
		title = c.Panel.title
		// Add default keybindings for TextPanel
		defaultBindings = []Keybinding{
			{
				Key:      gocui.KeyArrowUp,
				Modifier: gocui.ModNone,
				Handler: func(g *gocui.Gui, v *gocui.View) error {
					// Manual implementation: scroll up
					if c.Panel.originY > 0 {
						c.Panel.originY--
						logDebugf("TextPanel ScrollUp: new originY=%d", c.Panel.originY)
					}
					return nil
				},
				DisplayName: "↑",
				Description: "scroll up",
				Displayable: false,
			},
			{
				Key:      gocui.KeyArrowDown,
				Modifier: gocui.ModNone,
				Handler: func(g *gocui.Gui, v *gocui.View) error {
					// Manual implementation: scroll down
					if v != nil {
						// Get content lines from the rendered view buffer
						contentLines := len(v.ViewBufferLines())
						_, h := v.Size()
						innerHeight := h - 2

						// Calculate maxOrigin
						maxOrigin := contentLines - innerHeight
						if maxOrigin < 0 {
							maxOrigin = 0
						}

						// Only scroll if we haven't reached the bottom
						if c.Panel.originY < maxOrigin {
							c.Panel.originY++
							logDebugf("TextPanel ScrollDown: new originY=%d, maxOrigin=%d, contentLines=%d",
								c.Panel.originY, maxOrigin, contentLines)
						}
					}
					return nil
				},
				DisplayName: "↓",
				Description: "scroll down",
				Displayable: false,
			},
		}
	default:
		// Fallback to component ID if type unknown
		title = child.ID()
	}

	// Merge default bindings with user-provided bindings
	// User bindings take precedence (override defaults with same key)
	allBindings := append(defaultBindings, keybindings...)

	tab := TabItem{
		Title:       title,
		Component:   child,
		Keybindings: allBindings,
	}
	t.tabs = append(t.tabs, tab)
	logDebugf("TabPanel %s: Added tab '%s' with component %s (total: %d)", t.ID(), title, child.ID(), len(t.tabs))
}

func (t *TabPanel) GetActiveChild() UIComponent {
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		return t.tabs[t.activeIndex].Component
	}
	return nil
}

// SwitchToTab switches to the specified tab index (with keybinding management)
func (t *TabPanel) SwitchToTab(g *gocui.Gui, index int) {
	t.switchTab(g, index)
}

// NextTab switches to the next tab (wraps around)
func (t *TabPanel) NextTab(g *gocui.Gui) {
	if len(t.tabs) == 0 {
		return
	}
	nextIndex := (t.activeIndex + 1) % len(t.tabs)
	t.switchTab(g, nextIndex)
}

// PrevTab switches to the previous tab (wraps around)
func (t *TabPanel) PrevTab(g *gocui.Gui) {
	if len(t.tabs) == 0 {
		return
	}
	prevIndex := t.activeIndex - 1
	if prevIndex < 0 {
		prevIndex = len(t.tabs) - 1
	}
	t.switchTab(g, prevIndex)
}

// SwitchToTabByID switches to the tab with the given child component ID
func (t *TabPanel) SwitchToTabByID(g *gocui.Gui, childID string) {
	for i, tab := range t.tabs {
		if tab.Component.ID() == childID {
			t.switchTab(g, i)
			return
		}
	}
	logDebugf("TabPanel %s: Tab with child ID %s not found", t.ID(), childID)
}

// GetActiveTabIndex returns the currently active tab index
func (t *TabPanel) GetActiveTabIndex() int {
	return t.activeIndex
}

// GetTabCount returns the total number of tabs
func (t *TabPanel) GetTabCount() int {
	return len(t.tabs)
}

// switchTab switches to the specified tab index and manages keybindings
func (t *TabPanel) switchTab(g *gocui.Gui, newIndex int) {
	if newIndex < 0 || newIndex >= len(t.tabs) || newIndex == t.activeIndex {
		return
	}

	// 1. Delete previous tab's keybindings
	for _, kb := range t.tabs[t.activeIndex].Keybindings {
		g.DeleteKeybinding(t.ID(), kb.Key, kb.Modifier)
		logDebugf("TabPanel %s: Deleted keybinding %v for tab %d", t.ID(), kb.Key, t.activeIndex)
	}

	// 2. Clear previous child's keybindings from StatusBar
	if t.app != nil && t.app.viewKeybindings != nil {
		prevChild := t.tabs[t.activeIndex].Component
		delete(t.app.viewKeybindings, prevChild.ID())
		logDebugf("TabPanel %s: Cleared StatusBar keybindings for child %s", t.ID(), prevChild.ID())
	}

	// 3. Update active index
	t.activeIndex = newIndex
	logDebugf("TabPanel %s: Switched to tab %d (%s)", t.ID(), newIndex, t.tabs[newIndex].Title)

	// 4. Register new tab's keybindings to gocui
	for _, kb := range t.tabs[newIndex].Keybindings {
		err := g.SetKeybinding(t.ID(), kb.Key, kb.Modifier, kb.Handler)
		if err != nil {
			logDebugf("TabPanel %s: Failed to register keybinding %v for tab %d: %v", t.ID(), kb.Key, newIndex, err)
		} else {
			logDebugf("TabPanel %s: Registered keybinding %v for tab %d", t.ID(), kb.Key, newIndex)
		}
	}

	// 5. Register new child's keybindings to StatusBar
	if t.app != nil {
		newChild := t.tabs[newIndex].Component
		if t.app.viewKeybindings == nil {
			t.app.viewKeybindings = make(map[string][]Keybinding)
		}
		t.app.viewKeybindings[newChild.ID()] = t.tabs[newIndex].Keybindings
		logDebugf("TabPanel %s: Registered StatusBar keybindings for child %s", t.ID(), newChild.ID())
	}
}

// RegisterBindings adds tab panel specific keybindings
func (t *TabPanel) RegisterBindings(g *gocui.Gui, app *App) error {
	// Register mouse wheel bindings directly for TabPanel
	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: t.ID(),
		Key:      gocui.MouseWheelUp,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			// Ignore if modal is open
			if app.IsModalOpen {
				return nil
			}
			t.ScrollUp()
			return nil
		},
	})

	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: t.ID(),
		Key:      gocui.MouseWheelDown,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			// Ignore if modal is open
			if app.IsModalOpen {
				return nil
			}
			t.ScrollDown()
			return nil
		},
	})

	// Register mouse left click for item selection (if active tab is ListPanel)
	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: t.ID(),
		Key:      gocui.MouseLeft,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			return t.handleContentClick(g, app, opts)
		},
	})

	logDebugf("Mouse bindings registered for TabPanel: %s", t.ID())

	// Tab: next tab
	if err := app.RegisterViewKeybinding(t.ID(), Keybinding{
		Key:      gocui.KeyTab,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			t.NextTab(g)
			return nil
		},
		DisplayName: "Tab",
		Description: "next tab",
		Displayable: true,
	}); err != nil {
		return err
	}

	// Shift+Tab (Backtab): previous tab
	if err := app.RegisterViewKeybinding(t.ID(), Keybinding{
		Key:      gocui.KeyBacktab,
		Modifier: gocui.ModNone,
		Handler: func(g *gocui.Gui, v *gocui.View) error {
			t.PrevTab(g)
			return nil
		},
		DisplayName: "Shift+Tab",
		Description: "prev tab",
		Displayable: true,
	}); err != nil {
		return err
	}

	// Mouse click on tab bar using SetTabClickBinding
	g.SetTabClickBinding(t.ID(), func(tabIndex int) error {
		return t.handleTabClick(g, app, tabIndex)
	})

	// Register initial active tab's keybindings to gocui (index 0)
	// Note: StatusBar registration is handled by App.RegisterTabPanel()
	if len(t.tabs) > 0 {
		for _, kb := range t.tabs[0].Keybindings {
			err := g.SetKeybinding(t.ID(), kb.Key, kb.Modifier, kb.Handler)
			if err != nil {
				logDebugf("TabPanel %s: Failed to register initial keybinding %v: %v", t.ID(), kb.Key, err)
			} else {
				logDebugf("TabPanel %s: Registered initial keybinding %v", t.ID(), kb.Key)
			}
		}
	}

	logDebugf("Tab navigation bindings registered for TabPanel: %s", t.ID())
	return nil
}

// handleTabClick handles mouse click on tab bar
func (t *TabPanel) handleTabClick(g *gocui.Gui, app *App, tabIndex int) error {
	// Ignore if modal is open
	if app.IsModalOpen {
		logDebug("handleTabClick: modal is open, ignoring")
		return nil
	}

	logDebugf("TabPanel click: tabIndex=%d, current activeIndex=%d", tabIndex, t.activeIndex)

	// Switch to the clicked tab using switchTab (handles keybindings)
	t.switchTab(g, tabIndex)

	// Update view's TabIndex
	v := t.GetView(g)
	if v != nil {
		v.TabIndex = tabIndex
	}

	// Handle focus switch
	focusables := app.getFocusableComponents()
	for i, f := range focusables {
		if f.ID() == t.ID() {
			// Already focused, just update tab
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
			g.SetCurrentView(t.ID())

			logDebugf("Focus switched to TabPanel: %s (index: %d)", t.ID(), i)
			return nil
		}
	}

	return nil
}

// handleContentClick handles mouse click on TabPanel content area
func (t *TabPanel) handleContentClick(g *gocui.Gui, app *App, opts gocui.ViewMouseBindingOpts) error {
	// Ignore if modal is open
	if app.IsModalOpen {
		return nil
	}

	logDebugf("TabPanel handleContentClick: Y=%d, activeIndex=%d", opts.Y, t.activeIndex)

	// Handle focus switch first (same as app.handlePanelClick)
	focusables := app.getFocusableComponents()
	for i, f := range focusables {
		if f.ID() == t.ID() {
			// Not focused yet, switch focus
			if i != app.currentFocusIdx {
				// Blur current focus
				if app.currentFocusIdx >= 0 && app.currentFocusIdx < len(focusables) {
					focusables[app.currentFocusIdx].OnBlur(g)
				}

				// Update focus
				app.currentFocusIdx = i
				focusables[i].OnFocus(g)
				g.SetCurrentView(t.ID())

				logDebugf("Focus switched to TabPanel: %s", t.ID())
			}
			break
		}
	}

	// Get active tab
	activeChild := t.GetActiveChild()
	if activeChild == nil {
		return nil
	}

	// Only handle item selection for ListPanel
	lp, ok := activeChild.(*ListPanel)
	if !ok {
		return nil
	}

	// opts.Y is content-relative Y coordinate (includes origin offset)
	clickedIndex := opts.Y

	logDebugf("TabPanel content click on ListPanel: Y=%d, clickedIndex=%d, itemsLen=%d",
		opts.Y, clickedIndex, len(lp.items))

	// Validate index
	if clickedIndex >= 0 && clickedIndex < len(lp.items) {
		lp.selectedIndex = clickedIndex
		logDebugf("TabPanel: selected item %d (%s)", clickedIndex, lp.items[clickedIndex].DisplayText)
	}

	return nil
}
