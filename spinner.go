package main

import (
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jesseduffield/gocui"
)

// ============================================================================
// Spinner - Overlay on StatusBar during task execution
// ============================================================================

// SpinnerStyle defines the animation style for the spinner
type SpinnerStyle int

const (
	SpinnerStyleDefault SpinnerStyle = iota // |, /, -, \
	SpinnerStyleDots                        // Braille pattern dots
)

// Spinner displays an animated spinner on top of StatusBar during task execution
type Spinner struct {
	*BaseView
	frames   []rune    // Animation frames
	frameIdx int       // Current frame index
	message  string    // Message to display (e.g., "Loading")
	ticker   *time.Ticker
	stopChan chan bool
	mu       sync.Mutex // Protects stopChan access
	config   AppConfig  // For right section formatting (same as StatusBar)
}

// NewSpinner creates a new spinner with the specified style
func NewSpinner(id string, style SpinnerStyle, config AppConfig) *Spinner {
	var frames []rune
	switch style {
	case SpinnerStyleDefault:
		frames = []rune{'|', '/', '-', '\\'}
	case SpinnerStyleDots:
		frames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	default:
		frames = []rune{'|', '/', '-', '\\'}
	}

	return &Spinner{
		BaseView: NewBaseView(id, false), // initially hidden
		frames:   frames,
		frameIdx: 0,
		message:  "",
		config:   config,
	}
}

// Start begins the spinner animation
// Runs in a goroutine and updates the UI periodically
func (s *Spinner) Start(app *App, message string) {
	logDebugf("Spinner.Start() called: message=%s, visible=%v, ticker=%v, stopChan=%v", message, s.visible, s.ticker, s.stopChan)

	s.mu.Lock()
	s.message = message
	s.frameIdx = 0
	s.visible = true
	s.ticker = time.NewTicker(100 * time.Millisecond)
	s.stopChan = make(chan bool)
	stopChan := s.stopChan // Store local copy for goroutine
	s.mu.Unlock()

	logDebugf("Spinner.Start() initialized: visible=%v, ticker created, stopChan created", s.visible)

	go func() {
		logDebugf("Spinner goroutine started")
		for {
			select {
			case <-s.ticker.C:
				// Advance to next frame
				s.frameIdx = (s.frameIdx + 1) % len(s.frames)
				// Trigger UI update
				app.Refresh(func(g *gocui.Gui) error {
					return nil // layoutManager will call Render
				})
			case <-stopChan:
				logDebugf("Spinner received stop signal")
				s.ticker.Stop()
				s.visible = false
				// Final refresh to restore StatusBar and hide spinner view
				app.Refresh(func(g *gocui.Gui) error {
					if v, err := g.View(s.id); err == nil {
						v.Visible = false
						logDebugf("Spinner view hidden: %s", s.id)
					}
					return nil
				})
				logDebugf("Spinner goroutine exiting")
				return
			}
		}
	}()

	logDebugf("Spinner started: message=%s", message)
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	logDebugf("Spinner.Stop() called: stopChan=%v", s.stopChan)
	if s.stopChan != nil {
		s.stopChan <- true
		close(s.stopChan) // Close channel to prevent reuse
		s.stopChan = nil  // Reset to nil
		logDebugf("Spinner stopped: signal sent, channel closed")
	} else {
		logDebugf("Spinner.Stop() called but stopChan is nil!")
	}
}

// Render implements UIComponent interface
// Renders spinner overlay on StatusBar position
func (s *Spinner) Render(g *gocui.Gui, dim Dimension) error {
	if !s.visible {
		// Hide view if not visible
		if v, err := g.View(s.id); err == nil {
			v.Visible = false
		}
		return nil
	}

	logDebugf("Spinner.Render() called: visible=%v, frameIdx=%d, message=%s", s.visible, s.frameIdx, s.message)

	// Use same frame offset as StatusBar (no frame)
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
	v.Visible = true // CRITICAL: Show the view
	v.Clear()

	// Left content: spinner frame + message
	currentFrame := s.frames[s.frameIdx]
	leftContent := fmt.Sprintf("%c %s...", currentFrame, s.message)

	// Right content: same format as StatusBar (app info)
	rightContent := s.formatRightSection()

	// Calculate padding (same as StatusBar)
	width := dim.X1 - dim.X0 + 1
	leftLen := utf8.RuneCountInString(leftContent)
	rightLen := utf8.RuneCountInString(rightContent)

	if leftLen+rightLen < width {
		padding := width - leftLen - rightLen
		fmt.Fprintf(v, "%s%*s%s", leftContent, padding, "", rightContent)
	} else {
		// Not enough space, prioritize right section
		fmt.Fprintf(v, "%s", rightContent)
	}

	return nil
}

// formatRightSection builds the right section string (same as StatusBar)
// Format: "AppName vVersion | by Developer [DEBUG]"
func (s *Spinner) formatRightSection() string {
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

// RegisterBindings implements UIComponent interface (no bindings for Spinner)
func (s *Spinner) RegisterBindings(g *gocui.Gui, app *App) error {
	return nil
}
