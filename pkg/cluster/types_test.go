package cluster

import "testing"

func TestNewClusterConfig_VaultPortsAssignedOnce(t *testing.T) {
	cfg, err := newClusterConfig("test", "0.4.0")
	if err != nil {
		t.Fatalf("newClusterConfig() error = %v", err)
	}

	vaultCount := 0
	for _, node := range cfg.Nodes {
		if node.Kind != "vault" {
			continue
		}
		vaultCount++
		if len(node.Ports) != 1 {
			t.Fatalf("vault node ports len = %d, want 1", len(node.Ports))
		}
		if node.Ports[0].HostPort != 8200 || node.Ports[0].ContainerPort != 8200 {
			t.Fatalf("vault port mapping = %+v, want 8200:8200", node.Ports[0])
		}
	}

	if vaultCount == 0 {
		t.Fatal("expected at least one vault node")
	}
}
