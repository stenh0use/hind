package app

import (
	"os"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind"
)

// Main is the entrypoint for the hind CLI.
func Main() {
	// Get log level from environment variable, defaulting to INFO
	logLevel := cmd.GetLogLevelFromEnv()
	logger := cmd.NewLogger(logLevel, "text")
	if err := Run(logger, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// Run sets up and executes the CLI root command.
func Run(logger *log.Logger, args []string) error {
	cmd := hind.NewCommand(logger)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		return err
	}
	return nil
}
