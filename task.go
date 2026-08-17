package main

import "context"

// ============================================================================
// TuiTask - Background task with spinner support
// ============================================================================

// TuiTask represents a background task that runs in a goroutine.
// While running, a spinner is displayed on the StatusBar overlay.
// Only one TuiTask can run at a time.
type TuiTask struct {
	progressMsg string                           // Message to display in spinner (e.g., "Loading", "Processing")
	taskFunc    func(ctx context.Context) error  // Function to execute in background
	onComplete  func(error)                      // Callback when task completes (optional)
}

// NewTuiTask creates a new background task
// progressMsg: Message to display in spinner (e.g., "Loading", "Saving")
// taskFunc: Function to execute in background (receives context for cancellation)
func NewTuiTask(progressMsg string, taskFunc func(context.Context) error) *TuiTask {
	return &TuiTask{
		progressMsg: progressMsg,
		taskFunc:    taskFunc,
	}
}

// SetOnComplete registers a callback to execute when task completes
// The callback receives an error if the task failed, or nil if successful
func (t *TuiTask) SetOnComplete(fn func(error)) {
	t.onComplete = fn
}

// GetProgressMsg returns the progress message for spinner display
func (t *TuiTask) GetProgressMsg() string {
	return t.progressMsg
}

// Execute runs the task function with the given context
func (t *TuiTask) Execute(ctx context.Context) error {
	return t.taskFunc(ctx)
}
