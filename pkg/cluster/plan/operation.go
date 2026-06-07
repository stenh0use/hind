package plan

type Goal string

const (
	GoalStart  Goal = "start"
	GoalStop   Goal = "stop"
	GoalDelete Goal = "delete"
)

type OperationKind string

type OperationID string

const (
	OpCreateNetwork     OperationKind = "create_network"
	OpDeleteNetwork     OperationKind = "delete_network"
	OpCreateContainer   OperationKind = "create_container"
	OpStartContainer    OperationKind = "start_container"
	OpStopContainer     OperationKind = "stop_container"
	OpDeleteContainer   OperationKind = "delete_container"
	OpRecreateContainer OperationKind = "recreate_container"
)

type ResourceKind string

const (
	ResourceNetwork   ResourceKind = "network"
	ResourceContainer ResourceKind = "container"
)

type ResourceRef struct {
	Kind ResourceKind
	Name string
}

type ResourceSpec struct {
	Image       string
	Environment map[string]string
	Ports       []int32
	Devices     []string
	Network     string
	Labels      map[string]string
	SpecHash    string
}

type Operation struct {
	ID        OperationID
	Kind      OperationKind
	Resource  ResourceRef
	Spec      *ResourceSpec
	DependsOn []OperationID
}

type Plan struct {
	Goal       Goal
	Operations []Operation
	Noop       bool
}
