package main

import (
	"fmt"
	"unicode/utf8"

	"github.com/jesseduffield/gocui"
)

// ============================================================================
// StatusBar - Non-focusable panel at the bottom
// ============================================================================

type StatusBar struct {
	*BaseView
	leftContent string
	config      AppConfig // For right section formatting
}

func NewStatusBar(id string, config AppConfig) *StatusBar {
	return &StatusBar{
		BaseView:    NewBaseView(id, true),
		leftContent: "",
		config:      config,
	}
}

// Render implements UIComponent interface.
func (s *StatusBar) Render(g *gocui.Gui, dim Dimension) error {
	if !s.visible {
		if v, err := g.View(s.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	// StatusBar has no frame, so adjust dimensions
	frameOffset := 1
	x0 := dim.X0 - frameOffset
	y0 := dim.Y0 - frameOffset
	x1 := dim.X1 + frameOffset
	y1 := dim.Y1 + frameOffset
	
	v, err := g.SetView(s.id, x0, y0, x1, y1, 0)
	if err != nil && err.Error() != "unknown view" {
		return err
	}
	s.SetView(v)

	v.Frame = false

	v.Clear()

	// Format right section based on AppConfig
	rightContent := s.formatRightSection()

	// Use actual screen width from dim, not v.Size()
	// v.Size() includes frameOffset which goes beyond screen bounds
	width := dim.X1 - dim.X0 + 1
	
	// Use rune count instead of byte length for proper UTF-8 handling
	leftLen := utf8.RuneCountInString(s.leftContent)
	rightLen := utf8.RuneCountInString(rightContent)

	// Format: left content + padding + right content
	if leftLen+rightLen < width {
		padding := width - leftLen - rightLen
		fmt.Fprintf(v, "%s%*s%s", s.leftContent, padding, "", rightContent)
	} else {
		// Not enough space, prioritise right section
		fmt.Fprintf(v, "%s", rightContent)
	}

	return nil
}

// SetLeftContent updates the left section (keybindings).
func (s *StatusBar) SetLeftContent(content string) {
	s.leftContent = content
}

// formatRightSection builds the right section string from AppConfig.
// Format: "AppName vVersion | by Developer [DEBUG]"
// All fields are nullable.
func (s *StatusBar) formatRightSection() string {
	var parts []string

	// AppName + Version
	if s.config.AppName != "" {
		if s.config.Version != "" {
			parts = append(parts, fmt.Sprintf("%s v%s", s.config.AppName, s.config.Version))
		} else {
			parts = append(parts, s.config.AppName)
		}
	} else if s.config.Version != "" {
		parts = append(parts, fmt.Sprintf("v%s", s.config.Version))
	}

	// Developer
	if s.config.Developer != "" {
		parts = append(parts, fmt.Sprintf("by %s", s.config.Developer))
	}

	// Debug mode indicator
	if s.config.DebugMode {
		parts = append(parts, "[DEBUG]")
	}

	// Join with " | "
	if len(parts) == 0 {
		return ""
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " | "
		}
		result += part
	}

	return result
}
