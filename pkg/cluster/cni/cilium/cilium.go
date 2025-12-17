package cilium

import (
	"fmt"
	"strconv"

	"github.com/stenh0use/hind/pkg/cluster/cni"
)

// CiliumCNI implements CNI interface for Cilium
type CiliumCNI struct {
	name      string
	ipv4Range string
	options   map[string]string
	enabled   bool
}

// NewCiliumCNI creates a new instance of Cilium CNI
func NewCiliumCNI(name, ipv4Range string, options map[string]string) *CiliumCNI {
	if options == nil {
		options = make(map[string]string)
	}

	return &CiliumCNI{
		name:      name,
		ipv4Range: ipv4Range,
		options:   options,
		enabled:   true,
	}
}

// Type returns the CNI type
func (c *CiliumCNI) Type() cni.CNIType {
	return cni.CNITypeCilium
}

// Enabled returns whether this CNI is enabled
func (c *CiliumCNI) Enabled() bool {
	return c.enabled
}

// Start starts the Cilium CNI
func (c *CiliumCNI) Start() error {
	if !c.enabled {
		return nil
	}

	// TODO: Implement proper Cilium startup
	return nil
}

// Stop stops the Cilium CNI
func (c *CiliumCNI) Stop() error {
	if !c.enabled {
		return nil
	}

	// TODO: Implement proper Cilium shutdown
	return nil
}

// Status returns the current status of Cilium CNI
func (c *CiliumCNI) Status() (string, error) {
	if !c.enabled {
		return "disabled", nil
	}

	// TODO: Implement proper status checking
	return "running", nil
}

// GetEnvironmentVars returns environment variables for Cilium CNI
func (c *CiliumCNI) GetEnvironmentVars() map[string]string {
	if !c.enabled {
		return map[string]string{}
	}

	envVars := map[string]string{
		"CILIUM_ENABLED":    strconv.FormatBool(c.enabled),
		"CILIUM_IPV4_RANGE": c.ipv4Range,
	}

	// Add any custom options
	for k, v := range c.options {
		envVars[fmt.Sprintf("CILIUM_%s", k)] = v
	}

	return envVars
}
