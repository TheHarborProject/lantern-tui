package main

import "github.com/jesseduffield/gocui"

// ============================================================================
// OkOnlyModal - Modal with only OK button (Enter/ESC to close)
// ============================================================================

type OkOnlyModal struct {
	*BaseModal
}

// NewOkOnlyModal creates a new OK-only modal
// Footer is automatically set to guide user interaction
func NewOkOnlyModal(id, title string, modalType ModalType, app *App) *OkOnlyModal {
	modal := &OkOnlyModal{
		BaseModal: NewBaseModal(id, title, "", "Press Enter or ESC to close", modalType, app),
	}
	// Set self reference for proper currentModal assignment
	modal.BaseModal.self = modal
	logDebugf("OkOnlyModal created: id=%s, title=%s, type=%d, footer=%s", id, title, modalType, modal.Panel.footer)
	return modal
}

// GetBaseModal returns the embedded BaseModal (for ModalComponent interface)
func (m *OkOnlyModal) GetBaseModal() *BaseModal {
	return m.BaseModal
}

// RegisterBindings adds Enter key binding in addition to ESC
func (m *OkOnlyModal) RegisterBindings(g *gocui.Gui, app *App) error {
	// Call parent RegisterBindings for ESC
	if err := m.BaseModal.RegisterBindings(g, app); err != nil {
		return err
	}

	// Enter: close modal
	err := g.SetKeybinding(m.ID(), gocui.KeyEnter, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("OkOnlyModal %s: Enter pressed, closing modal", m.ID())
			m.Hide()
			return nil
		})

	if err != nil {
		return err
	}

	logDebugf("OkOnlyModal %s: Registered Enter keybinding", m.ID())
	return nil
}
