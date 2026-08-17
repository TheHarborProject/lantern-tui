package main

import "github.com/jesseduffield/gocui"

// ============================================================================
// Panel - Base component for lazycore layout targets
// ============================================================================

// Panel represents a component that participates in lazycore layout.
// It has frames, title, subtitle, footer metadata.
type Panel struct {
	*BaseView
	frameRunes []rune
	title      string
	subtitle   string
	footer     string
	focused    bool
	originY    int // Vertical scroll position
}

func NewPanel(id, title, subtitle, footer string, visible bool) *Panel {
	return &Panel{
		BaseView:   NewBaseView(id, visible),
		frameRunes: []rune{'─', '│', '╭', '╮', '╰', '╯'},
		title:      title,
		subtitle:   subtitle,
		footer:     footer,
		focused:    false,
		originY:    0,
	}
}

// IsFocused implements Focusable interface.
func (p *Panel) IsFocused() bool {
	return p.focused
}

// OnFocus implements Focusable interface.
// Sets frame and title to green + bold when focused.
func (p *Panel) OnFocus(g *gocui.Gui) {
	p.focused = true
	v := p.GetView(g)
	if v == nil {
		return
	}
	v.FrameColor = gocui.ColorGreen
	v.TitleColor = gocui.ColorGreen | gocui.AttrBold
}

// OnBlur implements Focusable interface.
// Resets frame and title to white when blurred.
func (p *Panel) OnBlur(g *gocui.Gui) {
	p.focused = false
	v := p.GetView(g)
	if v == nil {
		return
	}
	v.FrameColor = gocui.ColorWhite
	v.TitleColor = gocui.ColorWhite
}

// applyStyle applies common panel styling to the gocui.View.
func (p *Panel) applyStyle(v *gocui.View) {
	v.FrameRunes = p.frameRunes
	v.Title = p.title
	v.Subtitle = p.subtitle
	v.Footer = p.footer

	if p.focused {
		v.FrameColor = gocui.ColorGreen
		v.TitleColor = gocui.ColorGreen | gocui.AttrBold
	} else {
		v.FrameColor = gocui.ColorWhite
		v.TitleColor = gocui.ColorWhite
	}
}

func (p *Panel) SetFooter(footer string) {
	p.footer = footer
}

func (p *Panel) SetTitle(title string) {
	p.title = title
}

// AdjustOrigin adjusts the origin to ensure it's within valid bounds
// Call this after content is rendered but before SetOrigin
func (p *Panel) AdjustOrigin(v *gocui.View) {
	if v == nil {
		return
	}

	// Get actual content lines from the rendered view buffer
	contentLines := len(v.ViewBufferLines())
	_, viewHeight := v.Size()
	innerHeight := viewHeight - 2 // Exclude frame (top + bottom)

	// Calculate maxOrigin
	maxOrigin := contentLines - innerHeight
	if maxOrigin < 0 {
		maxOrigin = 0
	}

	// Adjust origin if it exceeds maxOrigin (e.g., after terminal resize)
	if p.originY > maxOrigin {
		p.originY = maxOrigin
	}
}

// ============================================================================
// Scrollable interface implementation for Panel (unified implementation)
// ============================================================================

func (p *Panel) ScrollUp() {
	if p.originY > 0 {
		p.originY--
	}
}

func (p *Panel) ScrollDown() {
	v := p.GetView(nil)
	if v == nil {
		return
	}

	// Get actual content lines from the rendered view buffer
	contentLines := len(v.ViewBufferLines())
	_, viewHeight := v.Size()
	innerHeight := viewHeight - 2 // Exclude frame (top + bottom)

	// Calculate maxOrigin
	maxOrigin := contentLines - innerHeight
	if maxOrigin < 0 {
		maxOrigin = 0
	}

	// Only scroll if we haven't reached the bottom
	if p.originY < maxOrigin {
		p.originY++
	}
}

// ScrollToTop scrolls to the top of the content
func (p *Panel) ScrollToTop() {
	p.originY = 0
	logDebugf("ScrollToTop: originY=%d", p.originY)
}

// ScrollToBottom scrolls to the bottom of the content
// If contentLines is provided (> 0), use it instead of ViewBufferLines (for pre-render scenarios)
func (p *Panel) ScrollToBottom(contentLines int) {
	v := p.GetView(nil)
	if v == nil {
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
	p.originY = maxOrigin

	logDebugf("ScrollToBottom: contentLines=%d, viewHeight=%d, innerHeight=%d, maxOrigin=%d, originY=%d",
		contentLines, viewHeight, innerHeight, maxOrigin, p.originY)
}

func (p *Panel) GetOrigin() (int, int) {
	return 0, p.originY
}

func (p *Panel) SetOrigin(x, y int) {
	p.originY = y
}

// ============================================================================
// RegisterBindings for Panel (mouse wheel scrolling)
// ============================================================================

func (p *Panel) RegisterBindings(g *gocui.Gui, app *App) error {
	// Mouse wheel up
	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: p.ID(),
		Key:      gocui.MouseWheelUp,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			// Ignore if modal is open
			if app.IsModalOpen {
				return nil
			}
			p.ScrollUp()
			return nil
		},
	})

	// Mouse wheel down
	g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: p.ID(),
		Key:      gocui.MouseWheelDown,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			// Ignore if modal is open
			if app.IsModalOpen {
				return nil
			}
			p.ScrollDown()
			return nil
		},
	})

	logDebugf("Mouse wheel bindings registered for Panel: %s", p.ID())
	return nil
}
