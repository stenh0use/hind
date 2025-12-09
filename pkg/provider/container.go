package provider

// Common container options across providers
type containerOptions struct {
	Name   string
	Labels map[string]string
}

type ContainerOption func(*containerOptions)

type ContainerInfo struct {
	ID       string
	Name     string
	Created  string
	HostName string
	Status   string
	Image    string
	Ports    []string
	Labels   map[string]string
	Network  string
	Address  string
}

type ContainerSummary struct{}
