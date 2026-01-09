package main

import (
	"fmt"
	"time"
)

var debugEnabled = false

// SetDebug enables or disables debug output
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// DebugLog prints debug message if debug is enabled
func DebugLog(format string, args ...interface{}) {
	if debugEnabled {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[DEBUG %s] %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}
