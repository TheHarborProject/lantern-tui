package main

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

type AppHeader struct {
	*BaseView
	appName    string
	workspaces []Workspace
	active     int
}

func NewAppHeader(id, appName string) *AppHeader {
	return &AppHeader{BaseView: NewBaseView(id, true), appName: appName}
}

func (h *AppHeader) SetWorkspaces(workspaces []Workspace, active int) {
	h.workspaces = workspaces
	h.active = active
}

func (h *AppHeader) Render(g *gocui.Gui, dim Dimension) error {
	v, err := g.SetView(h.id, dim.X0-1, dim.Y0-1, dim.X1+1, dim.Y1+1, 0)
	if err != nil && err.Error() != gocui.ErrUnknownView.Error() {
		return err
	}
	h.SetView(v)
	v.Visible = true
	v.Frame = false
	v.Clear()
	fmt.Fprint(v, h.Content())
	return nil
}

func (h *AppHeader) Content() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s    Accessibility audit explorer\n", h.appName)
	var tabs []string
	for i, workspace := range h.workspaces {
		label := fmt.Sprintf("%d %s", i+1, workspace.Name)
		if i == h.active {
			label = "[" + label + "]"
		}
		tabs = append(tabs, label)
	}
	b.WriteString(strings.Join(tabs, "   "))
	return b.String()
}
