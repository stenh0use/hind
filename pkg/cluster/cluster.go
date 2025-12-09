// Package cluster provides cluster lifecycle management for HashiCorp services.
// It handles creating, starting, stopping, and deleting multi-node clusters with
// support for networking, service discovery, and scaling operations.
package cluster

import (
	"fmt"
	"time"

	"github.com/stenh0use/hind/pkg/file"
)

const (
	ClusterConfigFile      = "cluster.json"
	ClusterConfigDir       = "cluster"
	ActiveClusterFile      = "active"
	DefaultConfigParentDir = ".config"
	DefaultConfigName      = "hind"
	DefaultProvider        = "dockercli"

	// Container startup timeouts and polling intervals
	DefaultContainerStartTimeout = 30 * time.Second
	DefaultContainerPollInterval = 1 * time.Second
)

// List returns all cluster names found in the cluster configuration directory.
func List() ([]string, error) {
	var clusters []string
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return nil, err
	}
	entries, err := fm.ListDir(ClusterConfigDir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			clusters = append(clusters, e.Name())
		}
	}
	return clusters, nil
}

// GetActiveCluster returns the name of the currently active cluster
// Returns empty string if no active cluster is set
func GetActiveCluster() (string, error) {
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return "", err
	}

	activeFile := file.JoinPath(ClusterConfigDir, ActiveClusterFile)
	if !fm.FileExists(activeFile) {
		return "", nil
	}

	data, err := fm.ReadFile(activeFile)
	if err != nil {
		return "", fmt.Errorf("failed to read active cluster file: %w", err)
	}

	return string(data), nil
}

// SetActiveCluster sets the currently active cluster
func SetActiveCluster(clusterName string) error {
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return err
	}

	// Ensure cluster directory exists
	if err := fm.EnsureDir(ClusterConfigDir); err != nil {
		return fmt.Errorf("failed to ensure cluster directory exists: %w", err)
	}

	// Verify cluster exists
	clusterDir := file.JoinPath(ClusterConfigDir, clusterName)
	if !fm.DirExists(clusterDir) {
		return fmt.Errorf("cluster '%s' does not exist", clusterName)
	}

	activeFile := file.JoinPath(ClusterConfigDir, ActiveClusterFile)
	if err := fm.WriteFile(activeFile, []byte(clusterName)); err != nil {
		return fmt.Errorf("failed to write active cluster file: %w", err)
	}

	return nil
}

// ClearActiveCluster clears the currently active cluster setting
func ClearActiveCluster() error {
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return err
	}

	activeFile := file.JoinPath(ClusterConfigDir, ActiveClusterFile)
	if !fm.FileExists(activeFile) {
		return nil // Already cleared
	}

	if err := fm.RemoveFile(activeFile); err != nil {
		return fmt.Errorf("failed to remove active cluster file: %w", err)
	}

	return nil
}
