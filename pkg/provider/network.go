package provider

import "time"

type NetworkInfo struct {
	ID      string
	Name    string
	Created time.Time
	Driver  string
	Labels  map[string]string
}
