package stop

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

	if cmd.Use != "stop [cluster-name]" {
		t.Errorf("Expected Use to be 'stop [cluster-name]', got '%s'", cmd.Use)
	}

	if cmd.Short != "Stop a hind cluster" {
		t.Errorf("Expected Short to be 'Stop a hind cluster', got '%s'", cmd.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 30 * time.Second
	if DefaultStopTimeout != expected {
		t.Errorf("Expected DefaultStopTimeout to be %v, got %v", expected, DefaultStopTimeout)
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

	if timeoutFlag.DefValue != "30s" {
		t.Errorf("Expected timeout default value to be '30s', got '%s'", timeoutFlag.DefValue)
	}
}

func TestCommandArgs(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}

	// Test with valid number of args (0 or 1)
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
			cmd := NewCommand(logger)
			cmd.SetArgs(tt.args)
			err := cmd.Args(cmd, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
