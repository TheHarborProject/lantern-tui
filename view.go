package main

import "github.com/jesseduffield/gocui"

// Dimension represents the bounding box of a view (calculated by lazycore)
type Dimension struct {
	X0, Y0, X1, Y1 int
}

// ============================================================================
// Interface Hierarchy
// ============================================================================

// UIComponent is the minimal interface that all UI elements must implement.
type UIComponent interface {
	ID() string
	Render(g *gocui.Gui, dim Dimension) error
	IsVisible() bool
	SetVisible(bool)
	RegisterBindings(g *gocui.Gui, app *App) error // Register component-specific bindings
}

// Focusable represents a component that can receive focus.
// Focus state affects visual styling (e.g., frame colour).
type Focusable interface {
	UIComponent
	OnFocus(g *gocui.Gui)
	OnBlur(g *gocui.Gui)
	IsFocused() bool
}

// Scrollable represents a component that supports vertical scrolling.
type Scrollable interface {
	ScrollUp()
	ScrollDown()
	GetOrigin() (int, int)
	SetOrigin(int, int)
}

// Selectable represents a component that manages item selection (e.g., ListView).
type Selectable interface {
	GetSelectedIndex() int
	SetSelectedIndex(int)
	SelectNext()
	SelectPrev()
}

// Container represents a component that contains child components (e.g., TabPanel).
type Container interface {
	Focusable
	Children() []UIComponent
	AddChild(child UIComponent, keybindings []Keybinding)
	GetActiveChild() UIComponent
}

// ============================================================================
// Base View Implementation
// ============================================================================

// BaseView provides common fields and methods for all UI components.
// It wraps gocui.View and provides basic visibility management.
type BaseView struct {
	id      string
	v       *gocui.View
	visible bool
}

func NewBaseView(id string, visible bool) *BaseView {
	return &BaseView{
		id:      id,
		visible: visible,
	}
}

func (b *BaseView) ID() string {
	return b.id
}

func (b *BaseView) IsVisible() bool {
	return b.visible
}

func (b *BaseView) SetVisible(visible bool) {
	b.visible = visible
	if b.v != nil {
		b.v.Visible = visible
	}
}

// GetView returns the underlying gocui.View, or attempts to fetch it if nil.
// Returns nil if g is nil or view doesn't exist yet.
func (b *BaseView) GetView(g *gocui.Gui) *gocui.View {
	if b.v == nil && g != nil {
		v, err := g.View(b.id)
		if err == nil {
			b.v = v
		}
	}
	return b.v
}

// SetView assigns the gocui.View after it's created.
func (b *BaseView) SetView(v *gocui.View) {
	b.v = v
}

// Render is a no-op for BaseView; concrete types override this.
func (b *BaseView) Render(g *gocui.Gui, dim Dimension) error {
	return nil
}

// RegisterBindings is a no-op for BaseView; concrete types override this.
func (b *BaseView) RegisterBindings(g *gocui.Gui, app *App) error {
	return nil
}
