package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cluster/plan"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

// ScaleHandler implements Scaler using injected ports.
type ScaleHandler struct {
	name    domain.Name
	repo    persistence.Repository
	rt      runtime.Runtime
	waiter  Waiter
	timeout time.Duration
}

// ScaleHandlerOptions configures a ScaleHandler.
type ScaleHandlerOptions struct {
	Name         domain.Name
	Repo         persistence.Repository
	Runtime      runtime.Runtime
	Waiter       Waiter
	StartTimeout time.Duration
	PollInterval time.Duration
}

func (o *ScaleHandlerOptions) defaults() {
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

// NewScaleHandler constructs a handler that uses only port interfaces.
func NewScaleHandler(opts ScaleHandlerOptions) (*ScaleHandler, error) {
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
	return &ScaleHandler{
		name:    opts.Name,
		repo:    opts.Repo,
		rt:      opts.Runtime,
		waiter:  opts.Waiter,
		timeout: opts.StartTimeout,
	}, nil
}

// Scale adjusts the cluster to the target number of client nodes.
func (h *ScaleHandler) Scale(ctx context.Context, targetClientCount int) error {
	desired, err := h.repo.Load(ctx, h.name)
	if err != nil {
		return fmt.Errorf("scale: load: %w", err)
	}

	if err := domain.SetClientCount(&desired, targetClientCount, true); err != nil {
		return fmt.Errorf("scale: set client count: %w", err)
	}

	snap, err := h.rt.Inspect(ctx, runtime.Selector{Cluster: string(h.name)})
	if err != nil {
		return fmt.Errorf("scale: inspect: %w", err)
	}

	p, err := plan.Build(desired, snap, plan.GoalStart)
	if err != nil {
		return fmt.Errorf("scale: plan: %w", err)
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
			return fmt.Errorf("scale: apply %s %s: %w", op.Kind, op.Resource.Name, err)
		}
	}

	scaleCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	if err := h.waiter.WaitRunning(scaleCtx, desired); err != nil {
		return fmt.Errorf("scale: wait: %w", err)
	}

	if err := h.repo.Save(ctx, desired); err != nil {
		return fmt.Errorf("scale: save: %w", err)
	}

	return nil
}
