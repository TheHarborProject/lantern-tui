package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	debugLog     *os.File
	debugEnabled bool
	logMutex     sync.Mutex
)

// initLogger initialises the logger
func initLogger(enabled bool, logFilePath string) error {
	debugEnabled = enabled

	if !debugEnabled {
		return nil
	}

	var err error
	debugLog, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logDebug("=== LazyTui Debug Log Started ===")
	logDebug(fmt.Sprintf("Timestamp: %s", time.Now().Format(time.RFC3339)))

	return nil
}

// closeLogger closes the log file
func closeLogger() {
	if debugLog != nil {
		logDebug("=== LazyTui Debug Log Ended ===")
		debugLog.Sync()
		debugLog.Close()
		debugLog = nil
	}
}

// logDebug writes a debug message to the log file
func logDebug(msg string) {
	if !debugEnabled || debugLog == nil {
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(debugLog, "[%s] %s\n", timestamp, msg)
	debugLog.Sync() // Flush to disk immediately
}

// logDebugf writes a formatted debug message to the log file
func logDebugf(format string, args ...any) {
	logDebug(fmt.Sprintf(format, args...))
}
