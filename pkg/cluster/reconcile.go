package cluster

import (
	"context"
	"fmt"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

// ReconcilePlan represents the difference between desired and actual state
type ReconcilePlan struct {
	NetworkToCreate      *config.Network
	ContainersToCreate   []config.Node
	ContainersToStart    []string
	ContainersToRecreate []RecreateAction
}

// RecreateAction describes a container that needs to be recreated
type RecreateAction struct {
	ExistingName string
	NewConfig    config.Node
	Reason       string // "unhealthy", "config_mismatch", etc
}

// ActualState represents the current state in Docker
type ActualState struct {
	Network    *provider.NetworkInfo
	Containers map[string]*provider.ContainerInfo // keyed by container name
}

// IsEmpty returns true if there are no changes to make
func (p *ReconcilePlan) IsEmpty() bool {
	return p.NetworkToCreate == nil &&
		len(p.ContainersToCreate) == 0 &&
		len(p.ContainersToStart) == 0 &&
		len(p.ContainersToRecreate) == 0
}

// Reconcile brings actual state in line with desired state (config)
// This is the ONLY method that should directly manipulate infrastructure
func (m *Manager) Reconcile(ctx context.Context) error {
	m.logger.Debug("Starting reconciliation")

	// 1. Get actual state from Docker
	actual, err := m.getActualState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get actual state: %w", err)
	}

	// 2. Calculate what needs to change
	plan, err := m.calculateReconcilePlan(ctx, actual)
	if err != nil {
		return fmt.Errorf("failed to calculate reconcile plan: %w", err)
	}

	if plan.IsEmpty() {
		m.logger.Info("Cluster state matches desired configuration")
		return nil
	}

	m.logger.Infof("Reconciliation plan: create=%d, start=%d, recreate=%d",
		len(plan.ContainersToCreate),
		len(plan.ContainersToStart),
		len(plan.ContainersToRecreate))

	// 3. Execute plan
	if err := m.executeReconcilePlan(ctx, plan); err != nil {
		return fmt.Errorf("failed to execute reconcile plan: %w", err)
	}

	// 4. Verify convergence
	m.logger.Debug("Waiting for containers to reach running state")
	if err := m.waitForContainersRunning(ctx, DefaultContainerStartTimeout); err != nil {
		return fmt.Errorf("cluster did not converge: %w", err)
	}

	// 5. Persist config only after successful reconciliation
	if err := m.saveConfig(); err != nil {
		return fmt.Errorf("failed to save config after reconciliation: %w", err)
	}

	m.logger.Info("Reconciliation completed successfully")
	return nil
}

// getActualState queries Docker for current cluster state
func (m *Manager) getActualState(ctx context.Context) (*ActualState, error) {
	state := &ActualState{
		Containers: make(map[string]*provider.ContainerInfo),
	}

	// Query network
	if m.config.Network.Name != "" {
		netInfo, err := m.provider.InspectNetwork(ctx, m.config.Network.Name)
		if err == nil && netInfo != nil {
			state.Network = netInfo
		}
		// Note: InspectNetwork returns error if not found, which is fine - it means we need to create it
	}

	// Query all containers by name from config
	for _, node := range m.config.Nodes {
		containerInfo, err := m.provider.InspectContainer(ctx, node.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container '%s': %w", node.Name, err)
		}
		if containerInfo != nil {
			state.Containers[node.Name] = containerInfo
		}
	}

	return state, nil
}

// calculateReconcilePlan compares desired vs actual and produces a plan
func (m *Manager) calculateReconcilePlan(ctx context.Context, actual *ActualState) (*ReconcilePlan, error) {
	plan := &ReconcilePlan{
		ContainersToCreate:   []config.Node{},
		ContainersToStart:    []string{},
		ContainersToRecreate: []RecreateAction{},
	}

	// Check network
	if actual.Network == nil && m.config.Network.Name != "" {
		plan.NetworkToCreate = &m.config.Network
	}

	// Check each desired node
	for _, desiredNode := range m.config.Nodes {
		actualContainer := actual.Containers[desiredNode.Name]

		if actualContainer == nil {
			// Container doesn't exist - needs creation
			plan.ContainersToCreate = append(plan.ContainersToCreate, desiredNode)
		} else {
			// Container exists - check state
			switch actualContainer.Status {
			case provider.Running.String():
				// Running - assume good for now
				// Future: add config drift detection here
				continue

			case provider.Error.String():
				// Unhealthy - needs recreation
				plan.ContainersToRecreate = append(plan.ContainersToRecreate, RecreateAction{
					ExistingName: desiredNode.Name,
					NewConfig:    desiredNode,
					Reason:       "unhealthy",
				})

			default:
				// Stopped - needs start
				plan.ContainersToStart = append(plan.ContainersToStart, desiredNode.Name)
			}
		}
	}

	return plan, nil
}

// executeReconcilePlan executes infrastructure changes
func (m *Manager) executeReconcilePlan(ctx context.Context, plan *ReconcilePlan) error {
	labels := config.Labels{
		"hind.cluster": m.config.Name,
		"hind.version": m.config.Version,
	}

	// Step 1: Create network if needed
	if plan.NetworkToCreate != nil {
		m.logger.Infof("Creating network '%s'", plan.NetworkToCreate.Name)
		plan.NetworkToCreate.Labels = labels
		id, err := m.provider.CreateNetwork(ctx, *plan.NetworkToCreate)
		if err != nil {
			return fmt.Errorf("failed to create network: %w", err)
		}
		m.logger.Infof("Created network '%s' (id: %s)", plan.NetworkToCreate.Name, id)
	}

	// Step 2: Recreate unhealthy containers
	for _, action := range plan.ContainersToRecreate {
		m.logger.Infof("Recreating unhealthy container '%s'", action.ExistingName)

		// Stop (ignore errors if already stopped)
		_ = m.provider.StopContainer(ctx, action.ExistingName)

		// Delete
		if err := m.provider.DeleteContainer(ctx, action.ExistingName); err != nil {
			return fmt.Errorf("failed to delete container '%s': %w", action.ExistingName, err)
		}

		// Recreate
		action.NewConfig.Labels = labels
		id, err := m.provider.CreateContainer(ctx, action.NewConfig)
		if err != nil {
			return fmt.Errorf("failed to recreate container '%s': %w", action.ExistingName, err)
		}
		m.logger.Infof("Recreated container '%s' (id: %s)", action.ExistingName, id)
	}

	// Step 3: Create new containers
	for _, node := range plan.ContainersToCreate {
		m.logger.Infof("Creating container '%s'", node.Name)
		node.Labels = labels
		id, err := m.provider.CreateContainer(ctx, node)
		if err != nil {
			return fmt.Errorf("failed to create container '%s': %w", node.Name, err)
		}
		m.logger.Infof("Created container '%s' (id: %s)", node.Name, id)
	}

	// Step 4: Start stopped containers
	for _, name := range plan.ContainersToStart {
		m.logger.Infof("Starting container '%s'", name)
		if err := m.provider.StartContainer(ctx, name); err != nil {
			return fmt.Errorf("failed to start container '%s': %w", name, err)
		}
		m.logger.Infof("Started container '%s'", name)
	}

	return nil
}
