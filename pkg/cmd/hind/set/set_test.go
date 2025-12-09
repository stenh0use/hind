package set

import (
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/cluster"
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

	// Create logger
	logger := &log.Logger{Handler: discard.New()}

	// Create command
	cmd := NewCommand(logger)

	// Set args
	cmd.SetArgs([]string{"profile", testClusterName})

	// Execute command
	err = cmd.Execute()
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

	// Create logger
	logger := &log.Logger{Handler: discard.New()}

	// Create command
	cmd := NewCommand(logger)

	// Set args to non-existent cluster
	cmd.SetArgs([]string{"profile", "non-existent-cluster"})

	// Execute command - should fail
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when setting non-existent cluster as active, got nil")
	}
}

func TestSetProfileCommand_NoArgs(t *testing.T) {
	// Create logger
	logger := &log.Logger{Handler: discard.New()}

	// Create command
	cmd := NewCommand(logger)

	// Set no args - should fail
	cmd.SetArgs([]string{"profile"})

	// Execute command - should fail
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when no cluster name provided, got nil")
	}
}
