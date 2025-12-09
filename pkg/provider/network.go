package provider

import "time"

type NetworkInfo struct {
	ID      string
	Name    string
	Created time.Time
	Driver  string
	Status  string
	Image   string
	Ports   []string
	Labels  map[string]string
	Network string
	Address string
}

type NetworkSummary struct{}
