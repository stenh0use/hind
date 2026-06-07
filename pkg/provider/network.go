package provider

import "time"

type NetworkInfo struct {
	ID      string
	Name    string
	Created time.Time
	Driver  string
	Labels  map[string]string
}

// NetworkSpec is the runtime-neutral spec for creating a network.
type NetworkSpec struct {
	Name    string
	Driver  string
	Subnet  string
	Gateway string
	Labels  map[string]string
}
