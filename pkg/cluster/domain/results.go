package domain

// StartOutcome describes what happened during a cluster start operation.
type StartOutcome int

const (
	// StartOutcomeCreated indicates a new cluster was created.
	StartOutcomeCreated StartOutcome = iota
	// StartOutcomeResumed indicates an existing cluster was started.
	StartOutcomeResumed
	// StartOutcomeAlreadyRunning indicates the cluster was already running.
	StartOutcomeAlreadyRunning
	// StartOutcomeScaled indicates the cluster changed size during start flow.
	StartOutcomeScaled
	// StartOutcomeNoOp indicates the start flow resulted in no changes.
	StartOutcomeNoOp
)

// ContainerStatus is the canonical typed container lifecycle state used across
// the cluster layer. It is derived from runtime.ContainerStatus by the runtime
// adapter and must not be re-parsed from raw strings downstream.
type ContainerStatus string

const (
	// ContainerStatusRunning indicates the container is running.
	ContainerStatusRunning ContainerStatus = "running"
	// ContainerStatusStopped indicates the container is stopped or exited.
	ContainerStatusStopped ContainerStatus = "stopped"
	// ContainerStatusUnhealthy indicates the container is in an error state.
	ContainerStatusUnhealthy ContainerStatus = "unhealthy"
	// ContainerStatusUnknown indicates the container status could not be determined.
	ContainerStatusUnknown ContainerStatus = "unknown"
)

// ContainerSummary is a runtime-neutral container descriptor.
// Status is typed as ContainerStatus; it is populated by the runtime adapter.
type ContainerSummary struct {
	Name     string
	ID       string
	Status   ContainerStatus
	Image    string
	Created  string
	HostName string
}

// InspectResult is a runtime-neutral inspect DTO.
type InspectResult struct {
	NetworkName string
	Containers  []ContainerSummary
}

// ClusterStatus is the typed aggregate cluster status derived from containers.
type ClusterStatus string

const (
	// ClusterStatusRunning indicates all containers are running.
	ClusterStatusRunning ClusterStatus = "running"
	// ClusterStatusStopped indicates all containers are stopped.
	ClusterStatusStopped ClusterStatus = "stopped"
	// ClusterStatusDegraded indicates one or more containers are unhealthy or unknown.
	ClusterStatusDegraded ClusterStatus = "degraded"
	// ClusterStatusPartial indicates a mix of running and stopped containers.
	ClusterStatusPartial ClusterStatus = "partial"
	// ClusterStatusNotFound indicates no containers were found for the cluster.
	ClusterStatusNotFound ClusterStatus = "not-found"
)

// AggregateContainerStatus derives a ClusterStatus from a slice of ContainerSummary.
// Rules:
//   - empty slice                          → ClusterStatusNotFound
//   - all running                          → ClusterStatusRunning
//   - all stopped                          → ClusterStatusStopped
//   - mix of running + stopped             → ClusterStatusPartial
//   - any unhealthy/unknown (with or
//     without running/stopped)             → ClusterStatusDegraded
func AggregateContainerStatus(containers []ContainerSummary) ClusterStatus {
	if len(containers) == 0 {
		return ClusterStatusNotFound
	}

	hasRunning := false
	hasStopped := false
	hasDegraded := false

	for _, c := range containers {
		switch c.Status {
		case ContainerStatusRunning:
			hasRunning = true
		case ContainerStatusStopped:
			hasStopped = true
		default:
			hasDegraded = true
		}
	}

	switch {
	case hasDegraded:
		return ClusterStatusDegraded
	case hasRunning && hasStopped:
		return ClusterStatusPartial
	case hasRunning:
		return ClusterStatusRunning
	default:
		return ClusterStatusStopped
	}
}
