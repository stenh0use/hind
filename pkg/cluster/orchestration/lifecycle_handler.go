package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cluster/plan"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
	hindversion "github.com/stenh0use/hind/pkg/version"
)

// LifecycleHandler implements Lifecycle using injected ports.
type LifecycleHandler struct {
	name    domain.Name
	repo    persistence.Repository
	active  persistence.ActiveRepository
	rt      runtime.Runtime
	logger  log.Interface
	waiter  Waiter
	timeout time.Duration
	poll    time.Duration
}

// LifecycleHandlerOptions configures a LifecycleHandler.
type LifecycleHandlerOptions struct {
	Name         domain.Name
	Repo         persistence.Repository
	Active       persistence.ActiveRepository
	Runtime      runtime.Runtime
	Logger       log.Interface
	Waiter       Waiter
	StartTimeout time.Duration
	PollInterval time.Duration
}

func (o *LifecycleHandlerOptions) defaults() {
	if o.Logger == nil {
		o.Logger = &log.Logger{Handler: discard.Default, Level: log.InfoLevel}
	}
	if o.StartTimeout == 0 {
		o.StartTimeout = 30 * time.Second
	}
	if o.PollInterval == 0 {
		o.PollInterval = time.Second
	}
	if o.Waiter == nil && o.Runtime != nil {
		o.Waiter = NewRuntimeWaiter(o.Runtime, o.PollInterval)
	}
}

// NewLifecycleHandler constructs a handler that uses only port interfaces.
func NewLifecycleHandler(opts LifecycleHandlerOptions) (*LifecycleHandler, error) {
	opts.defaults()
	if opts.Name == "" {
		return nil, fmt.Errorf("cluster name required")
	}
	if opts.Repo == nil {
		return nil, fmt.Errorf("repository required")
	}
	if opts.Runtime == nil {
		return nil, fmt.Errorf("runtime required")
	}
	return &LifecycleHandler{
		name:    opts.Name,
		repo:    opts.Repo,
		active:  opts.Active,
		rt:      opts.Runtime,
		logger:  opts.Logger,
		waiter:  opts.Waiter,
		timeout: opts.StartTimeout,
		poll:    opts.PollInterval,
	}, nil
}

// Start creates or resumes a cluster.
func (h *LifecycleHandler) Start(ctx context.Context, req StartRequest) (domain.StartOutcome, error) {
	h.logger.Info("starting cluster")

	existed, err := h.repo.Exists(ctx, h.name)
	if err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("start: check existence: %w", err)
	}

	var desired domain.Cluster
	if existed {
		desired, err = h.repo.Load(ctx, h.name)
		if err != nil {
			return domain.StartOutcomeCreated, fmt.Errorf("start: load: %w", err)
		}
	} else {
		desired, err = domain.BuildDefaultCluster(string(h.name), hindversion.HindVersion)
		if err != nil {
			return domain.StartOutcomeCreated, fmt.Errorf("start: build default: %w", err)
		}
	}

	// Apply client count override (only on create).
	if !existed && req.ClientCount > 0 && req.ClientCount != 1 {
		if err := domain.SetClientCount(&desired, req.ClientCount, false); err != nil {
			return domain.StartOutcomeCreated, fmt.Errorf("start: set client count: %w", err)
		}
	}

	if err := req.Validate(); err != nil {
		return domain.StartOutcomeCreated, err
	}

	// Apply version overrides.
	applyVersionOverrides(&desired, req)

	snap, err := h.rt.Inspect(ctx, runtime.Selector{Cluster: string(h.name)})
	if err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("start: inspect: %w", err)
	}

	p, err := plan.Build(desired, snap, plan.GoalStart)
	if err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("start: plan: %w", err)
	}

	if p.Noop && existed {
		h.logger.Info("cluster already running")
		return domain.StartOutcomeNoOp, nil
	}

	for _, op := range p.Operations {
		runtimeOp := runtime.Operation{
			ID:   string(op.ID),
			Kind: string(op.Kind),
			Resource: runtime.ResourceRef{
				Kind: runtime.ResourceKind(op.Resource.Kind),
				Name: op.Resource.Name,
			},
			Spec: planSpecToRuntimeSpec(op.Spec),
		}
		if _, err := h.rt.Apply(ctx, runtimeOp); err != nil {
			return domain.StartOutcomeCreated, fmt.Errorf("start: apply %s %s: %w", op.Kind, op.Resource.Name, err)
		}
	}

	startCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	if err := h.waiter.WaitRunning(startCtx, desired); err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("start: wait: %w", err)
	}

	if err := h.repo.Save(ctx, desired); err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("start: save: %w", err)
	}

	if !existed {
		h.logger.Info("cluster created")
		return domain.StartOutcomeCreated, nil
	}
	h.logger.Info("cluster resumed")
	return domain.StartOutcomeResumed, nil
}

