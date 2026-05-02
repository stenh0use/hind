package dockercli

import (
	"context"
	"encoding/json"
	"fmt"
)

// dockerInfo holds the parsed output of `docker system info --format {{json .}}`.
type dockerInfo struct {
	ClientInfo clientInfo `json:"ClientInfo"`
}

type clientInfo struct {
	Plugins []plugin `json:"Plugins"`
	Version string   `json:"Version"`
}

type plugin struct {
	Name string `json:"Name"`
}

// hasClientPlugin reports whether a named client plugin is present in the docker info.
func (i *dockerInfo) hasClientPlugin(name string) bool {
	for _, p := range i.ClientInfo.Plugins {
		if p.Name == name {
			return true
		}
	}
	return false
}

// checkBuildxAvailable checks at call time whether the buildx plugin is installed.
// It uses the provided executor so tests can inject fakes.
func checkBuildxAvailable(ctx context.Context, executor CommandExecutor) error {
	raw, err := executor.Output(ctx, "", "docker", "system", "info", "--format", "{{json .}}")
	if err != nil {
		return fmt.Errorf("failed to get docker system info: %w", err)
	}

	info := dockerInfo{}
	if err := json.Unmarshal(raw, &info); err != nil {
		return fmt.Errorf("failed to parse docker system info: %w", err)
	}

	if !info.hasClientPlugin("buildx") {
		return fmt.Errorf("buildx client plugin is needed but not installed")
	}

	return nil
}
