package runtime

import "context"

type Runtime interface {
	Inspect(ctx context.Context, selector Selector) (Snapshot, error)
	Apply(ctx context.Context, op Operation) (OperationResult, error)
}

type Selector struct {
	Cluster string
}

type Operation struct {
	ID        string
	Kind      string
	Resource  ResourceRef
	Spec      *ResourceSpec
	DependsOn []string
}

type OperationResult struct {
	Resource ResourceRef
}
