package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stenh0use/hind/pkg/config"
)

// formatBool formats a boolean for display
func formatBool(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

// formatPorts formats port mappings for display
func formatPorts(ports []config.PortMapping) string {
	if len(ports) == 0 {
		return "-"
	}

	var portStrs []string
	for _, port := range ports {
		if port.ListenAddress != "" {
			portStrs = append(portStrs, fmt.Sprintf("%s:%s->%s/%s", port.ListenAddress, int32String(port.HostPort), int32String(port.ContainerPort), port.Protocol))
		} else {
			portStrs = append(portStrs, fmt.Sprintf("%s->%s/%s", int32String(port.HostPort), int32String(port.ContainerPort), port.Protocol))
		}
	}

	return strings.Join(portStrs, ", ")
}

func int32String(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}
