package start

import (
	"io"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cmd"
)

func TestNewCommand(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	if command == nil {
		t.Fatal("NewCommand() returned nil")
	}

	if command.Use != "start [cluster-name]" {
		t.Errorf("Expected Use to be 'start [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Start or create a hind cluster" {
		t.Errorf("Expected Short to be 'Start or create a hind cluster', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 5 * time.Minute
	if DefaultStartTimeout != expected {
		t.Errorf("Expected DefaultStartTimeout to be %v, got %v", expected, DefaultStartTimeout)
	}
}

func TestCommandFlags(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}

	if timeoutFlag.DefValue != "5m0s" {
		t.Errorf("Expected timeout default value to be '5m0s', got '%s'", timeoutFlag.DefValue)
	}

	clientsFlag := command.Flags().Lookup("clients")
	if clientsFlag == nil {
		t.Fatal("Expected 'clients' flag to exist")
	}

	if clientsFlag.DefValue != "1" {
		t.Errorf("Expected clients default value to be '1', got '%s'", clientsFlag.DefValue)
	}

	verboseFlag := command.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("Expected 'verbose' flag to exist")
	}

	if command.Flags().Lookup("version") != nil {
		t.Fatal("Expected 'version' flag to be absent")
	}
}

func TestCommandArgs(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no args",
			args:      []string{},
			wantError: false,
		},
		{
			name:      "one arg",
			args:      []string{"test-cluster"},
			wantError: false,
		},
		{
			name:      "too many args",
			args:      []string{"cluster1", "cluster2"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCommand(logger, streams)
			command.SetArgs(tt.args)
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
