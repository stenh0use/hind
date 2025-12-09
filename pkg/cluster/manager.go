package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/file"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

// Manager handles cluster lifecycle operations.
type Manager struct {
	logger     *log.Logger
	provider   provider.Client
	config     *config.Cluster
	fm         *file.Manager
	configFile string
}

// Config returns the cluster configuration
func (m *Manager) Config() *config.Cluster {
	return m.config
}

// SetConfig sets the cluster configuration
func (m *Manager) SetConfig(cfg *config.Cluster) {
	m.config = cfg
}

// New creates a new cluster manager with the given name and default configuration.
// It initializes the file manager, provider, and cluster configuration for the specified cluster name.
func New(logger *log.Logger, name string) (*Manager, error) {
	cfg, err := newClusterConfig(name, release.Latest().Hind)
	if err != nil {
		return nil, fmt.Errorf("failed to create default cluster config for '%s': %w", name, err)
	}
	logger.Debugf("created cluster defaults: %+v", cfg)

	fm, err := file.NewFromHomeDir(DefaultConfigParentDir, DefaultConfigName)
	if err != nil {
		return nil, fmt.Errorf("failed to create file manager with path: %w", err)
	}

	m := &Manager{
		logger:     logger,
		provider:   dockercli.New(logger),
		config:     cfg,
		fm:         fm,
		configFile: file.JoinPath(fm.GetRootDir(), ClusterConfigDir, name, ClusterConfigFile),
	}
	return m, nil
}

// Start starts or creates a cluster based on its current state.
// This is the declarative entry point - it makes the cluster running.
// If the cluster doesn't exist, it will be created.
// If the cluster exists but is stopped, it will be started.
// If the cluster is already running, this is a no-op (idempotent).
// Returns a StartResult indicating what action was taken.
func (m *Manager) Start(ctx context.Context) (StartResult, error) {
	m.logger.Debug("Starting cluster")

	existed := m.ConfigFileExists()

	if existed {
		// Load existing config
		cfg, err := m.loadConfig()
		if err != nil {
			return StartResultCreated, fmt.Errorf("failed to load cluster config: %w", err)
		}
		m.config = cfg
		m.logger.Debug("Loaded existing cluster configuration")
	} else {
		// Use the config created by New() - it already has defaults
		// Just ensure the directory exists
		clusterDir := file.JoinPath(m.fm.GetRootDir(), ClusterConfigDir, m.config.Name)
		if err := m.fm.EnsureDir(clusterDir); err != nil {
			return StartResultCreated, fmt.Errorf("failed to create cluster dir: %w", err)
		}
		m.logger.Debugf("Created cluster directory '%s'", clusterDir)
	}

	// Reconcile makes reality match config
	if err := m.Reconcile(ctx); err != nil {
		return StartResultCreated, err
	}

	// Determine result for user feedback
	if !existed {
		m.logger.Infof("Cluster '%s' created successfully", m.config.Name)
		return StartResultCreated, nil
	}

	m.logger.Infof("Cluster '%s' started successfully", m.config.Name)
	return StartResultResumed, nil
}

// waitForContainersRunning waits for all containers to reach running state
func (m *Manager) waitForContainersRunning(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		clusterInfo, err := m.Get(ctx)
		if err != nil {
			return err
		}

		allRunning := true
		for _, container := range clusterInfo.Containers {
			if container.Status != provider.Running.String() {
				allRunning = false
				m.logger.Debugf("Container '%s' is in state '%s', waiting...", container.Name, container.Status)
				break
			}
		}

		if allRunning {
			m.logger.Debug("All containers are running")
			return nil
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(DefaultContainerPollInterval):
			// Continue waiting
		}
	}

	return fmt.Errorf("timeout waiting for containers to reach running state")
}

