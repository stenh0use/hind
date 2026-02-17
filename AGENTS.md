# Claude Code Assistant Instructions - Hashistack in Docker (hind)

## 🤖 AI Context & Project Overview

You are assisting with **hind** - a Go-based CLI tool that builds and runs different components from the HashiCorp ecosystem (the "Hashistack") in Docker containers. This project provides a quick playground for Nomad, Consul, and related services, similar to how `kind` works for Kubernetes.

### Key Project Components

- **CLI Tool**: `hind` binary built with Cobra framework
- **Docker Images**: Custom images for Nomad, Consul, and supporting services
- **Cluster Management**: Multi-node Nomad clusters with service discovery
- **Network Integration**: Optional support for CNI and Service Mesh

### Key Project Files

- `cmd/hind/` - CLI entry point and command structure
- `pkg/` - Core Go packages organized by functionality
- `Makefile` - Build and deployment automation

## 🎯 Primary Objectives

1. **Build reliable HashiCorp service containers** - Custom images optimized for development
2. **Provide simple cluster lifecycle management** - Easy up/down operations
3. **Enable multi-node testing scenarios** - Scalable client nodes
4. **Support advanced networking** - CNI integration
5. **Maintain Go best practices** - Clean, idiomatic Go code

## 🏛️ Architecture Decisions

**Why Docker CLI via provider abstraction instead of Docker SDK?**
- Better compatibility with existing Docker installations
- Simpler debugging (can replicate issues with docker commands)
- Matches kind's approach (proven pattern for local clusters)
- Easy to add alternative container runtimes (podman, etc.) later

**Why Cobra for CLI framework?**
- Industry standard for Go CLIs (kubectl, gh, docker use it)
- Built-in help generation and shell completion
- Easy subcommand management and flag parsing
- Excellent documentation and community support

**Why separate provider abstraction layer?**
- Allows future support for different container runtimes
- Makes testing easier (can mock container operations)
- Keeps Docker-specific logic isolated
- Follows dependency inversion principle

## ⚡ Quick Command Reference

```bash
# Build the CLI tool
make hind-cli

# Build Docker images
./bin/hind build all                    # Build all images
./bin/hind build nomad                  # Build specific image

# Cluster management
./bin/hind start                        # Start cluster (default profile)
./bin/hind start <cluster-name>         # Start with named profile
./bin/hind start --clients=3            # Start with 3 client nodes
./bin/hind list                         # List all clusters
./bin/hind get <cluster-name>           # Get cluster details
./bin/hind rm <cluster-name>            # Delete a cluster

# Go development commands
go build -o bin/hind                    # Build CLI
go test ./...                           # Run all tests
go mod tidy                             # Clean dependencies
go fmt ./...                            # Format code
go vet ./...                            # Lint code
make test                               # Run fmt, vet, and tests
```

## 🚨 CRITICAL RULES - NO EXCEPTIONS

### After Every Code Change

1. ✅ Run `make test ` - Format all Go code
2. ✅ Test CLI functionality manually if applicable
3. ✅ Never skip quality checks for "small changes"

### Go Code Style Mandates

- **Follow Go conventions** - Use `gofmt`, `golint`, and `go vet`
- **Package organization** - Keep packages focused and well-named
- **Error handling** - Always handle errors appropriately
- **No global state** - Use dependency injection patterns
- **Interfaces over structs** - Keep interfaces small and focused
- **120 char line limit** - Keep code readable
- **Comments explain WHY, not WHAT** - Code should be self-documenting (see [docs/STYLE_GUIDE.md](docs/STYLE_GUIDE.md))

## ⚠️ Common Pitfalls

**Container Naming:**
- ❌ Don't use arbitrary container names
- ✅ Always use the pattern: `hind.<cluster-name>.<service>.<number>`
- Example: `hind.default.nomad.01`, `hind.test.consul.01`

**Network Cleanup:**
- ❌ Networks won't delete if containers still reference them
- ✅ Always delete containers before deleting networks
- ✅ Use `./bin/hind delete <cluster>` to ensure proper cleanup order

**Image Building:**
- ❌ Don't assume cached layers are current
- ✅ Use `docker build --no-cache` if build behavior seems inconsistent
- ✅ Check base image digests in `pkg/build/image/` when debugging

**Provider Abstraction:**
- ❌ Don't call Docker commands directly in cluster code
- ✅ Always go through the `pkg/provider` interface
- ✅ This keeps the code testable and runtime-agnostic

**Configuration Management:**
- ❌ Don't hardcode HashiCorp versions in cluster code
- ✅ Always use `pkg/build/release/` for version management
- ✅ This ensures consistency across images and runtime

**Test Cleanup:**
- ❌ Don't leave test clusters running
- ✅ Always defer cleanup in tests: `defer cluster.Delete(ctx)`
- ✅ Use unique cluster names per test to avoid conflicts

