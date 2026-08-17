package main

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

// ============================================================================
// InputModal - Modal for text input (Enter to submit, ESC to cancel)
// ============================================================================

type InputModal struct {
	*BaseModal
	onSubmit         func(input string) // Called when user submits input (Enter)
	onCancel         func()             // Called when user cancels (ESC) - optional
	onValidateSubmit func(input string) error
	initialValue     string
	maxLength        int  // Maximum input length (0 = unlimited)
	spaceReplacement rune // If set, replaces space with this character (0 = allow spaces)
}

// NewInputModal creates a new input modal
// Footer is automatically set to guide user interaction
// spaceReplacement: if non-zero, replaces spaces with this character (e.g., '_')
func NewInputModal(id, title string, modalType ModalType, spaceReplacement rune, app *App) *InputModal {
	modal := &InputModal{
		BaseModal:        NewBaseModal(id, title, "", "Press Enter to submit or ESC to cancel", modalType, app),
		maxLength:        0,                // unlimited by default
		spaceReplacement: spaceReplacement, // 0 = allow spaces
	}
	// Set self reference for proper currentModal assignment
	modal.BaseModal.self = modal
	logDebugf("InputModal created: id=%s, title=%s, type=%d, spaceReplacement=%q, footer=%s",
		id, title, modalType, spaceReplacement, modal.Panel.footer)
	return modal
}

// GetBaseModal returns the embedded BaseModal (for ModalComponent interface)
func (m *InputModal) GetBaseModal() *BaseModal {
	return m.BaseModal
}

// SetOnSubmit registers a callback to execute when user submits input
func (m *InputModal) SetOnSubmit(fn func(input string)) {
	m.onSubmit = fn
}

// SetOnCancel registers a callback to execute when user cancels
func (m *InputModal) SetOnCancel(fn func()) {
	m.onCancel = fn
}

func (m *InputModal) SetOnValidateSubmit(fn func(input string) error) {
	m.onValidateSubmit = fn
}

func (m *InputModal) SetInitialValue(value string) {
	m.initialValue = value
}

func (m *InputModal) InitialValue() string {
	return m.initialValue
}

// SetMaxLength sets the maximum input length (0 = unlimited)
func (m *InputModal) SetMaxLength(length int) {
	m.maxLength = length
}

// Show displays the modal and prepares for input
func (m *InputModal) Show(message string) {
	// Set message as subtitle instead of content
	m.Panel.subtitle = message
	m.message = "" // Clear message so it doesn't render in buffer

	// Call parent Show with empty message
	m.BaseModal.Show("")
	if m.app.g == nil {
		return
	}

	// Enable cursor globally for input
	m.app.g.Cursor = true

	// Clear input and set cursor to start position
	m.app.g.Update(func(g *gocui.Gui) error {
		v := m.GetView(g)
		if v != nil {
			// Clear TextArea (the actual editable buffer, not just View buffer)
			v.TextArea.Clear()
			v.TextArea.TypeString(m.initialValue)
			v.RenderTextArea()

			v.SetCursor(len([]rune(m.initialValue)), 0)
			v.SetOrigin(0, 0)

			// Enable editing immediately
			v.Editable = true
			// Use custom editor if spaceReplacement is set, otherwise use DefaultEditor
			if m.spaceReplacement != 0 {
				v.Editor = gocui.EditorFunc(m.customEditor)
			} else {
				v.Editor = gocui.DefaultEditor
			}

			logDebugf("InputModal %s: TextArea cleared, reset cursor, and enabled editing (spaceReplacement=%q)", m.ID(), m.spaceReplacement)
		}
		// Set focus to modal for input
		g.SetCurrentView(m.ID())
		return nil
	})

	logDebugf("InputModal %s: Cursor enabled", m.ID())
}

// Hide closes the modal and disables cursor
func (m *InputModal) Hide() {
	if m.app.g == nil {
		m.BaseModal.Hide()
		return
	}
	// Disable cursor when modal closes
	m.app.g.Cursor = false
	logDebugf("InputModal %s: Cursor disabled", m.ID())

	// Call parent Hide
	m.BaseModal.Hide()
}

// customEditor handles input with space replacement
func (m *InputModal) customEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	logDebugf("customEditor called: key=%v, ch=%q, spaceReplacement=%q", key, ch, m.spaceReplacement)

	// Handle space key specially (key=32 for spacebar, ch may be ' ' or '\x00')
	if (key == gocui.KeySpace || ch == ' ') && m.spaceReplacement != 0 {
		logDebugf("Space detected! Replacing with %q", m.spaceReplacement)
		// Replace space with the configured character
		v.TextArea.TypeRune(m.spaceReplacement)
		v.RenderTextArea()
		return true
	}

	// For all other keys, use default handling
	matched := gocui.DefaultEditor.Edit(v, key, ch, mod)
	return matched
}

// Render implements UIComponent interface
// InputModal overrides BaseModal.Render() completely to avoid v.Clear() resetting Editable
func (m *InputModal) Render(g *gocui.Gui, dim Dimension) error {
	if !m.visible {
		if v, err := g.View(m.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	// Calculate center overlay dimensions (same as BaseModal)
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

	// Height
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

	// Set subtitle and footer
	v.Subtitle = m.Panel.subtitle
	v.Footer = m.Panel.footer

	// DO NOT call v.Clear() here - it resets Editable!
	// Enable editing
	v.Editable = true
	// Set editor based on spaceReplacement setting
	if m.spaceReplacement != 0 {
		v.Editor = gocui.EditorFunc(m.customEditor)
	} else {
		v.Editor = gocui.DefaultEditor
	}

	logDebugf("InputModal %s: Rendered with Editable=true, Editor set", m.ID())
	return nil
}

// RegisterBindings adds Enter key binding for submit
func (m *InputModal) RegisterBindings(g *gocui.Gui, app *App) error {
	// Call parent RegisterBindings for ESC and 'q'
	if err := m.BaseModal.RegisterBindings(g, app); err != nil {
		return err
	}

	// Override ESC to call onCancel if set
	err := g.SetKeybinding(m.ID(), gocui.KeyEsc, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("InputModal %s: ESC pressed, canceling input", m.ID())
			if m.onCancel != nil {
				m.onCancel()
			}
			m.Hide()
			return nil
		})
	if err != nil {
		return err
	}

	// Enter: submit input
	err = g.SetKeybinding(m.ID(), gocui.KeyEnter, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			// Read input from view buffer
			input := strings.TrimSpace(v.Buffer())
			logDebugf("InputModal %s: Enter pressed, input=%s", m.ID(), input)

			m.SubmitAndClose(input)
			return nil
		})
	if err != nil {
		return err
	}

	logDebugf("InputModal %s: Registered Enter keybinding", m.ID())
	return nil
}

func (m *InputModal) SubmitAndClose(input string) bool {
	if !m.Submit(input) {
		return false
	}
	m.Hide()
	return true
}

func (m *InputModal) Submit(input string) bool {
	if m.onValidateSubmit != nil {
		if err := m.onValidateSubmit(input); err != nil {
			m.Panel.subtitle = fmt.Sprintf("Error: %v", err)
			return false
		}
	}
	if m.onSubmit != nil {
		m.onSubmit(input)
	}
	return true
}
