package set

import (
	"io"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/file"
)

func TestSetProfileCommand(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a test cluster directory
	fm, err := file.NewFromHomeDir(cluster.DefaultConfigParentDir, cluster.DefaultConfigName)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	testClusterName := "test-cluster"
	clusterDir := file.JoinPath(cluster.ClusterConfigDir, testClusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("Failed to create test cluster directory: %v", err)
	}

	// Create logger and streams
	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Create command
	command := NewCommand(logger, streams)

	// Set args
	command.SetArgs([]string{"profile", testClusterName})

	// Execute command
	err = command.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify active cluster was set
	activeCluster, err := cluster.GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster() failed: %v", err)
	}

	if activeCluster != testClusterName {
		t.Errorf("Expected active cluster '%s', got: '%s'", testClusterName, activeCluster)
	}
}

func TestSetProfileCommand_NonExistentCluster(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create logger and streams
	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Create command
	command := NewCommand(logger, streams)

	// Set args to non-existent cluster
	command.SetArgs([]string{"profile", "non-existent-cluster"})

	// Execute command - should fail
	err := command.Execute()
	if err == nil {
		t.Fatal("Expected error when setting non-existent cluster as active, got nil")
	}
}

func TestSetProfileCommand_NoArgs(t *testing.T) {
	// Create logger and streams
	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Create command
	command := NewCommand(logger, streams)

	// Set no args - should fail
	command.SetArgs([]string{"profile"})

	// Execute command - should fail
	err := command.Execute()
	if err == nil {
		t.Fatal("Expected error when no cluster name provided, got nil")
	}
}
