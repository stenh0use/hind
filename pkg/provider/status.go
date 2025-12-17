package provider

// Status represents the overall different lifecycle states
type Status string

const (
	Stopped Status = "stopped"
	Running Status = "running"
	Error   Status = "error"
	NA      Status = "n/a"
)

func (s Status) String() string {
	return string(s)
}

type ClusterInfo struct {
	Name       string
	Containers []ContainerInfo
	Network    NetworkInfo
}
