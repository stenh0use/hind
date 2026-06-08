package domain

import (
	"fmt"
	"strings"
)

func ValidateName(v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if v == "." || v == ".." || strings.HasPrefix(v, "/") || strings.Contains(v, "/") || strings.Contains(v, "\\") || strings.Contains(v, "..") {
		return fmt.Errorf("name must be relative and cannot contain path separators or traversal")
	}
	return nil
}

func (c Cluster) Validate() error {
	if err := ValidateName(string(c.Name)); err != nil {
		return err
	}
	if strings.TrimSpace(string(c.Network.Name)) == "" {
		return fmt.Errorf("network required")
	}
	seen := map[NodeName]bool{}
	hasConsul, hasNomadServer, hasClient := false, false, false
	for _, n := range c.Nodes {
		if seen[n.Name] {
			return fmt.Errorf("duplicate node name: %s", n.Name)
		}
		seen[n.Name] = true
		if n.ID < 1 {
			return fmt.Errorf("node id must be positive")
		}
		if n.Network != c.Network.Name {
			return fmt.Errorf("node %s references wrong network", n.Name)
		}
		for _, p := range n.Ports {
			if p.HostPort < 1 || p.ContainerPort < 1 {
				return fmt.Errorf("invalid ports for node %s", n.Name)
			}
		}
		if n.Kind == KindConsul && n.Role == RoleServer {
			hasConsul = true
		}
		if n.Kind == KindNomad && n.Role == RoleServer {
			hasNomadServer = true
		}
		if n.Kind == KindNomad && n.Role == RoleClient {
			hasClient = true
		}
	}
	if !hasConsul || !hasNomadServer || !hasClient {
		return fmt.Errorf("required baseline topology missing")
	}
	return nil
}
