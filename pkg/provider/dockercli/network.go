package dockercli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/moby/moby/api/types/network"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

const networkCmd = "network"

func baseNetworkCmd(ctx context.Context) *exec.Cmd {
	return baseClientCmd(ctx, networkCmd)
}

// Create a new docker network
func (c *Client) CreateNetwork(ctx context.Context, cfg config.Network) (string, error) {
	if cfg.Name == "" {
		return "", fmt.Errorf("name is required to create a network")
	}

	cmd := baseNetworkCmd(ctx)
	cmd.Args = append(cmd.Args, "create")

	if cfg.Driver != "" {
		cmd.Args = append(cmd.Args, "--driver", cfg.Driver)
	}
	if cfg.Subnet != "" {
		cmd.Args = append(cmd.Args, "--subnet", cfg.Subnet)
	}
	if cfg.Gateway != "" {
		cmd.Args = append(cmd.Args, "--gateway", cfg.Gateway)
	}
	if cfg.Labels != nil {
		for k, v := range cfg.Labels {
			cmd.Args = append(cmd.Args, "--label", fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd.Args = append(cmd.Args, cfg.Name)

	c.logger.WithField("command", cmd.String()).Debug("Running network create command")

	id, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	return string(id), nil
}

// Delete a network
func (c *Client) DeleteNetwork(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("name is required to delete a network")
	}

	cmd := baseNetworkCmd(ctx)
	cmd.Args = append(cmd.Args, "rm", name)

	c.logger.WithField("command", cmd.String()).Debug("Running network delete command")

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// Inspect network state
func (c *Client) InspectNetwork(ctx context.Context, name string) (*provider.NetworkInfo, error) {
	var response *provider.NetworkInfo

	if name == "" {
		return nil, fmt.Errorf("name is required to inspect a network")
	}

	cmd := baseNetworkCmd(ctx)
	cmd.Args = append(cmd.Args, "inspect", "--format", "{{ . | json }}", name)

	c.logger.WithField("command", cmd.String()).Debug("Running network inspect command")

	out, err := cmd.Output()
	if err != nil {
		// Check if network doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			c.logger.WithField("name", name).Debug("network not found")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to inspect network: %w", err)
	}

	res := &network.Network{}

	err = json.Unmarshal(out, res)
	if err != nil {
		c.logger.WithField("unmarshaled", out).Debug("partial info")
		return response, fmt.Errorf("failed to unmarshal inspect response: %w", err)
	}

	c.logger.WithField("unmarshaled", res).Debug("network info")

	response = &provider.NetworkInfo{
		ID:      res.ID,
		Name:    res.Name,
		Created: res.Created,
		Driver:  res.Driver,
		Labels:  res.Labels,
	}

	c.logger.WithField("NetworkInfo", response).Debug("network info")

	return response, nil

}

// List networks
func (c *Client) ListNetworks(ctx context.Context, filters []string) ([]provider.NetworkInfo, error) {
	var response []provider.NetworkInfo

	cmd := baseNetworkCmd(ctx)
	cmd.Args = append(cmd.Args, "ls", "--format", "{{ . | json }}")

	for _, f := range filters {
		cmd.Args = append(cmd.Args, "--filter", f)
	}

	c.logger.WithField("command", cmd.String()).Debug("Running network list command")

	out, err := cmd.Output()
	if err != nil {
		return response, fmt.Errorf("failed to inspect network: %w", err)
	}

	if len(out) == 0 {
		return response, nil
	}

	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		res := &networkSummary{}
		err = json.Unmarshal(line, res)
		if err != nil {
			c.logger.Debugf("%s", line)
			c.logger.WithField("unmarshaled", res).Debug("partial data")
			return response, fmt.Errorf("failed to unmarshal inspect response: %w", err)
		}
		labels := map[string]string{}
		if res.Labels != "" {
			for _, label := range strings.Split(res.Labels, ",") {
				label = strings.TrimSpace(label)
				if label == "" {
					continue
				}
				kvpair := strings.SplitN(label, "=", 2)
				if len(kvpair) == 2 {
					labels[kvpair[0]] = kvpair[1]
				} else {
					c.logger.WithField("label", label).Debug("skipping malformed label")
				}
			}
		}
		response = append(response, provider.NetworkInfo{
			ID:      res.ID,
			Name:    res.Name,
			Created: res.Created,
			Driver:  res.Driver,
			Labels:  labels,
		})
	}
	return response, nil
}

type networkSummary struct {

	// Name of the network.
	//
	// Example: my_network
	Name string `json:"Name"`

	// ID that uniquely identifies a network on a single machine.
	//
	// Example: 7d86d31b1478e7cca9ebed7e73aa0fdeec46c5ca29497431d3007d2d9e15ed99
	ID string `json:"Id"`

	// Date and time at which the network was created in
	// [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.
	//
	// Example: 2016-10-19T04:33:30.360899459Z
	Created time.Time `json:"Created"`

	// The level at which the network exists (e.g. `swarm` for cluster-wide
	// or `local` for machine level)
	//
	// Example: local
	Scope string `json:"Scope"`

	// The name of the driver used to create the network (e.g. `bridge`,
	// `overlay`).
	//
	// Example: overlay
	Driver string `json:"Driver"`

	// Whether the network was created with IPv4 enabled.
	//
	// Example: true
	EnableIPv4 bool `json:"EnableIPv4"`

	// Whether the network was created with IPv6 enabled.
	//
	// Example: false
	EnableIPv6 bool `json:"EnableIPv6"`

	// The network's IP Address Management.
	//
	IPAM network.IPAM `json:"IPAM"`

	// Whether the network is created to only allow internal networking
	// connectivity.
	//
	// Example: false
	Internal string `json:"Internal"`

	// Whether a global / swarm scope network is manually attachable by regular
	// containers from workers in swarm mode.
	//
	// Example: false
	Attachable bool `json:"Attachable"`

	// Whether the network is providing the routing-mesh for the swarm cluster.
	//
	// Example: false
	Ingress string `json:"Ingress"`

	// config from
	ConfigFrom network.ConfigReference `json:"ConfigFrom"`

	// Whether the network is a config-only network. Config-only networks are
	// placeholder networks for network configurations to be used by other
	// networks. Config-only networks cannot be used directly to run containers
	// or services.
	//
	ConfigOnly bool `json:"ConfigOnly"`

	// Network-specific options uses when creating the network.
	//
	// Example: {"com.docker.network.bridge.default_bridge":"true","com.docker.network.bridge.enable_icc":"true","com.docker.network.bridge.enable_ip_masquerade":"true","com.docker.network.bridge.host_binding_ipv4":"0.0.0.0","com.docker.network.bridge.name":"docker0","com.docker.network.driver.mtu":"1500"}
	Options map[string]string `json:"Options"`

	// Metadata specific to the network being created.
	//
	// Example: {"com.example.some-label":"some-value","com.example.some-other-label":"some-other-value"}
	Labels string `json:"Labels"`

	// List of peer nodes for an overlay network. This field is only present
	// for overlay networks, and omitted for other network types.
	//
	Peers []network.PeerInfo `json:"Peers,omitempty"`
}