func (m *Manager) Stop(ctx context.Context) error {
	// Load cluster config from disk if not already in memory
	// This allows Stop to work even if Manager was created without loading config
	if m.config == nil || m.config.Name == "" {
		cfg, err := m.loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load cluster config: %w", err)
		}
		m.config = cfg
	}

	// Track how many containers were stopped
	stoppedCount := 0
	alreadyStoppedCount := 0

	// Stop each node container
	for _, node := range m.config.Nodes {
		containerInfo, err := m.provider.InspectContainer(ctx, node.Name)

		// Skip if container doesn't exist
		if containerInfo == nil {
			m.logger.WithField("name", node.Name).Debug("container not found, skipping...")
			continue
		} else if err != nil {
			return err
		}

		// Check current status and stop if running
		if containerInfo.Status == provider.Running.String() {
			m.logger.WithField("name", node.Name).Debug("stopping container")
			if err := m.provider.StopContainer(ctx, node.Name); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", node.Name, err)
			}
			m.logger.WithField("name", node.Name).Info("stopped container")
			stoppedCount++
		} else {
			m.logger.WithField("name", node.Name).Debug("container already stopped")
			alreadyStoppedCount++
		}
	}

	// Log summary
	if stoppedCount == 0 && alreadyStoppedCount > 0 {
		m.logger.Debug("all containers already stopped")
	} else if stoppedCount > 0 {
		m.logger.Debugf("stopped %d container(s)", stoppedCount)
	}

	return nil
}

func (m *Manager) Delete(ctx context.Context) error {
	// Load cluster config from disk if not already in memory
	// This allows Delete to work even if Manager was created without loading config
	if m.config == nil || m.config.Name == "" {
		cfg, err := m.loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load cluster config: %w", err)
		}
		m.config = cfg
	}

	// TODO: make delete idempotent

	// TODO: get all containers by label and pass as one command
	// Delete cluster nodes
	for _, node := range m.config.Nodes {
		containerInfo, err := m.provider.InspectContainer(ctx, node.Name)
		if containerInfo == nil {
			m.logger.WithField("name", node.Name).Debug("container not found, skipping...")
			continue
		} else if err != nil {
			return err
		} else if containerInfo.Status == provider.Running.String() {
			if err = m.provider.StopContainer(ctx, node.Name); err != nil {
				return err
			}
		}

		if err := m.provider.DeleteContainer(ctx, node.Name); err != nil {
			return fmt.Errorf("failed to delete node '%s': %w", node.Name, err)
		}
		m.logger.WithField("name", node.Name).Info("deleted node")
	}

	// Check if network exists
	netInfo, err := m.provider.InspectNetwork(ctx, m.config.Network.Name)
	if err == nil && netInfo != nil {
		if err := m.provider.DeleteNetwork(ctx, m.config.Network.Name); err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}
		m.logger.WithField("name", m.config.Network.Name).Info("deleted network")
	}

	if err := m.fm.RemoveDir(file.JoinPath(ClusterConfigDir, m.config.Name)); err != nil {
		return fmt.Errorf("failed to remove cluster config directory: %w", err)
	}
	m.logger.WithField("name", m.config.Name).Info("deleted cluster")

	return nil
}

func (m *Manager) Get(ctx context.Context) (*provider.ClusterInfo, error) {
	state := &provider.ClusterInfo{}

	// Use in-memory config (don't load from disk)
	// This allows Get() to work during reconciliation before config is saved
	networkInfo, err := m.provider.InspectNetwork(ctx, m.config.Network.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network: %w", err)
	}
	state.Network = *networkInfo

	containerInfos := []provider.ContainerInfo{}
	for _, node := range m.config.Nodes {
		nodeInfo, err := m.provider.InspectContainer(ctx, node.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect node '%s': %w", node.Name, err)
		}
		// Skip containers that don't exist (nodeInfo will be nil)
		if nodeInfo != nil {
			containerInfos = append(containerInfos, *nodeInfo)
		}
	}

	state.Containers = containerInfos

	return state, nil
}

// Provider returns the provider client
func (m *Manager) Provider() provider.Client {
	return m.provider
}

// ConfigFileExists checks if the cluster config file exists
func (m *Manager) ConfigFileExists() bool {
	return m.fm.FileExists(m.configFile)
}

// SetClientCount updates the number of client nodes in the cluster configuration
func (m *Manager) SetClientCount(ctx context.Context, count int) error {
	if count < 1 {
		return fmt.Errorf("client count must be at least 1")
	}

	// Remove existing client nodes
	newNodes := []config.Node{}
	for _, node := range m.config.Nodes {
		if node.Role != config.Client {
			newNodes = append(newNodes, node)
		}
	}

	// Add new client nodes
	name := m.config.Name
	v, err := release.Get(m.config.Version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	for i := 0; i < count; i++ {
		nomadClient := config.Node{
			Name:    fmt.Sprintf("hind.%s.client.%.2d", name, i+1),
			Kind:    config.NomadNode,
			Role:    config.Client,
			Network: m.config.Network.Name,
			Image: config.Image{
				Name: release.NomadClient.ImageName(),
				Tag:  v.Hind,
			},
			Devices: []string{"/dev/fuse"},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE":     "client",
				"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", name, 1),
				"NOMAD_AGENT_MODE":      "client",
			},
		}
		newNodes = append(newNodes, nomadClient)
	}

	m.config.Nodes = newNodes
	return nil
}

