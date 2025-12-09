package build

import (
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
)

func TestNewCommand(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}

	cmd := NewCommand(logger)

	if cmd == nil {
		t.Fatal("NewCommand() returned nil")
	}

	// Verify Use contains expected text
	if cmd.Use == "" {
		t.Error("Expected Use to be non-empty")
	}

	if cmd.Short != "Build container images" {
		t.Errorf("Expected Short to be 'Build container images', got '%s'", cmd.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 15 * time.Minute
	if DefaultBuildTimeout != expected {
		t.Errorf("Expected DefaultBuildTimeout to be %v, got %v", expected, DefaultBuildTimeout)
	}
}

func TestCommandFlags(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}

	cmd := NewCommand(logger)

	// Check if timeout flag exists
	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}

	if timeoutFlag.DefValue != "15m0s" {
		t.Errorf("Expected timeout default value to be '15m0s', got '%s'", timeoutFlag.DefValue)
	}
}

func TestCommandArgs(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}

	// Test with valid number of args (exactly 1)
	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no args",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "one valid arg",
			args:      []string{"all"},
			wantError: false,
		},
		{
			name:      "too many args",
			args:      []string{"nomad", "consul"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCommand(logger)
			cmd.SetArgs(tt.args)
			err := cmd.Args(cmd, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
