package cluster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stenh0use/hind/pkg/file"
)

func TestGetActiveCluster_NoActiveCluster(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	activeCluster, err := GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster() failed: %v", err)
	}

	if activeCluster != "" {
		t.Errorf("Expected no active cluster, got: %s", activeCluster)
	}
}

func TestSetActiveCluster_Success(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a test cluster directory
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	testClusterName := "test-cluster"
	clusterDir := file.JoinPath(ClusterConfigDir, testClusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("Failed to create test cluster directory: %v", err)
	}

	// Set active cluster
	err = SetActiveCluster(testClusterName)
	if err != nil {
		t.Fatalf("SetActiveCluster() failed: %v", err)
	}

	// Verify active cluster was set
	activeCluster, err := GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster() failed: %v", err)
	}

	if activeCluster != testClusterName {
		t.Errorf("Expected active cluster '%s', got: '%s'", testClusterName, activeCluster)
	}
}

func TestSetActiveCluster_NonExistentCluster(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Try to set a non-existent cluster as active
	err := SetActiveCluster("non-existent-cluster")
	if err == nil {
		t.Fatal("Expected error when setting non-existent cluster as active, got nil")
	}
}

func TestClearActiveCluster_Success(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a test cluster directory
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	testClusterName := "test-cluster"
	clusterDir := file.JoinPath(ClusterConfigDir, testClusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("Failed to create test cluster directory: %v", err)
	}

	// Set active cluster
	if err := SetActiveCluster(testClusterName); err != nil {
		t.Fatalf("SetActiveCluster() failed: %v", err)
	}

	// Clear active cluster
	if err := ClearActiveCluster(); err != nil {
		t.Fatalf("ClearActiveCluster() failed: %v", err)
	}

	// Verify active cluster was cleared
	activeCluster, err := GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster() failed: %v", err)
	}

	if activeCluster != "" {
		t.Errorf("Expected no active cluster after clearing, got: %s", activeCluster)
	}
}

func TestClearActiveCluster_NoActiveCluster(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Clear when no active cluster is set (should not error)
	if err := ClearActiveCluster(); err != nil {
		t.Fatalf("ClearActiveCluster() failed when no active cluster exists: %v", err)
	}
}

func TestActiveClusterPersistence(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a test cluster directory
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	testClusterName := "persistent-cluster"
	clusterDir := file.JoinPath(ClusterConfigDir, testClusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("Failed to create test cluster directory: %v", err)
	}

	// Set active cluster
	if err := SetActiveCluster(testClusterName); err != nil {
		t.Fatalf("SetActiveCluster() failed: %v", err)
	}

	// Verify file was written
	activeFile := file.JoinPath(ClusterConfigDir, ActiveClusterFile)
	if !fm.FileExists(activeFile) {
		t.Fatal("Active cluster file was not created")
	}

	// Read file directly to verify contents
	data, err := fm.ReadFile(activeFile)
	if err != nil {
		t.Fatalf("Failed to read active cluster file: %v", err)
	}

	if string(data) != testClusterName {
		t.Errorf("Active cluster file contents incorrect. Expected '%s', got: '%s'", testClusterName, string(data))
	}

	// Verify we can read it back
	activeCluster, err := GetActiveCluster()
	if err != nil {
		t.Fatalf("GetActiveCluster() failed: %v", err)
	}

	if activeCluster != testClusterName {
		t.Errorf("Expected active cluster '%s', got: '%s'", testClusterName, activeCluster)
	}
}

func TestActiveClusterFilePath(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a test cluster directory
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}

	testClusterName := "test-cluster"
	clusterDir := file.JoinPath(ClusterConfigDir, testClusterName)
	if err := fm.EnsureDir(clusterDir); err != nil {
		t.Fatalf("Failed to create test cluster directory: %v", err)
	}

	// Set active cluster
	if err := SetActiveCluster(testClusterName); err != nil {
		t.Fatalf("SetActiveCluster() failed: %v", err)
	}

	// Verify the active cluster file is in the correct location
	expectedPath := filepath.Join(tmpDir, DefaultConfigParentDir, DefaultConfigName, ClusterConfigDir, ActiveClusterFile)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Active cluster file not found at expected path: %s", expectedPath)
	}
}
