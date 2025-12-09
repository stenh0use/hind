package dockercli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/moby/moby/api/types/container"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

const containerCmd = "container"

func baseContainerCmd(ctx context.Context) *exec.Cmd {
	return baseClientCmd(ctx, containerCmd)
}

// Create and start a container
func (c *Client) CreateContainer(ctx context.Context, cfg config.Node) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("name is required to create a container")
	}

	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "run")

	// TODO: this needs to be moved to an opts abstraction
	// Add standard Docker flags
	cmd.Args = append(cmd.Args, "--cgroupns=private")
	cmd.Args = append(cmd.Args, "--detach")
	cmd.Args = append(cmd.Args, "--init=false")
	cmd.Args = append(cmd.Args, "--privileged")
	cmd.Args = append(cmd.Args, "--restart", "on-failure:1")
	cmd.Args = append(cmd.Args, "--tmpfs", "/run")
	cmd.Args = append(cmd.Args, "--tmpfs", "/tmp")
	cmd.Args = append(cmd.Args, "--tty")
	cmd.Args = append(cmd.Args, "--security-opt", "seccomp=unconfined")
	cmd.Args = append(cmd.Args, "--security-opt", "apparmor=unconfined")
	cmd.Args = append(cmd.Args, "--volume", "/lib/modules:/lib/modules:ro")
	cmd.Args = append(cmd.Args, "--volume", "/var")

	// Add network if specified
	if cfg.Network != "" {
		cmd.Args = append(cmd.Args, "--network", cfg.Network)
	}

	if cfg.Image.Name == "" {
		return "", fmt.Errorf("image name is required")
	}

	var imgRef string
	if cfg.Image.Digest != "" {
		imgRef = fmt.Sprintf("%s:%s", cfg.Image.Name, cfg.Image.Digest)
	} else if cfg.Image.Tag != "" {
		imgRef = fmt.Sprintf("%s:%s", cfg.Image.Name, cfg.Image.Tag)
	} else {
		imgRef = cfg.Name
	}
	// add container name
	if cfg.Name != "" {
		cmd.Args = append(cmd.Args, "--name", cfg.Name)
		cmd.Args = append(cmd.Args, "--hostname", cfg.Name)
	}
	// TODO add volumes
	if cfg.Ports != nil {
		for _, p := range cfg.Ports {
			var publishPorts []string
			// need to do error checking required values
			if p.ListenAddress != "" {
				publishPorts = append(publishPorts, p.ListenAddress)
			}
			if p.HostPort != 0 {
				publishPorts = append(publishPorts, strconv.FormatInt(int64(p.HostPort), 10))
			}
			if p.ContainerPort == 0 {
				return "", fmt.Errorf("container port is required")
			}
			if p.ContainerPort != 0 {
				publishPorts = append(publishPorts, strconv.FormatInt(int64(p.ContainerPort), 10))
			}

			c.logger.WithField("publishPorts", publishPorts).Debug("published ports")

			publishStr := strings.Join(publishPorts, ":")
			if p.Protocol != "" {
				publishStr = fmt.Sprintf("%s/%s", publishStr, p.Protocol)
			}

			cmd.Args = append(cmd.Args, "--publish", publishStr)
		}
	}
	if cfg.Environment != nil {
		for k, v := range cfg.Environment {
			cmd.Args = append(cmd.Args, "--env", fmt.Sprintf("%s=%s", k, v))
		}
	}
	if cfg.Labels != nil {
		for k, v := range cfg.Labels {
			cmd.Args = append(cmd.Args, "--label", fmt.Sprintf("%s=%s", k, v))
		}
	}
	if cfg.Devices != nil {
		for _, d := range cfg.Devices {
			cmd.Args = append(cmd.Args, "--device", d)
		}
	}

	cmd.Args = append(cmd.Args, imgRef)

	c.logger.WithField("command", cmd.String()).Debug("Running container create command")

	id, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return string(id), nil
}

// Start a container if it is stopped
func (c *Client) StartContainer(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name is required to start a container")
	}

	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "start", name)

	c.logger.WithField("container", name).Debug("starting container")

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

// Stop a container if it is stopped
func (c *Client) StopContainer(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name or id is required to stop a container")
	}
	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "stop", name)

	c.logger.WithField("container", name).Debug("stopping container")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// Delete a container
// TODO: need options such as `-v` to remove anonymous volumes on delete
func (c *Client) DeleteContainer(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name or id is required to delete a container")
	}

	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "rm", name)

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil

}

// Inspect container state
func (c *Client) InspectContainer(ctx context.Context, name string) (*provider.ContainerInfo, error) {
	var response *provider.ContainerInfo

	if name == "" {
		return response, fmt.Errorf("name is required to inspect a container")
	}

	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "inspect", "--format", "{{ . | json }}", name)

	c.logger.WithField("command", cmd.String()).Debug("Running container inspect command")

	out, err := cmd.Output()
	if err != nil {
		// Check if container doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			c.logger.WithField("name", name).Debug("container not found")
			return nil, nil
		}
		c.logger.WithFields(log.Fields{"output": out, "err": err}).Debug("error")
		return response, fmt.Errorf("failed to inspect container: %w", err)
	}

	res := &container.InspectResponse{}

	err = json.Unmarshal(out, res)
	if err != nil {
		return response, fmt.Errorf("failed to unmarshal inspect response: %w", err)
	}

	response = &provider.ContainerInfo{
		ID:       res.ID,
		Name:     res.Name,
		Created:  res.Created,
		HostName: res.Config.Hostname,
		Status:   res.State.Status,
		Image:    res.Config.Image,
	}

	return response, nil
}

// listEntry represents the JSON output from docker container ls
type listEntry struct {
	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	ID           string `json:"ID"`
	Image        string `json:"Image"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Names        string `json:"Names"`
	Networks     string `json:"Networks"`
	Ports        string `json:"Ports"`
	RunningFor   string `json:"RunningFor"`
	Size         string `json:"Size"`
	State        string `json:"State"`
	Status       string `json:"Status"`
}

// List containers
func (c *Client) ListContainers(ctx context.Context, filters []string) ([]provider.ContainerInfo, error) {
	var response []provider.ContainerInfo

	cmd := baseContainerCmd(ctx)
	cmd.Args = append(cmd.Args, "ls", "--format", "{{ . | json }}")

	for _, f := range filters {
		cmd.Args = append(cmd.Args, "--filter", f)
	}

	c.logger.WithField("command", cmd.String()).Debug("Running container list command")

	out, err := cmd.Output()
	if err != nil {
		return response, fmt.Errorf("failed to list containers: %w", err)
	}

	if len(out) == 0 {
		return response, nil
	}

	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry listEntry

		err = json.Unmarshal(line, &entry)
		if err != nil {
			return response, fmt.Errorf("failed to unmarshal list response: %w", err)
		}

		response = append(response, provider.ContainerInfo{
			ID:     entry.ID,
			Name:   entry.Names,
			Status: entry.State,
			Image:  entry.Image,
		})
	}

	return response, nil

}