// StopWithOptions stops all running containers in the cluster.
func (h *LifecycleHandler) StopWithOptions(ctx context.Context, opts StopOptions) (StopResult, error) {
	result := StopResult{}
	h.logger.Info("stopping cluster")

	exists, err := h.repo.Exists(ctx, h.name)
	if err != nil {
		return result, fmt.Errorf("stop: check existence: %w", err)
	}
	if !exists {
		return result, &NotFoundError{Operation: "stop", Cluster: string(h.name)}
	}

	desired, err := h.repo.Load(ctx, h.name)
	if err != nil {
		return result, fmt.Errorf("stop: load: %w", err)
	}

	snap, err := h.rt.Inspect(ctx, runtime.Selector{Cluster: string(h.name)})
	if err != nil {
		return result, fmt.Errorf("stop: inspect: %w", err)
	}

	p, err := plan.Build(desired, snap, plan.GoalStop)
	if err != nil {
		return result, fmt.Errorf("stop: plan: %w", err)
	}

	for _, op := range p.Operations {
		runtimeOp := runtime.Operation{
			ID:   string(op.ID),
			Kind: string(op.Kind),
			Resource: runtime.ResourceRef{
				Kind: runtime.ResourceKind(op.Resource.Kind),
				Name: op.Resource.Name,
			},
		}
		if _, err := h.rt.Apply(ctx, runtimeOp); err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, op.Resource.Name)
			h.logger.Warnf("failed to stop %s: %v", op.Resource.Name, err)
			continue
		}
		result.StoppedCount++
		result.StoppedContainers = append(result.StoppedContainers, op.Resource.Name)
	}

	// Count pre-failed and already-stopped containers from the pre-stop snapshot.
	for _, n := range desired.Nodes {
		c, ok := snap.Containers[n.Name]
		if !ok {
			continue
		}
		if c.Status != runtime.ContainerRunning {
			result.AlreadyStoppedCount++
		}
		if c.Status == runtime.ContainerUnhealthy || c.Status == runtime.ContainerUnknown {
			result.FailedPreStopCount++
		}
	}

	switch {
	case result.FailedCount > 0:
		result.Outcome = StopOutcomePartialFailure
	case result.FailedPreStopCount > 0:
		result.Outcome = StopOutcomeDegradedPreFailed
	case result.StoppedCount == 0 && result.AlreadyStoppedCount > 0:
		result.Outcome = StopOutcomeAlreadyStopped
	default:
		result.Outcome = StopOutcomeSuccess
	}

	return result, nil
}

