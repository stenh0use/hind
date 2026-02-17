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
	streams := cmd.StandardIOStreams()
	if err := Run(logger, streams, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// Run sets up and executes the CLI root command.
func Run(logger *log.Logger, streams cmd.IOStreams, args []string) error {
	rootCmd := hind.NewCommand(logger, streams)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		return err
	}
	return nil
}
