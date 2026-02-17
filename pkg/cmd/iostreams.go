package cmd

import (
	"io"
	"os"
)

// IOStreams provides standard IO abstraction for CLI commands.
// This enables testable output by allowing injection of mock streams.
type IOStreams struct {
	// In is the input stream (stdin) for reading user input
	In io.Reader

	// Out is the output stream (stdout) for program output.
	// Use this for structured data, tables, and machine-readable output.
	Out io.Writer

	// ErrOut is the error output stream (stderr) for diagnostics.
	// Use this for status messages, warnings, and human-readable feedback.
	ErrOut io.Writer
}

// StandardIOStreams returns an IOStreams configured with standard OS streams.
// This should be used in production code.
func StandardIOStreams() IOStreams {
	return IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}
