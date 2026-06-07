package fs

import (
	"fmt"
	"path/filepath"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

const (
	clusterConfigFile = "cluster.json"
	clusterConfigDir  = "cluster"
	activeClusterFile = "active"
)

func clusterConfigPath(root string, name domain.Name) (string, error) {
	validated, err := validateName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, clusterConfigDir, string(validated), clusterConfigFile), nil
}

func clusterDirPath(root string, name domain.Name) (string, error) {
	validated, err := validateName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, clusterConfigDir, string(validated)), nil
}

func activePath(root string) string {
	return filepath.Join(root, clusterConfigDir, activeClusterFile)
}

func validateName(name domain.Name) (domain.Name, error) {
	if err := domain.ValidateName(string(name)); err != nil {
		return "", fmt.Errorf("invalid cluster name %q: %w", name, err)
	}
	return name, nil
}
