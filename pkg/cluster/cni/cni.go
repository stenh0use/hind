package cni

// CNI represents a Container Network Interface implementation
type CNI interface {
	// Type returns the CNI type (none, cilium)
	Type() CNIType

	// Enabled returns whether this CNI is enabled
	Enabled() bool

	// Start starts the CNI
	Start() error

	// Stop stops the CNI
	Stop() error

	// Status returns the current status
	Status() (string, error)

	// GetEnvironmentVars returns environment variables to inject into containers
	GetEnvironmentVars() map[string]string
}

// CNIType represents the type of CNI
type CNIType string

const (
	CNITypeNone   CNIType = "none"
	CNITypeCilium CNIType = "cilium"
)

// Factory can create CNI instances
type Factory interface {
	CreateCNI(cniType CNIType, config map[string]string) (CNI, error)
}
