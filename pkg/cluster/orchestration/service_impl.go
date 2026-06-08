package orchestration

import (
	"context"
	"fmt"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

type rootService struct {
	lifecycle Lifecycle
	scale     Scaler
	inspect   Inspector
	list      Lister
	options   Options
	// repo, rt, and name back the default Inspector/Lister implementations
	// when Options.Inspect / Options.List are nil.
	repo persistence.Repository
	rt   runtime.Runtime
	name domain.Name
}

// NewService wires a root orchestration service facade.
func NewService(opts Options) Service {
	opts.ensureDefaults()
	svc := &rootService{
		lifecycle: opts.Lifecycle,
		scale:     opts.Scale,
		inspect:   opts.Inspect,
		list:      opts.List,
		options:   opts,
		repo:      opts.Repo,
		rt:        opts.Runtime,
		name:      opts.Name,
	}
	return svc
}

func (s *rootService) Start(ctx context.Context, req StartRequest) (domain.StartOutcome, error) {
	var out domain.StartOutcome
	err := runMutating(ctx, s.options, "start", func(lockCtx context.Context) error {
		res, err := s.lifecycle.Start(lockCtx, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

func (s *rootService) StopWithOptions(ctx context.Context, opts StopOptions) (StopResult, error) {
	var out StopResult
	err := runMutating(ctx, s.options, "stop", func(lockCtx context.Context) error {
		res, err := s.lifecycle.StopWithOptions(lockCtx, opts)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

func (s *rootService) Delete(ctx context.Context) error {
	return runMutating(ctx, s.options, "delete", func(lockCtx context.Context) error {
		return s.lifecycle.Delete(lockCtx)
	})
}

func (s *rootService) Scale(ctx context.Context, targetClientCount int) error {
	return runMutating(ctx, s.options, "scale", func(lockCtx context.Context) error {
		return s.scale.Scale(lockCtx, targetClientCount)
	})
}

func (s *rootService) Inspect(ctx context.Context) (InspectResult, error) {
	var out InspectResult
	err := runReadOnly(ctx, "inspect", func(lockCtx context.Context) error {
		res, err := s.inspect.Inspect(lockCtx)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

func (s *rootService) List(ctx context.Context) (ListResult, error) {
	var out ListResult
	err := runReadOnly(ctx, "list", func(lockCtx context.Context) error {
		res, err := s.list.List(lockCtx)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

// defaultInspector is the inline Inspector implementation used when no
// override is supplied via Options.Inspect.
type defaultInspector struct {
	name domain.Name
	rt   runtime.Runtime
}

func (d defaultInspector) Inspect(ctx context.Context) (InspectResult, error) {
	snap, err := d.rt.Inspect(ctx, runtime.Selector{Cluster: string(d.name)})
	if err != nil {
		return InspectResult{}, fmt.Errorf("inspect: %w", err)
	}

	if snap.Network == nil && len(snap.Containers) == 0 {
		return InspectResult{}, &NotFoundError{Operation: "inspect", Cluster: string(d.name)}
	}

	result := InspectResult{}
	if snap.Network != nil {
		result.NetworkName = string(snap.Network.Name)
	}

	for _, c := range snap.Containers {
		result.Containers = append(result.Containers, ContainerSummary{
			Name:     string(c.Name),
			Status:   runtimeToContainerStatus(c.Status),
			Image:    c.Image,
			ID:       c.ID,
			Created:  c.Created,
			HostName: c.HostName,
		})
	}
	return result, nil
}

// defaultLister is the inline Lister implementation used when no override is
// supplied via Options.List.
type defaultLister struct {
	repo persistence.Repository
}

func (d defaultLister) List(ctx context.Context) (ListResult, error) {
	names, err := d.repo.List(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("list: %w", err)
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, string(n))
	}
	return ListResult{Names: out}, nil
}

// runtimeToContainerStatus maps a runtime.ContainerStatus to a domain.ContainerStatus.
func runtimeToContainerStatus(s runtime.ContainerStatus) domain.ContainerStatus {
	switch s {
	case runtime.ContainerRunning:
		return domain.ContainerStatusRunning
	case runtime.ContainerStopped:
		return domain.ContainerStatusStopped
	case runtime.ContainerUnhealthy:
		return domain.ContainerStatusUnhealthy
	default:
		return domain.ContainerStatusUnknown
	}
}
