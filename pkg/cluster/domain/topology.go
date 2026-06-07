package domain

import "fmt"

func BuildDefaultCluster(name string, version string) (Cluster, error) {
	if err := ValidateName(name); err != nil {
		return Cluster{}, err
	}

	network := Network{Name: NetworkName("hind." + name)}
	c := Cluster{
		Name:    Name(name),
		Version: version,
		Network: network,
		Nodes: []Node{
			newNode(name, network.Name, version, KindConsul, RoleServer, 1),
			newNode(name, network.Name, version, KindNomad, RoleServer, 1),
			newNomadClientNode(name, network.Name, version, 1),
			newNode(name, network.Name, version, KindVault, RoleServer, 1),
		},
	}

	if err := c.Validate(); err != nil {
		return Cluster{}, fmt.Errorf("invalid default topology: %w", err)
	}
	return c, nil
}

func SetClientCount(c *Cluster, count int, preserveIDs bool) error {
	if count < 1 {
		return fmt.Errorf("client count must be at least 1")
	}
	nonClients := make([]Node, 0, len(c.Nodes))
	clients := make([]Node, 0)
	for _, n := range c.Nodes {
		if n.Role == RoleClient {
			clients = append(clients, n)
		} else {
			nonClients = append(nonClients, n)
		}
	}
	if !preserveIDs {
		clients = []Node{}
		for i := 1; i <= count; i++ {
			clients = append(clients, newClient(*c, i))
		}
		c.Nodes = append(nonClients, clients...)
		return nil
	}
	for len(clients) > count {
		clients = clients[:len(clients)-1]
	}
	nextID := 1
	for _, n := range clients {
		if n.ID >= nextID {
			nextID = n.ID + 1
		}
	}
	for len(clients) < count {
		clients = append(clients, newClient(*c, nextID))
		nextID++
	}
	c.Nodes = append(nonClients, clients...)
	return nil
}
