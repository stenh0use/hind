# Hashistack in Docker (hind)

This repository takes inspiration from the Kubernetes in Docker (kind) project. It provides a quick and easy playground for HashiCorp's Nomad orchestrator, enabling multi-node clusters for testing, development, and failure scenario simulation.

## Features

- **Multi-node Nomad clusters** - Run server and client nodes in Docker containers
- **Integrated service discovery** - Consul integration for service mesh capabilities
- **Simple CLI** - Intuitive commands for cluster lifecycle management
- **Multiple clusters** - Run multiple isolated clusters simultaneously
- **Named profiles** - Save and reuse cluster configurations

## Implemented Stack Components

- Nomad (server and client nodes)
- Consul (service discovery and configuration)
- Vault (secrets management)

## Credits

Bits and pieces adapted from:
- https://github.com/kubernetes-sigs/kind
- https://github.com/multani/docker-nomad
- https://github.com/hashicorp/docker-consul

## Quick Start

### Build the CLI

```bash
make build
```

This builds the `hind` binary to `./bin/hind`.

### Build Docker Images

Build all required Docker images:

```bash
./bin/hind build all
```

Or build specific images:

```bash
./bin/hind build nomad    # Build Nomad image
./bin/hind build consul   # Build Consul image
```

### Cluster Management

Start a cluster (default name is "default"):

```bash
./bin/hind start
```

Start a named cluster with custom configuration:

```bash
./bin/hind start dev --clients=3
```

List all running clusters:

```bash
./bin/hind list
```

Get details about a specific cluster:

```bash
./bin/hind get dev
```

Stop a cluster (keeps containers for restart):

```bash
./bin/hind stop dev
```

Delete a cluster completely:

```bash
./bin/hind rm dev
```

### Accessing the Web UI

Once your cluster is running, access the web interfaces:

```bash
# Nomad UI
open http://localhost:4646/ui

# Consul UI
open http://localhost:8500/ui
```

### Running Nomad Jobs

After starting a cluster, you can submit jobs to Nomad:

```bash
nomad run jobs/example.hcl
```

## CLI Commands Reference

### Image Building

```bash
./bin/hind build <image>         # Build a specific image (nomad, consul, etc.)
./bin/hind build all              # Build all images
```

### Cluster Lifecycle

```bash
./bin/hind start [cluster-name]   # Create and start a cluster
  --clients int                   # Number of client nodes (default: 1)
  --version string                # Hind image version to use (default: "latest")
  --timeout duration              # Timeout for starting cluster (default: 5m)
  --verbose                       # Enable verbose output

./bin/hind list                   # List all clusters
./bin/hind get <name>             # Get details about a cluster
./bin/hind stop <name>            # Stop a cluster
./bin/hind rm <name>              # Delete a cluster completely
./bin/hind version                # Show version information
```

## Requirements

This project requires:
- **Go 1.21+** - For building the CLI
- **Docker** - For running containers
- **make** - For build automation

### Recommended Setup (macOS)

- [Colima](https://github.com/abiosoft/colima) 0.6.x+ with cgroups v2 enabled
- [Docker CLI](https://docs.docker.com/engine/install/binaries/#install-client-binaries-on-macos)

### Optional Tools

- [Nomad CLI](https://developer.hashicorp.com/nomad/downloads) - For submitting jobs and interacting with Nomad
- [Consul CLI](https://developer.hashicorp.com/consul/downloads) - For service discovery operations

## Tested Platforms

This project has been tested on:
- **macOS** with Colima 0.6.x+ (Docker 24.0+, cgroups v2)

The key requirement is cgroups v2 support on the Docker host.

## Known Limitations

- Cluster state is persisted in `~/.hind/clusters/<cluster-name>/`
- Port conflicts may occur when running multiple clusters simultaneously

## Development

```bash
# Build the CLI
make build

# Run tests
make test

# Format and vet code
go fmt ./...
go vet ./...
```

## Contributing

See [CLAUDE.md](CLAUDE.md) for development guidelines and project structure.

## License

MIT
