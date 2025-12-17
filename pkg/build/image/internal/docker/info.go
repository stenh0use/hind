package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type DockerInfo struct {
	ClientInfo   ClientInfo  `json:"ClientInfo"`
	DriverStatus [][2]string `json:"DriverStatus"`
}

type ClientInfo struct {
	Plugins []Plugin `json:"Plugins"`
	Version string   `json:"Version"`
}

type Plugin struct {
	SchemaVersion    string `json:"SchemaVersion"`
	Vendor           string `json:"Vendor"`
	Version          string `json:"Version"`
	ShortDescription string `json:"ShortDescription"`
	Name             string `json:"Name"`
	Path             string `json:"Path"`
}

func (i *DockerInfo) Get(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "system", "info", "-f", "json")
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get docker info: %w", err)
	}

	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("failed to unmarshal docker info: %w", err)
	}

	return nil
}

func (i *DockerInfo) HasClientPlugin(name string) bool {
	for _, plugin := range i.ClientInfo.Plugins {
		if plugin.Name == name {
			return true
		}
	}
	return false
}

func (i *DockerInfo) HasDriverType(name string) bool {
	for _, ds := range i.DriverStatus {
		if ds[0] == "driver-type" && ds[1] == name {
			return true
		}
	}
	return false
}
