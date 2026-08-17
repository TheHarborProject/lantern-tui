package main

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// ============================================================================
// ModalType - Modal type enum for different visual styles
// ============================================================================

type ModalType int

const (
	ModalTypeInfo    ModalType = iota // Cyan (정보)
	ModalTypeWarning                  // Yellow (경고)
	ModalTypeError                    // Red (에러/주의)
	ModalTypeSuccess                  // Green (성공)
)

// ============================================================================
// BaseModal - Common modal functionality (Panel-based)
// ============================================================================

// BaseModal provides common modal functionality for all modal types.
// Child modals should embed this and add their specific behaviour.
// BaseModal embeds Panel to reuse frame, title, subtitle, footer functionality.
type BaseModal struct {
	*Panel
	app       *App      // App reference for focus management
	message   string    // Message to display
	modalType ModalType // Modal type for color styling

	// Layout dimensions (calculated in Render)
	width  int
	height int

	// Callbacks
	onClose func() // Called when modal closes (optional)

	// Self reference for currentModal assignment
	self ModalComponent
}

// NewBaseModal creates a new base modal
func NewBaseModal(id, title, subtitle, footer string, modalType ModalType, app *App) *BaseModal {
	return &BaseModal{
		Panel:     NewPanel(id, title, subtitle, footer, false), // initially hidden
		app:       app,
		message:   "",
		modalType: modalType,
		width:     0,
		height:    5, // Default height
	}
}

// ============================================================================
// Public API
// ============================================================================

// Show displays the modal and saves previous focus
func (m *BaseModal) Show(message string) {
	m.message = message

	// Save previous focus
	if m.app.g == nil {
		m.SetVisible(true)
		m.app.IsModalOpen = true
		m.app.currentModal = m.self
		return
	}
	if cv := m.app.g.CurrentView(); cv != nil {
		m.app.previousView = cv.Name()
		logDebugf("BaseModal %s: Saved previous view: %s", m.ID(), m.app.previousView)
	} else {
		logDebugf("BaseModal %s: No previous view to save", m.ID())
	}

	// Set modal state
	m.SetVisible(true)
	m.app.IsModalOpen = true
	// Use self reference if available, otherwise use self (for BaseModal used directly)
	if m.self != nil {
		m.app.currentModal = m.self
	} else {
		m.app.currentModal = m
	}

	logDebugf("BaseModal %s: Shown with message: %s", m.ID(), message)
}

// Hide closes the modal and restores previous focus
func (m *BaseModal) Hide() {
	m.SetVisible(false)
	m.app.IsModalOpen = false
	m.app.currentModal = nil // Clear current modal

	// Execute onClose callback if set
	if m.onClose != nil {
		logDebugf("BaseModal %s: Executing onClose callback", m.ID())
		m.onClose()
	}

	// Restore previous focus
	if m.app.g != nil && m.app.previousView != "" {
		m.app.g.Update(func(g *gocui.Gui) error {
			_, err := g.SetCurrentView(m.app.previousView)
			if err != nil {
				logDebugf("BaseModal %s: Failed to restore focus to %s: %v",
					m.ID(), m.app.previousView, err)
			} else {
				logDebugf("BaseModal %s: Restored focus to %s", m.ID(), m.app.previousView)
			}
			return nil
		})
	} else {
		logDebugf("BaseModal %s: No previous view to restore", m.ID())
	}

	logDebugf("BaseModal %s: Hidden", m.ID())
}

// SetOnClose registers a callback to execute when modal closes
func (m *BaseModal) SetOnClose(fn func()) {
	m.onClose = fn
}

// GetMessage returns the current message
func (m *BaseModal) GetMessage() string {
	return m.message
}

// GetBaseModal returns self (for ModalComponent interface)
func (m *BaseModal) GetBaseModal() *BaseModal {
	return m
}

// ============================================================================
// Rendering (Common Layout)
// ============================================================================

// Render implements UIComponent interface.
// Child modals should call this and then add their specific content.
func (m *BaseModal) Render(g *gocui.Gui, dim Dimension) error {
	if !m.visible {
		if v, err := g.View(m.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	// Calculate center overlay dimensions
	screenWidth, screenHeight := g.Size()

	// Calculate width (4/7 of screen, min 80)
	m.width = 4 * screenWidth / 7
	minWidth := 80
	if m.width < minWidth {
		if screenWidth-2 < minWidth {
			m.width = screenWidth - 2
		} else {
			m.width = minWidth
		}
	}

	// Height should be set by child modal based on content
	// Default to 5 if not set
	if m.height == 0 {
		m.height = 5
	}

	// Center coordinates
	x0 := screenWidth/2 - m.width/2
	y0 := screenHeight/2 - m.height/2 - m.height%2 - 1
	x1 := screenWidth/2 + m.width/2
	y1 := screenHeight/2 + m.height/2

	v, err := g.SetView(m.id, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	m.SetView(v)
	v.Visible = true

	// Apply frame and title styling
	v.FrameRunes = m.Panel.frameRunes
	v.Title = m.Panel.title

	// Modal-specific colors based on modalType
	switch m.modalType {
	case ModalTypeInfo:
		v.FrameColor = gocui.ColorCyan
		v.TitleColor = gocui.ColorCyan | gocui.AttrBold
	case ModalTypeWarning:
		v.FrameColor = gocui.ColorYellow
		v.TitleColor = gocui.ColorYellow | gocui.AttrBold
	case ModalTypeError:
		v.FrameColor = gocui.ColorRed
		v.TitleColor = gocui.ColorRed | gocui.AttrBold
	case ModalTypeSuccess:
		v.FrameColor = gocui.ColorGreen
		v.TitleColor = gocui.ColorGreen | gocui.AttrBold
	default:
		v.FrameColor = gocui.ColorCyan
		v.TitleColor = gocui.ColorCyan | gocui.AttrBold
	}

	// Always set subtitle and footer from Panel fields
	v.Subtitle = m.Panel.subtitle
	v.Footer = m.Panel.footer

	v.Clear()

	// Render message
	m.RenderMessage(v)

	return nil
}

// RenderMessage renders the message text (helper for child modals)
func (m *BaseModal) RenderMessage(v *gocui.View) {
	if m.message != "" {
		fmt.Fprintf(v, "\n  %s\n\n", m.message)
	}
}

// ============================================================================
// Common Keybindings
// ============================================================================

// RegisterBindings registers ESC and 'q' to close modal (common for all modals)
func (m *BaseModal) RegisterBindings(g *gocui.Gui, app *App) error {
	// ESC: close modal
	err := g.SetKeybinding(m.ID(), gocui.KeyEsc, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("BaseModal %s: ESC pressed, closing modal", m.ID())
			m.Hide()
			return nil
		})

	if err != nil {
		return err
	}

	// 'q': close modal (same as ESC for better UX)
	err = g.SetKeybinding(m.ID(), 'q', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("BaseModal %s: 'q' pressed, closing modal", m.ID())
			m.Hide()
			return nil
		})

	if err != nil {
		return err
	}

	logDebugf("BaseModal %s: Registered ESC and 'q' keybindings", m.ID())
	return nil
}
