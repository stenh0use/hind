package domain

import "fmt"

func ConsulNodeName(cluster string, id int) NodeName {
	return NodeName(fmt.Sprintf("hind.%s.consul.%.2d", cluster, id))
}
func NomadServerNodeName(cluster string, id int) NodeName {
	return NodeName(fmt.Sprintf("hind.%s.nomad.%.2d", cluster, id))
}
func ClientNodeName(cluster string, id int) NodeName {
	return NodeName(fmt.Sprintf("hind.%s.client.%.2d", cluster, id))
}
func VaultNodeName(cluster string, id int) NodeName {
	return NodeName(fmt.Sprintf("hind.%s.vault.%.2d", cluster, id))
}
