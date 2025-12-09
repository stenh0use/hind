package factory

import (
	"fmt"

	"github.com/stenh0use/hind/pkg/cluster/cni"
	"github.com/stenh0use/hind/pkg/cluster/cni/cilium"
	"github.com/stenh0use/hind/pkg/cluster/cni/none"
)

// DefaultFactory implements CNI factory
type DefaultFactory struct{}

// NewDefaultFactory creates a new default CNI factory
func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{}
}

// CreateCNI creates a CNI instance based on type and configuration
func (f *DefaultFactory) CreateCNI(cniType cni.CNIType, config map[string]string) (cni.CNI, error) {
	switch cniType {
	case cni.CNITypeNone:
		return none.NewNoneCNI(), nil

	case cni.CNITypeCilium:
		name := config["name"]
		if name == "" {
			name = "cilium"
		}

		ipv4Range := config["ipv4_range"]
		if ipv4Range == "" {
			ipv4Range = "10.8.0.0/16"
		}

		// Remove standard config keys and pass the rest as options
		options := make(map[string]string)
		for k, v := range config {
			if k != "name" && k != "ipv4_range" {
				options[k] = v
			}
		}

		return cilium.NewCiliumCNI(name, ipv4Range, options), nil

	default:
		return nil, fmt.Errorf("unsupported CNI type: %s", cniType)
	}
}
