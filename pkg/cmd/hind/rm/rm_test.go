package rm

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/file"
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

	if command.Use != "rm [cluster-name]" {
		t.Errorf("Expected Use to be 'rm [cluster-name]', got '%s'", command.Use)
	}

	if command.Short != "Remove a hind cluster" {
		t.Errorf("Expected Short to be 'Remove a hind cluster', got '%s'", command.Short)
	}
}

func TestDefaultTimeout(t *testing.T) {
	expected := 2 * time.Minute
	if DefaultDeleteTimeout != expected {
		t.Errorf("Expected DefaultDeleteTimeout to be %v, got %v", expected, DefaultDeleteTimeout)
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

	// Check if timeout flag exists
	timeoutFlag := command.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("Expected 'timeout' flag to exist")
	}

	if timeoutFlag.DefValue != "2m0s" {
		t.Errorf("Expected timeout default value to be '2m0s', got '%s'", timeoutFlag.DefValue)
	}
}

// stubDeleter is a no-op clusterDeleter used to bypass real Docker calls in tests.
type stubDeleter struct{}

func (s *stubDeleter) Delete(_ context.Context) error { return nil }

// TestRunE_ClearsActiveClusterOnDelete verifies that when the cluster being removed
// is the currently active cluster, runE calls ClearActiveCluster so that subsequent
// commands fall back to the "default" cluster resolution path.
func TestRunE_ClearsActiveClusterOnDelete(t *testing.T) {
	// Redirect HOME so cluster state is isolated to this test.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	const clusterName = "my-cluster"

	// Pre-create the cluster directory so SetActiveCluster accepts the name.
	fm, err := file.NewFromHomeDir(cluster.DefaultConfigParentDir, cluster.DefaultConfigName)
	if err != nil {
		t.Fatalf("NewFromHomeDir: %v", err)
	}
	clusterDir := file.JoinPath(cluster.ClusterConfigDir, clusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	// Set the cluster as active.
	if err := cluster.SetActiveCluster(clusterName); err != nil {
		t.Fatalf("SetActiveCluster: %v", err)
	}

	// Replace the factory with a stub so Delete() succeeds without Docker.
	orig := newClusterManagerFn
	newClusterManagerFn = func(_ *log.Logger, _ string) (clusterDeleter, error) {
		return &stubDeleter{}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	if err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, clusterName); err != nil {
		t.Fatalf("runE returned unexpected error: %v", err)
	}

	// After deletion of the active cluster the active cluster file must be cleared,
	// yielding an empty string from GetActiveCluster (the canonical "no active cluster" state).
	active, err := cluster.GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster: %v", err)
	}
	if active != "" {
		t.Errorf("expected active cluster to be cleared (empty string), got %q", active)
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
			command := NewCommand(logger, streams)
			command.SetArgs(tt.args)
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
