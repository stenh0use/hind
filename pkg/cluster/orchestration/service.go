package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

var errInvalidStartRequest = errors.New("invalid start request")

// Service defines root cluster orchestration operations.
type Service interface {
	Start(ctx context.Context, req StartRequest) (domain.StartOutcome, error)
	StopWithOptions(ctx context.Context, opts StopOptions) (StopResult, error)
	Delete(ctx context.Context) error
	Scale(ctx context.Context, targetClientCount int) error
	Inspect(ctx context.Context) (InspectResult, error)
	List(ctx context.Context) (ListResult, error)
}

// StopOutcome describes canonical stop policy outcomes.
type StopOutcome string

const (
	StopOutcomeSuccess           StopOutcome = "success"
	StopOutcomeAlreadyStopped    StopOutcome = "already-stopped"
	StopOutcomeDegradedPreFailed StopOutcome = "degraded-pre-failed"
	StopOutcomePartialFailure    StopOutcome = "partial-failure"
)

// StopResult summarizes a stop operation.
type StopResult struct {
	Outcome             StopOutcome
	StoppedCount        int
	AlreadyStoppedCount int
	FailedCount         int
	FailedPreStopCount  int
	Failures            []string
	StoppedContainers   []string
}

// AlreadyStopped reports whether all containers were already stopped.
func (r StopResult) AlreadyStopped() bool {
	return r.Outcome == StopOutcomeAlreadyStopped
}

// InspectResult is a runtime-neutral inspect DTO.
type InspectResult = domain.InspectResult

// ContainerSummary is a runtime-neutral container DTO.
type ContainerSummary = domain.ContainerSummary

// ListResult is a runtime-neutral list DTO.
type ListResult struct {
	Names []string
}

// StopOptions controls stop behavior.
type StopOptions struct {
	Force bool
}

// Options configures service wiring.
type Options struct {
	Lifecycle Lifecycle
	Scale     Scaler
	// Inspect and List may be left nil when Repo and Runtime are provided;
	// ensureDefaults will supply inline implementations backed by those ports.
	// Tests that need fakes should set these fields directly.
	Inspect Inspector
	List    Lister
	Locker  persistence.Locker
	Name    domain.Name
	// Repo and Runtime back the default Inspect and List implementations.
	// They are ignored when Inspect / List are already set.
	Repo    persistence.Repository
	Runtime runtime.Runtime
}

func (o *Options) ensureDefaults() {
	if o.Locker == nil {
		o.Locker = NewInMemoryLocker()
	}
	if o.Inspect == nil && o.Runtime != nil {
		o.Inspect = defaultInspector{name: o.Name, rt: o.Runtime}
	}
	if o.List == nil && o.Repo != nil {
		o.List = defaultLister{repo: o.Repo}
	}
}

var ErrNilContext = errors.New("context cannot be nil")

func validateContext(ctx context.Context, operation string) error {
	if ctx == nil {
		return ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func runMutating(ctx context.Context, opts Options, operation string, fn func(context.Context) error) error {
	if err := validateContext(ctx, operation); err != nil {
		return err
	}
	lockName := opts.Name
	if lockName == "" {
		lockName = domain.Name("default")
	}
	if err := opts.Locker.WithClusterLock(ctx, lockName, fn); err != nil {
		return err
	}
	return nil
}

func runReadOnly(ctx context.Context, operation string, fn func(context.Context) error) error {
	if err := validateContext(ctx, operation); err != nil {
		return err
	}
	return fn(ctx)
}

// StartRequest carries parameters for a create-or-start operation.
// The handler is bound to a cluster name at construction, so no Name field is needed here.
type StartRequest struct {
	ClientCount   int
	NomadVersion  string
	ConsulVersion string
	VaultVersion  string
}

// Validate checks start request inputs.
func (r *StartRequest) Validate() error {
	overrides := []struct {
		flagName string
		value    *string
	}{
		{flagName: "--nomad-version", value: &r.NomadVersion},
		{flagName: "--consul-version", value: &r.ConsulVersion},
		{flagName: "--vault-version", value: &r.VaultVersion},
	}

	for _, override := range overrides {
		trimmed := strings.TrimSpace(*override.value)
		if trimmed == "" {
			if *override.value == "" {
				continue
			}
			return fmt.Errorf("%w: %s: value must not be empty", errInvalidStartRequest, override.flagName)
		}
		*override.value = trimmed
		if !IsValidVersionToken(trimmed) {
			return fmt.Errorf("%w: %s: invalid format %q", errInvalidStartRequest, override.flagName, trimmed)
		}
	}
	return nil
}

// Lifecycle provides cluster lifecycle operations.
type Lifecycle interface {
	Start(ctx context.Context, req StartRequest) (domain.StartOutcome, error)
	StopWithOptions(ctx context.Context, opts StopOptions) (StopResult, error)
	Delete(ctx context.Context) error
}

// Scaler provides topology scaling.
type Scaler interface {
	Scale(ctx context.Context, targetClientCount int) error
}

// Inspector provides cluster inspection.
type Inspector interface {
	Inspect(ctx context.Context) (InspectResult, error)
}

// Lister provides cluster listing.
type Lister interface {
	List(ctx context.Context) (ListResult, error)
}
