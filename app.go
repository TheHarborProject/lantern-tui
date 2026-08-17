package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

type App struct {
	g *gocui.Gui

	Config       AppConfig
	IsModalOpen  bool
	currentModal ModalComponent // Currently visible modal (nil if no modal is open)
	previousView string         // View ID before modal was opened (for focus restoration)

	LogFile *os.File

	GlobalKeybindings []Keybinding
	viewKeybindings   map[string][]Keybinding

	// App이 관리하는 모든 UI 컴포넌트 (패널, 탭뷰, 모달 등)
	components      []UIComponent
	currentFocusIdx int

	statusBar *StatusBar
	header    *AppHeader
	focusMode FocusMode

	focusModePanels map[string]FocusModeCapability
	workspaces      []Workspace
	activeWorkspace int

	layoutStrategies         []*boxlayout.Box
	currentLayoutStrategyIdx int

	// Task management
	IsTaskRunning bool               // Flag indicating if a task is currently running
	currentTask   *TuiTask           // Currently running task (nil if no task running)
	taskCancel    context.CancelFunc // Function to cancel current task
	spinner       *Spinner           // Spinner component for task visualization
}

type AppConfig struct {
	DebugMode bool
	AppName   string
	Version   string
	Developer string
}

func NewApp(config AppConfig) (*App, error) {
	g, err := gocui.NewGui(gocui.NewGuiOpts{
		OutputMode: gocui.OutputTrue,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		g:               g,
		Config:          config,
		focusModePanels: make(map[string]FocusModeCapability),
	}
	app.init()

	return app, nil
}

func (a *App) Run() (runErr error) {
	defer closeLogger()
	defer func() {
		if r := recover(); r != nil {
			logDebugf("PANIC RECOVERED: %v", r)
			runErr = fmt.Errorf("TUI panic: %v\n%s", r, debug.Stack())
		}
	}()

	// Ensure terminal is properly restored on exit
	defer a.g.Close()

	if err := a.g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func (a *App) init() {
	// Initialize logger
	if a.Config.DebugMode {
		logDir := "logs"
		os.MkdirAll(logDir, 0755)
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		logPath := filepath.Join(logDir, fmt.Sprintf("lazytui_%s.log", timestamp))

		if err := initLogger(true, logPath); err != nil {
			panic(fmt.Errorf("failed to initialise logger: %w", err))
		}
		logDebugf("App config: %+v", a.Config)
	}

	logDebug("Initialising app...")

	a.currentLayoutStrategyIdx = 0

	// Initialize StatusBar
	a.statusBar = NewStatusBar("statusbar", a.Config)
	a.header = NewAppHeader("appheader", a.Config.AppName)

	// Set layout manager
	a.g.SetManagerFunc(gocui.ManagerFunc(a.layoutManager))

	// Enable mouse support
	a.g.Mouse = true

	// Show footer for all views (including modals)
	a.g.ShowListFooter = true

	// Setup default keybindings
	if err := a.setupDefaultKeybindings(); err != nil {
		logDebugf("ERROR: Failed to setup keybindings: %v", err)
		panic(err)
	}

	logDebug("App initialisation complete")
}

// ModalComponent is a helper interface to access BaseModal from embedded types
type ModalComponent interface {
	UIComponent
	GetBaseModal() *BaseModal
	Show(message string)
	Hide()
}

// RegisterModal registers a modal component (special handling for modals)
// Accepts any modal type that embeds BaseModal (BaseModal, OkOnlyModal, etc.)
func (a *App) RegisterModal(mc ModalComponent) {
	m := mc.GetBaseModal()

	a.components = append(a.components, mc)
	logDebugf("Modal registered: %s", mc.ID())

	// Pre-create the view with dummy dimensions
	v, err := a.g.SetView(mc.ID(), 0, 0, 10, 5, 0)
	if err != nil {
		if err.Error() != "unknown view" {
			logDebugf("ERROR: Failed to pre-create modal view %s: %v", mc.ID(), err)
			return
		}
		// ErrUnknownView is expected for first-time view creation
		// Get the view again after creation
		v, _ = a.g.View(mc.ID())
	}

	logDebugf("Modal view pre-created: %s", mc.ID())
	// Modals are ALWAYS initially hidden
	if v != nil {
		v.Visible = false
		logDebugf("Modal %s set to invisible", mc.ID())
	}

	// Set BaseView reference
	m.SetView(v)

	// Register modal-specific bindings (ESC key, and child-specific bindings)
	if err := mc.RegisterBindings(a.g, a); err != nil {
		logDebugf("Failed to register modal bindings for %s: %v", mc.ID(), err)
	}

	// NOTE: Do NOT register mouse click handlers for modals
	// Modals handle their own focus in layoutManager
}

// RegisterComponent는 App이 관리할 UIComponent를 등록한다.
func (a *App) RegisterComponent(c UIComponent) {
	a.components = append(a.components, c)
	logDebugf("Component registered: %s (visible: %v)", c.ID(), c.IsVisible())

	// Pre-create the view with dummy dimensions
	// This ensures the view exists in gocui before first render
	_, err := a.g.SetView(c.ID(), 0, 0, 10, 5, 0)
	if err != nil && err.Error() != "unknown view" {
		logDebugf("WARNING: Failed to pre-create view %s: %v", c.ID(), err)
	} else {
		logDebugf("View pre-created: %s", c.ID())
	}

	// Register mouse click handler for focusable components
	// Skip ListPanel and TabPanel as they handle their own click events
	if _, ok := c.(Focusable); ok {
		_, isListPanel := c.(*ListPanel)
		_, isTabPanel := c.(*TabPanel)
		if !isListPanel && !isTabPanel {
			a.registerMouseClickForFocus(c.ID())
		}
	}

	// Register component-specific bindings (e.g., mouse wheel for scrollable)
	if err := c.RegisterBindings(a.g, a); err != nil {
		logDebugf("Failed to register bindings for %s: %v", c.ID(), err)
	}
}

// RegisterTabPanel은 TabPanel을 등록하고 초기 child의 keybindings를 StatusBar에 등록한다.
func (a *App) RegisterTabPanel(tp *TabPanel) {
	// Store app reference in TabPanel for dynamic keybinding updates
	tp.app = a

	// Register as normal component
	a.RegisterComponent(tp)

	// Register initial active child's keybindings to StatusBar (using child ID)
	if len(tp.tabs) > 0 {
		initialChild := tp.tabs[0].Component
		if a.viewKeybindings == nil {
			a.viewKeybindings = make(map[string][]Keybinding)
		}
		a.viewKeybindings[initialChild.ID()] = tp.tabs[0].Keybindings
		logDebugf("RegisterTabPanel: Registered initial child %s keybindings to StatusBar", initialChild.ID())
	}
}

// registerMouseClickForFocus registers a mouse click handler to switch focus
func (a *App) registerMouseClickForFocus(viewID string) {
	a.g.SetViewClickBinding(&gocui.ViewMouseBinding{
		ViewName: viewID,
		Key:      gocui.MouseLeft,
		Modifier: gocui.ModNone,
		Handler: func(opts gocui.ViewMouseBindingOpts) error {
			return a.handlePanelClick(viewID)
		},
	})
	logDebugf("Mouse click binding registered for: %s", viewID)
}

// handlePanelClick handles mouse click on a panel to switch focus
func (a *App) handlePanelClick(viewID string) error {
	// Ignore if modal is open
	if a.IsModalOpen {
		logDebug("handlePanelClick: modal is open, ignoring")
		return nil
	}

	logDebugf("handlePanelClick: %s", viewID)

	focusables := a.getFocusableComponents()

	// Find the index of the clicked component
	for i, f := range focusables {
		if f.ID() == viewID {
			// Already focused, do nothing
			if i == a.currentFocusIdx {
				logDebugf("handlePanelClick: %s already focused", viewID)
				return nil
			}

			// Blur current focus
			if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(focusables) {
				focusables[a.currentFocusIdx].OnBlur(a.g)
			}

			// Update focus
			a.currentFocusIdx = i
			focusables[i].OnFocus(a.g)
			a.g.SetCurrentView(viewID)

			logDebugf("Focus switched to: %s (index: %d)", viewID, i)
			return nil
		}
	}

	logDebugf("handlePanelClick: component %s not found in focusables", viewID)
	return nil
}

// getFocusableComponents returns only the focusable components.
func (a *App) getFocusableComponents() []Focusable {
	var focusables []Focusable
	for _, c := range a.components {
		if f, ok := c.(Focusable); ok && c.IsVisible() && a.componentInActiveWorkspace(c.ID()) {
			focusables = append(focusables, f)
		}
	}
	return focusables
}

// FocusNext는 다음 컴포넌트로 포커스를 옮긴다.
// 모달이 열린 경우에는 포커스를 이동시키지 않는다.
func (a *App) FocusNext() {
	if a.IsModalOpen || a.focusMode.Active {
		return
	}

	focusables := a.getFocusableComponents()
	if len(focusables) == 0 {
		return
	}

	// 이전 포커스 해제
	if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(focusables) {
		focusables[a.currentFocusIdx].OnBlur(a.g)
	}

	// 인덱스 갱신
	a.currentFocusIdx = (a.currentFocusIdx + 1) % len(focusables)

	// 새 포커스 적용
	focusables[a.currentFocusIdx].OnFocus(a.g)
	if a.g != nil {
		a.g.SetCurrentView(focusables[a.currentFocusIdx].ID())
	}
}

// FocusPrev는 이전 컴포넌트로 포커스를 옮긴다.
func (a *App) FocusPrev() {
	if a.IsModalOpen || a.focusMode.Active {
		return
	}

	focusables := a.getFocusableComponents()
	if len(focusables) == 0 {
		return
	}

	// 이전 포커스 해제
	if a.currentFocusIdx >= 0 && a.currentFocusIdx < len(focusables) {
		focusables[a.currentFocusIdx].OnBlur(a.g)
	}

	// 인덱스 갱신
	a.currentFocusIdx--
	if a.currentFocusIdx < 0 {
		a.currentFocusIdx = len(focusables) - 1
	}

	// 새 포커스 적용
	focusables[a.currentFocusIdx].OnFocus(a.g)
	if a.g != nil {
		a.g.SetCurrentView(focusables[a.currentFocusIdx].ID())
	}
}

// Refresh triggers a UI update from goroutines (thread-safe)
func (a *App) Refresh(updateFn func(*gocui.Gui) error) {
	a.g.Update(updateFn)
}

// ============================================================================
// Task Management
// ============================================================================

// RunTask executes a TuiTask in the background
// Returns error if a task is already running
// Displays spinner overlay on StatusBar while task is running
func (a *App) RunTask(task *TuiTask) error {
	logDebugf("RunTask: Called with IsTaskRunning=%v, spinner=%v", a.IsTaskRunning, a.spinner)
	if a.IsTaskRunning {
		logDebugf("RunTask: Task already running, returning error")
		return fmt.Errorf("task already running")
	}

	logDebugf("RunTask: Starting task with message='%s'", task.GetProgressMsg())

	a.IsTaskRunning = true
	a.currentTask = task

	// Start spinner animation
	if a.spinner != nil {
		logDebugf("RunTask: Calling spinner.Start()")
		a.spinner.Start(a, task.GetProgressMsg())
		logDebugf("RunTask: spinner.Start() returned")
	} else {
		logDebugf("RunTask: No spinner available!")
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	a.taskCancel = cancel

	// Run task in background
	go func() {
		logDebugf("RunTask: Executing task function")
		err := task.Execute(ctx)

		// Stop spinner
		if a.spinner != nil {
			logDebugf("RunTask: Calling spinner.Stop()")
			a.spinner.Stop()
			logDebugf("RunTask: spinner.Stop() returned")
		}

		logDebugf("RunTask: Task completed, err=%v", err)

		// Execute completion callback (if set)
		if task.onComplete != nil {
			a.Refresh(func(g *gocui.Gui) error {
				task.onComplete(err)
				return nil
			})
		}

		// Reset task state
		a.IsTaskRunning = false
		a.currentTask = nil
		a.taskCancel = nil

		logDebugf("RunTask: Task state reset")
	}()

	return nil
}

// StopTask cancels the currently running task (if any)
func (a *App) StopTask() {
	if a.taskCancel != nil {
		logDebugf("StopTask: Cancelling task")
		a.taskCancel()
	}
}
