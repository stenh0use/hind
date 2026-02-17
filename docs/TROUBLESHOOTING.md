# Troubleshooting Guide

This guide covers common issues and debugging techniques for hind.

## Quick Diagnostics

```bash
# Check hind version
./bin/hind version

# List all clusters
./bin/hind list

# Get details about a specific cluster
./bin/hind get <cluster-name>

# Check Docker connectivity
docker ps
docker info
```

## Common Issues

### Cluster Won't Start

**Symptoms:** `hind start` fails or hangs

**Diagnosis:**
```bash
# Check if Docker is running
docker ps

# Check for existing containers with same name
docker ps -a | grep hind

# Check Docker networks
docker network ls | grep hind

# Check Docker logs
docker logs hind.<cluster-name>.nomad.01
```

**Solutions:**
1. Ensure Docker daemon is running
2. Remove conflicting containers: `./bin/hind rm <cluster-name>`
3. Check Docker resources (CPU, memory, disk space)
4. Try with `--verbose` flag for more details

### Container Name Conflicts

**Symptoms:** Error about container already exists

**Problem:** Container names must follow pattern: `hind.<cluster-name>.<service>.<number>`

**Solution:**
```bash
# List all hind containers
docker ps -a | grep hind

# Remove old cluster
./bin/hind rm <cluster-name>

# Or manually remove specific container
docker rm -f hind.<cluster-name>.<service>.<number>
```

### Network Cleanup Issues

**Symptoms:** "Network in use" error when deleting cluster

**Problem:** Networks won't delete if containers still reference them

**Solution:**
```bash
# Always delete containers BEFORE networks
./bin/hind rm <cluster-name>

# Check which containers are using the network
docker network inspect hind.<cluster-name>

# Remove containers first
docker rm -f $(docker ps -aq --filter "network=hind.<cluster-name>")

# Then remove network
docker network rm hind.<cluster-name>
```

### Image Build Failures

**Symptoms:** `hind build` fails or produces unexpected results

**Diagnosis:**
```bash
# Check Docker build output
./bin/hind build nomad --verbose

# List current images
docker images | grep hind

# Check base image availability
docker pull hashicorp/nomad:latest
```

**Solutions:**
```bash
# Clear Docker cache and rebuild
docker build --no-cache -t hind-nomad:latest .

# Verify Dockerfile syntax
docker build -f pkg/build/image/files/nodes/nomad/Dockerfile .

# Check disk space
df -h
docker system df
```

### Test Clusters Not Cleaning Up

**Symptoms:** Test containers remain after tests complete

**Problem:** Tests didn't run cleanup code or crashed before cleanup

**Solution:**
```bash
# List all test clusters
docker ps -a | grep hind

# Clean up all hind containers
docker rm -f $(docker ps -aq --filter "name=hind")

# Clean up all hind networks
docker network rm $(docker network ls --filter "name=hind" -q)

# Use unique cluster names in tests to avoid conflicts
# Example: clusterName := fmt.Sprintf("test-%d", time.Now().Unix())
```

## Debugging Techniques

### Viewing Container Logs

```bash
# View logs for specific container (use actual container names)
docker logs hind.<cluster-name>.nomad.01
docker logs hind.<cluster-name>.consul.01

# Follow logs in real-time
docker logs -f hind.<cluster-name>.nomad.01

# View last 50 lines
docker logs --tail 50 hind.<cluster-name>.nomad.01

# Include timestamps
docker logs -t hind.<cluster-name>.nomad.01
```

### Inspecting Containers

```bash
# View container details
docker inspect hind.<cluster-name>.nomad.01

# Check container IP address
docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' hind.<cluster-name>.nomad.01

# Check environment variables
docker inspect -f '{{.Config.Env}}' hind.<cluster-name>.nomad.01
```

### Network Debugging

```bash
# List all networks
docker network ls

# Inspect specific network
docker network inspect hind.<cluster-name>

# Check which containers are connected
docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' hind.<cluster-name>

# Test connectivity between containers
docker exec hind.<cluster-name>.nomad.01 ping hind.<cluster-name>.consul.01
```

### CLI Debugging

```bash
# Run with verbose output
./bin/hind start --verbose --profile=debug

# Debug with race detection
go build -race -o bin/hind
./bin/hind start

# Enable Docker CLI debug mode
export DOCKER_BUILDKIT=0  # Disable BuildKit for more verbose output
./bin/hind build nomad
```

### Go Debugging

```bash
# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run specific test with debug info
go test -v ./pkg/cluster -run TestClusterManager_Create

# Profile CPU usage
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## Performance Issues

### Slow Cluster Startup

**Check:**
1. Docker resource allocation (increase CPU/memory if needed)
2. Number of concurrent image pulls
3. Local disk I/O performance
4. Network connectivity for image pulls

**Solutions:**
```bash
# Pre-pull images
docker pull hashicorp/nomad:latest
docker pull hashicorp/consul:latest

# Build images locally
./bin/hind build all

# Check Docker resource usage
docker stats
```

### High Memory Usage

**Check:**
```bash
# View container resource usage
docker stats

# Check Go memory usage
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

**Solutions:**
1. Reduce number of client nodes
2. Limit container memory in Docker settings
3. Profile Go code for memory leaks

## Docker-Specific Issues

### Permission Denied

**Symptoms:** Cannot connect to Docker daemon

**Solutions:**
```bash
# Ensure Docker daemon is running
sudo systemctl start docker  # Linux
open -a Docker              # macOS

# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo (not recommended for development)
sudo ./bin/hind start
```

### Disk Space Issues

**Check:**
```bash
# Check Docker disk usage
docker system df

# Check filesystem
df -h
```

**Solutions:**
```bash
# Clean up unused Docker resources
docker system prune

# Clean up unused images
docker image prune -a

# Clean up unused volumes
docker volume prune

# Remove all hind-specific resources
docker rm -f $(docker ps -aq --filter "name=hind")
docker network rm $(docker network ls --filter "name=hind" -q)
```

## Getting Help

If you're still experiencing issues:

1. Check [CONTRIBUTING.md](./CONTRIBUTING.md) for development guidelines
2. Review [CLAUDE.md](../CLAUDE.md) for architecture details
3. Search existing GitHub issues
4. Open a new issue with:
   - hind version (`./bin/hind version`)
   - Docker version (`docker --version`)
   - OS and architecture
   - Full error message
   - Steps to reproduce
   - Relevant logs

## Useful Docker Commands Reference

```bash
# List all containers (running and stopped)
docker ps -a

# List all networks
docker network ls

# List all images
docker images

# Remove specific container
docker rm -f <container-name>

# Remove specific network
docker network rm <network-name>

# Remove specific image
docker rmi <image-name>

# View container logs
docker logs <container-name>

# Execute command in running container
docker exec -it <container-name> /bin/sh

# Inspect container details
docker inspect <container-name>

# View resource usage
docker stats
```
