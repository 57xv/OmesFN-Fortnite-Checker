package main

import (
	"fmt"
	"sync"
)

const (
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorCyan   = "\033[36m"
	ColorReset  = "\033[0m"
)

var printLock sync.Mutex

func safePrint(color, prefix, message string) {
	printLock.Lock()
	defer printLock.Unlock()
	fmt.Printf("%s%s %s%s\n", color, prefix, message, ColorReset)
}

func LogSuccess(message string) {
	safePrint(ColorGreen, "[SUCCESS]", message)
}

func LogWarning(message string) {
	// safePrint(ColorYellow, "[WARNING]", message)
}

func LogError(message string) {
	// safePrint(ColorRed, "[ERROR]", message)
}

func Log(level, message string) {
	var color string
	switch level {
	case "SUCCESS":
		color = ColorGreen
	case "WARNING":
		color = ColorYellow
	case "ERROR":
		color = ColorRed
	case "INFO":
		color = ColorCyan
	default:
		color = ColorReset
	}
	safePrint(color, fmt.Sprintf("[%s]", level), message)
}

func LogInfo(message string) {
	safePrint(ColorCyan, "", message)
}
