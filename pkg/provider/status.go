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

// IsRunning reports whether this status represents a running container.
func (s Status) IsRunning() bool {
	return s == Running
}

// IsError reports whether this status represents an unhealthy/error container.
func (s Status) IsError() bool {
	return s == Error
}
