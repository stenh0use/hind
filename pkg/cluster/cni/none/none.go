package none

import (
	"github.com/stenh0use/hind/pkg/cluster/cni"
)

// NoneCNI represents no CNI (disabled networking)
type NoneCNI struct{}

// NewNoneCNI creates a new instance of NoneCNI
func NewNoneCNI() *NoneCNI {
	return &NoneCNI{}
}

// Type returns the CNI type
func (n *NoneCNI) Type() cni.CNIType {
	return cni.CNITypeNone
}

// Enabled returns whether this CNI is enabled (always false for none)
func (n *NoneCNI) Enabled() bool {
	return false
}

// Start is a no-op for none CNI
func (n *NoneCNI) Start() error {
	return nil
}

// Stop is a no-op for none CNI
func (n *NoneCNI) Stop() error {
	return nil
}

// Status returns that CNI is disabled
func (n *NoneCNI) Status() (string, error) {
	return "disabled", nil
}

// GetEnvironmentVars returns empty environment variables
func (n *NoneCNI) GetEnvironmentVars() map[string]string {
	return map[string]string{}
}