// Delete removes all cluster resources and persisted configuration.
func (h *LifecycleHandler) Delete(ctx context.Context) error {
	h.logger.Info("deleting cluster")

	exists, err := h.repo.Exists(ctx, h.name)
	if err != nil {
		return fmt.Errorf("delete: check existence: %w", err)
	}

	snap, err := h.rt.Inspect(ctx, runtime.Selector{Cluster: string(h.name)})
	if err != nil {
		return fmt.Errorf("delete: inspect: %w", err)
	}

	if !exists && !snapshotHasClusterArtifacts(snap) {
		// Attempt to remove a stale cluster directory (no cluster.json, no runtime
		// artifacts). repo.Delete returns nil on success or ErrNotFound when the
		// directory also does not exist.
		if err := h.repo.Delete(ctx, h.name); err != nil {
			return &NotFoundError{Operation: "delete", Cluster: string(h.name)}
		}
		return nil
	}

	desired, err := h.loadOrSynthesizeCluster(ctx, exists, snap)
	if err != nil {
		return err
	}

	if !exists {
		h.logger.Info("cluster config missing; using runtime-derived desired state for delete")
	}

	p, err := plan.Build(desired, snap, plan.GoalDelete)
	if err != nil {
		return fmt.Errorf("delete: plan: %w", err)
	}

	for _, op := range p.Operations {
		runtimeOp := runtime.Operation{
			ID:   string(op.ID),
			Kind: string(op.Kind),
			Resource: runtime.ResourceRef{
				Kind: runtime.ResourceKind(op.Resource.Kind),
				Name: op.Resource.Name,
			},
		}
		if _, err := h.rt.Apply(ctx, runtimeOp); err != nil {
			return fmt.Errorf("delete: apply %s %s: %w", op.Kind, op.Resource.Name, err)
		}
	}

	if exists {
		if err := h.repo.Delete(ctx, h.name); err != nil {
			return fmt.Errorf("delete: persist: %w", err)
		}
	}

	h.logger.Info("cluster deleted")
	return nil
}

func (h *LifecycleHandler) loadOrSynthesizeCluster(ctx context.Context, exists bool, snap runtime.Snapshot) (domain.Cluster, error) {
	if exists {
		desired, err := h.repo.Load(ctx, h.name)
		if err != nil {
			return domain.Cluster{}, fmt.Errorf("delete: load: %w", err)
		}
		return desired, nil
	}

	desired, err := domain.BuildDefaultCluster(string(h.name), hindversion.HindVersion)
	if err != nil {
		return domain.Cluster{}, fmt.Errorf("delete: build default: %w", err)
	}

	return desired, nil
}

func snapshotHasClusterArtifacts(snap runtime.Snapshot) bool {
	if snap.Network != nil {
		return true
	}
	return len(snap.Containers) > 0
}

func applyVersionOverrides(c *domain.Cluster, req StartRequest) {
	if req.NomadVersion == "" && req.ConsulVersion == "" && req.VaultVersion == "" {
		return
	}
	for i := range c.Nodes {
		switch c.Nodes[i].Kind {
		case domain.KindNomad:
			if req.NomadVersion != "" {
				c.Nodes[i].Image.Tag = req.NomadVersion
			}
		case domain.KindConsul:
			if req.ConsulVersion != "" {
				c.Nodes[i].Image.Tag = req.ConsulVersion
			}
		case domain.KindVault:
			if req.VaultVersion != "" {
				c.Nodes[i].Image.Tag = req.VaultVersion
			}
		}
	}
}

func planSpecToRuntimeSpec(s *plan.ResourceSpec) *runtime.ResourceSpec {
	if s == nil {
		return nil
	}
	return &runtime.ResourceSpec{
		Image:       s.Image,
		Environment: s.Environment,
		Ports:       portMappings(s.Ports),
		Devices:     s.Devices,
		Network:     domain.NetworkName(s.Network),
		Labels:      s.Labels,
		SpecHash:    s.SpecHash,
	}
}

func portMappings(ports []int32) []domain.PortMapping {
	if len(ports) == 0 {
		return nil
	}
	out := make([]domain.PortMapping, 0, len(ports))
	for _, p := range ports {
		out = append(out, domain.PortMapping{ContainerPort: p, HostPort: p})
	}
	return out
}
