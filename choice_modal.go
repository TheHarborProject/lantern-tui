package main

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

type ChoiceModal struct {
	*BaseModal
	multi         bool
	options       []string
	selectedIndex int
	selected      map[string]bool
	onSave        func([]string)
}

func NewChoiceModal(id, title string, multi bool, app *App) *ChoiceModal {
	footer := "↑/↓ select · Enter save · Esc cancel"
	if multi {
		footer = "↑/↓ select · Space toggle · Enter save · Esc cancel"
	}
	modal := &ChoiceModal{BaseModal: NewBaseModal(id, title, "", footer, ModalTypeInfo, app), multi: multi, selected: map[string]bool{}}
	modal.BaseModal.self = modal
	return modal
}

func (m *ChoiceModal) GetBaseModal() *BaseModal { return m.BaseModal }

func (m *ChoiceModal) ShowOptions(options, selected []string, onSave func([]string)) {
	m.options = append([]string(nil), options...)
	m.selected = map[string]bool{}
	for _, value := range selected {
		m.selected[value] = true
	}
	m.selectedIndex = 0
	for i, option := range m.options {
		if m.selected[option] {
			m.selectedIndex = i
			break
		}
	}
	m.onSave = onSave
	m.height = len(options) + 4
	m.BaseModal.Show("")
}

func (m *ChoiceModal) SelectNext() {
	if m.selectedIndex < len(m.options)-1 {
		m.selectedIndex++
	}
}
func (m *ChoiceModal) SelectPrev() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

func (m *ChoiceModal) ToggleSelected() {
	if !m.multi || m.selectedIndex < 0 || m.selectedIndex >= len(m.options) {
		return
	}
	option := m.options[m.selectedIndex]
	m.selected[option] = !m.selected[option]
}

func (m *ChoiceModal) SelectedValues() []string {
	if !m.multi {
		if m.selectedIndex >= 0 && m.selectedIndex < len(m.options) {
			return []string{m.options[m.selectedIndex]}
		}
		return nil
	}
	values := make([]string, 0)
	for _, option := range m.options {
		if m.selected[option] {
			values = append(values, option)
		}
	}
	return values
}

func (m *ChoiceModal) SaveAndClose() {
	if m.onSave != nil {
		m.onSave(m.SelectedValues())
	}
	m.Hide()
}

func (m *ChoiceModal) Render(g *gocui.Gui, dim Dimension) error {
	if !m.visible {
		if v, err := g.View(m.id); err == nil {
			v.Visible = false
		}
		return nil
	}
	width, height := g.Size()
	modalWidth := 50
	if width-2 < modalWidth {
		modalWidth = width - 2
	}
	x0, x1 := width/2-modalWidth/2, width/2+modalWidth/2
	y0, y1 := height/2-m.height/2, height/2+m.height/2
	v, err := g.SetView(m.id, x0, y0, x1, y1, 0)
	if err != nil && err.Error() != gocui.ErrUnknownView.Error() {
		return err
	}
	m.SetView(v)
	v.Visible = true
	m.applyStyle(v)
	v.Clear()
	for i, option := range m.options {
		cursor := "  "
		if i == m.selectedIndex {
			cursor = "> "
		}
		marker := ""
		if m.multi {
			marker = "[ ] "
			if m.selected[option] {
				marker = "[x] "
			}
		}
		fmt.Fprintf(v, "%s%s%s\n", cursor, marker, option)
	}
	return nil
}

func (m *ChoiceModal) RegisterBindings(g *gocui.Gui, app *App) error {
	if err := m.BaseModal.RegisterBindings(g, app); err != nil {
		return err
	}
	bindings := []struct {
		key     interface{}
		handler func()
	}{
		{gocui.KeyArrowUp, m.SelectPrev}, {gocui.KeyArrowDown, m.SelectNext},
		{gocui.KeySpace, m.ToggleSelected}, {gocui.KeyEnter, m.SaveAndClose},
		{gocui.KeyEsc, m.Hide},
	}
	for _, binding := range bindings {
		fn := binding.handler
		if err := g.SetKeybinding(m.ID(), binding.key, gocui.ModNone, func(*gocui.Gui, *gocui.View) error { fn(); return nil }); err != nil {
			return err
		}
	}
	return nil
}
