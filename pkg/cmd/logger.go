package cmd

import (
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/text"
)

// getLogLevelFromEnv reads log level from HIND_LOGLEVEL environment variable
func GetLogLevelFromEnv() log.Level {
	level := os.Getenv("HIND_LOGLEVEL")
	switch strings.ToLower(level) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// newLogger creates a new logger with the specified level and handler type
func NewLogger(level log.Level, handler string) *log.Logger {
	var logHandler log.Handler
	if handler == "text" {
		logHandler = text.Default
	} else {
		logHandler = cli.Default
	}
	return &log.Logger{
		Handler: logHandler,
		Level:   level,
	}
}
