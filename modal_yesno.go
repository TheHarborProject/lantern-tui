package main

import "github.com/jesseduffield/gocui"

// ============================================================================
// YesNoModal - Modal with Yes/No choice (Y/N to select, ESC to cancel)
// ============================================================================

type YesNoModal struct {
	*BaseModal
	onYes func() // Called when user selects Yes
	onNo  func() // Called when user selects No (optional)
}

// NewYesNoModal creates a new Yes/No modal
// Footer is automatically set to guide user interaction
func NewYesNoModal(id, title string, modalType ModalType, app *App) *YesNoModal {
	modal := &YesNoModal{
		BaseModal: NewBaseModal(id, title, "", "Press Y for Yes, N for No, or ESC to cancel", modalType, app),
	}
	// Set self reference for proper currentModal assignment
	modal.BaseModal.self = modal
	logDebugf("YesNoModal created: id=%s, title=%s, type=%d, footer=%s", id, title, modalType, modal.Panel.footer)
	return modal
}

// GetBaseModal returns the embedded BaseModal (for ModalComponent interface)
func (m *YesNoModal) GetBaseModal() *BaseModal {
	return m.BaseModal
}

// SetOnYes registers a callback to execute when user selects Yes
func (m *YesNoModal) SetOnYes(fn func()) {
	m.onYes = fn
}

// SetOnNo registers a callback to execute when user selects No
func (m *YesNoModal) SetOnNo(fn func()) {
	m.onNo = fn
}

// RegisterBindings adds Y/N key bindings in addition to ESC/Q
func (m *YesNoModal) RegisterBindings(g *gocui.Gui, app *App) error {
	// Call parent RegisterBindings for ESC and 'q'
	if err := m.BaseModal.RegisterBindings(g, app); err != nil {
		return err
	}

	// 'y' or 'Y': Yes
	err := g.SetKeybinding(m.ID(), 'y', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("YesNoModal %s: 'y' pressed, user selected Yes", m.ID())
			if m.onYes != nil {
				m.onYes()
			}
			m.Hide()
			return nil
		})
	if err != nil {
		return err
	}

	err = g.SetKeybinding(m.ID(), 'Y', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("YesNoModal %s: 'Y' pressed, user selected Yes", m.ID())
			if m.onYes != nil {
				m.onYes()
			}
			m.Hide()
			return nil
		})
	if err != nil {
		return err
	}

	// 'n' or 'N': No
	err = g.SetKeybinding(m.ID(), 'n', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("YesNoModal %s: 'n' pressed, user selected No", m.ID())
			if m.onNo != nil {
				m.onNo()
			}
			m.Hide()
			return nil
		})
	if err != nil {
		return err
	}

	err = g.SetKeybinding(m.ID(), 'N', gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			logDebugf("YesNoModal %s: 'N' pressed, user selected No", m.ID())
			if m.onNo != nil {
				m.onNo()
			}
			m.Hide()
			return nil
		})
	if err != nil {
		return err
	}

	logDebugf("YesNoModal %s: Registered Y/N keybindings", m.ID())
	return nil
}