**For detailed development guidelines, see:**
- [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - Development workflow and implementation checklist
- [docs/STYLE_GUIDE.md](docs/STYLE_GUIDE.md) - Code style guidelines
- [docs/TESTING.md](docs/TESTING.md) - Testing patterns and best practices
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Debugging guides

## 🏗️ Project Structure

```
hind/
├── cmd/hind/                      # CLI application entry point
│   ├── main.go                    # Main CLI entry
│   └── app/                       # Application setup and initialization
│
├── pkg/                           # Core Go packages
│   │
│   ├── cmd/hind/                  # Cobra CLI commands implementation
│   │   ├── root.go               # Root command setup, adds all subcommands
│   │   ├── build/                # Build command - builds Docker images
│   │   ├── start/                # Start command - creates/starts clusters
│   │   ├── get/                  # Get command - retrieves cluster details
│   │   ├── list/                 # List command - lists all clusters
│   │   ├── rm/                   # Delete command - removes clusters
│   │   ├── format/               # Format utilities for CLI output
│   │   └── version/              # Version command - displays version info
│   │
│   ├── build/                     # Image building and release management
│   │   ├── image/                # Docker image specifications and building
│   │   │                         # WHEN: Adding new HashiCorp service images
│   │   │                         # WHEN: Modifying image build configurations
│   │   └── release/              # Release version management for services
│   │                             # WHEN: Adding new HashiCorp version support
│   │                             # WHEN: Defining image metadata and versions
│   │
│   ├── cluster/                   # Cluster orchestration and lifecycle
│   │   ├── cluster.go            # Main cluster type and operations (Create, Start, Stop, Delete)
│   │   │                         # WHEN: Implementing cluster lifecycle features
│   │   ├── types.go              # Cluster type definitions and defaults
│   │   ├── cni/                  # Container Network Interface implementations
│   │   │   ├── cni.go           # CNI interface definition
│   │   │   ├── none/            # No CNI (basic Docker networking)
│   │   │   ├── cilium/          # Cilium CNI implementation
│   │   │   └── factory/         # CNI factory pattern for creating CNI instances
│   │   │                         # WHEN: Adding new CNI providers
│   │   │                         # WHEN: Implementing network policies
│   │   └── runtime/              # Runtime configuration and container orchestration
│   │                             # WHEN: Adding runtime-specific features
│   │
│   ├── provider/                  # Container provider abstraction layer
│   │   ├── provider.go           # Interface for container/network operations
│   │   │                         # WHEN: Adding support for new container runtimes
│   │   └── dockercli/            # Docker CLI implementation
│   │       ├── client.go         # Docker client wrapper
│   │       ├── container.go      # Container lifecycle operations
│   │       ├── network.go        # Network management
│   │       ├── image.go          # Image operations
│   │       └── build.go          # Image building
│   │                             # WHEN: Implementing Docker-specific features
│   │                             # WHEN: Adding new container operations
│   │
│   ├── config/                    # Configuration types and structures
│   │   └── config.go             # Cluster, Node, Network, Volume configs
│   │                             # WHEN: Adding new configuration options
│   │                             # WHEN: Defining node/cluster properties
│   │
│   └── file/                      # File system utilities
│       └── file.go               # File/directory operations, path management
│                                 # WHEN: Adding file I/O operations
│                                 # WHEN: Managing cluster state files
│
├── jobs/                          # Example Nomad job files for testing
│
└── features/                      # Feature definitions and planning documents
```

### Package Responsibilities Guide

**When adding NEW features, consider:**

- **CLI Commands** → `pkg/cmd/hind/<command>/` - User-facing commands
- **Image Changes** → `pkg/build/image/` - New services or image configurations
- **Cluster Logic** → `pkg/cluster/` - Cluster orchestration, lifecycle management
- **Networking** → `pkg/cluster/cni/` - CNI providers, network policies
- **Container Operations** → `pkg/provider/dockercli/` - Low-level container/network ops
- **Configuration** → `pkg/config/` - New config types, node properties
- **File Operations** → `pkg/file/` - State persistence, file management


## 🚀 Quick Start for Claude Code

When starting a session:

1. **Read this file first** for Go project context
2. **Check current branch** - Should be working on `feat/feat-name`
3. **Review recent commits** - Understand latest changes
4. **Run tests** - `go test ./...` to see current state
5. **Check CLI functionality** - `make hind-cli && ./bin/hind --help`

## 📌 Remember

- **Go conventions are mandatory** - `gofmt`, `go vet`, proper error handling
- **Test-driven development** - Write tests first when possible
- **Docker implications** - Consider container impact of changes
- **CLI usability** - Commands should be intuitive and well-documented
- **HashiCorp ecosystem** - Understand service interactions

---

_This document is optimized for Claude Code working on the hind Go CLI project. Always refer to current code structure and `features/*.feature` for authoritative requirements._
