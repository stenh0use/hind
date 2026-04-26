// Package cluster provides cluster lifecycle management for HashiCorp services.
// It handles creating, starting, stopping, and deleting multi-node clusters with
// support for networking, service discovery, and scaling operations.
package cluster

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ValidateClusterName ensures a cluster name cannot be used for path traversal
// or absolute/root escape when constructing persisted config paths.
func ValidateClusterName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("cluster name cannot be empty")
	}

	if filepath.IsAbs(trimmed) {
		return errors.New("cluster name must be relative")
	}

	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, segment := range segments {
		if segment == ".." {
			return errors.New("cluster name cannot contain traversal segments")
		}
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return errors.New("cluster name cannot resolve to current directory")
	}

	if strings.HasPrefix(cleaned, "..") {
		return errors.New("cluster name cannot escape root")
	}

	return nil
}

// List returns all cluster names found in the cluster configuration directory.
func List() ([]string, error) {
	var clusters []string
	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return nil, err
	}
	entries, err := fm.ListDir(ClusterConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return clusters, nil
		}
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
	if err := ValidateClusterName(clusterName); err != nil {
		return fmt.Errorf("invalid cluster name %q: %w", clusterName, err)
	}

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