func (m *Manager) loadConfig() (*config.Cluster, error) {
	data, err := m.fm.ReadFile(m.configFile)
	if err != nil {
		return nil, err
	}

	var cfg config.Cluster
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("loaded config file but no config was found")
	}

	return &cfg, nil
}

// saveConfig persists the current cluster configuration to disk
func (m *Manager) saveConfig() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := m.fm.WriteFile(m.configFile, data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.logger.Debug("Updated cluster configuration")
	return nil
}

// CountClientNodes returns the number of client nodes in the cluster
func (m *Manager) CountClientNodes() int {
	count := 0
	for _, node := range m.config.Nodes {
		if node.Role == config.Client {
			count++
		}
	}
	return count
}

// getClientNodes returns all client nodes from the cluster configuration
func (m *Manager) getClientNodes() []config.Node {
	clients := []config.Node{}
	for _, node := range m.config.Nodes {
		if node.Role == config.Client {
			clients = append(clients, node)
		}
	}
	return clients
}

// findNodeConfigByName finds a node configuration by container name
func (m *Manager) findNodeConfigByName(name string) *config.Node {
	for i := range m.config.Nodes {
		if m.config.Nodes[i].Name == name {
			return &m.config.Nodes[i]
		}
	}
	return nil
}

// Scale scales the cluster to the target number of client nodes.
// This is declarative - it updates the config and reconciles.
func (m *Manager) Scale(ctx context.Context, targetClientCount int) error {
	currentClientCount := m.CountClientNodes()

	if targetClientCount == currentClientCount {
		m.logger.Infof("Cluster already has %d client nodes", currentClientCount)
		return nil
	}

	if targetClientCount > currentClientCount {
		// Scale up: add node configs
		m.logger.Infof("Scaling up from %d to %d client nodes", currentClientCount, targetClientCount)
		if err := m.addClientNodes(targetClientCount - currentClientCount); err != nil {
			return err
		}
	} else {
		// Scale down: remove node configs
		m.logger.Infof("Scaling down from %d to %d client nodes", currentClientCount, targetClientCount)
		if err := m.removeClientNodes(currentClientCount - targetClientCount); err != nil {
			return err
		}
	}

	// Reconcile to make reality match config
	return m.Reconcile(ctx)
}

// addClientNodes adds N client node configs to the cluster config.
// Does NOT create infrastructure - just updates config.
func (m *Manager) addClientNodes(count int) error {
	m.logger.Debugf("Adding %d client node configs", count)

	currentClientCount := m.CountClientNodes()
	v, err := release.Get(m.config.Version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	name := m.config.Name

	for i := 0; i < count; i++ {
		nodeNum := currentClientCount + i + 1
		nomadClient := config.Node{
			Name:    fmt.Sprintf("hind.%s.client.%.2d", name, nodeNum),
			Kind:    config.NomadNode,
			Role:    config.Client,
			Network: m.config.Network.Name,
			Image: config.Image{
				Name: release.NomadClient.ImageName(),
				Tag:  v.Hind,
			},
			Devices: []string{"/dev/fuse"},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE":     "client",
				"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", name, 1),
				"NOMAD_AGENT_MODE":      "client",
			},
		}
		m.config.Nodes = append(m.config.Nodes, nomadClient)
	}

	return nil
}

// removeClientNodes removes N client node configs from the cluster config.
// Does NOT delete infrastructure - just updates config.
func (m *Manager) removeClientNodes(count int) error {
	m.logger.Debugf("Removing %d client node configs", count)

	clientNodes := m.getClientNodes()
	if len(clientNodes) < count {
		return fmt.Errorf("cannot remove %d clients, only %d exist", count, len(clientNodes))
	}

	// Remove last N client nodes from config
	nodesToRemove := clientNodes[len(clientNodes)-count:]
	namesToRemove := make(map[string]bool)
	for _, node := range nodesToRemove {
		namesToRemove[node.Name] = true
	}

	// Rebuild nodes slice without removed nodes
	newNodes := []config.Node{}
	for _, node := range m.config.Nodes {
		if !namesToRemove[node.Name] {
			newNodes = append(newNodes, node)
		}
	}
	m.config.Nodes = newNodes

	return nil
}
