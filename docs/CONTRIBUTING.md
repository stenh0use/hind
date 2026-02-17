# Contributing to hind

Thank you for your interest in contributing to hind! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker or compatible container runtime
- Make

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/hind.git
cd hind

# Build the CLI
make hind-cli

# Verify installation
./bin/hind version
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feat/feature-name
```

### 2. Development Cycle

```bash
# Make your changes

# Build the CLI
make hind-cli

# Run tests
make test

# Test manually
./bin/hind start --profile=test
./bin/hind get test
./bin/hind rm test
```

### 3. Code Quality Checks

Before committing, ensure your code passes all quality checks:

```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# Run tests
go test ./...

# Or use the convenience target
make test
```

### 4. Commit Your Changes

Follow conventional commit format:

```bash
git add .
git commit -m "feat: add support for new feature"
```

Commit message format:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

## Implementation Checklist

When implementing a feature, follow this checklist:

- [ ] Understand the HashiCorp service requirements
- [ ] Design Go package structure if needed
- [ ] Keep track of the feature implementation details in a plan file (e.g., `features/feature.plan`)
- [ ] Write tests first (TDD approach)
- [ ] Implement minimal code to pass tests
- [ ] Run quality checks (`go fmt`, `go vet`, `go test`)
- [ ] Test CLI integration manually
- [ ] Update documentation when changes are made
- [ ] **Update CLAUDE.md if you:**
  - Add/remove CLI commands
  - Change package structure or responsibilities
  - Add new workflows or development patterns
  - Modify build processes or Makefile targets

## Adding New Features

### Adding a New CLI Command

1. Create command file:
```bash
touch pkg/cmd/hind/newcommand/newcommand.go
```

2. Implement cobra command structure following existing patterns

3. Add to root command in `pkg/cmd/hind/root.go`

4. Write tests:
```bash
touch pkg/cmd/hind/newcommand/newcommand_test.go
```

5. Test the command:
```bash
go build -o bin/hind && ./bin/hind newcommand --help
```

### Adding Docker Image Support

1. Create Dockerfile:
```bash
mkdir -p pkg/build/image/files/nodes/newservice
touch pkg/build/image/files/nodes/newservice/Dockerfile
```

2. Add build logic in `pkg/build/image/`

3. Add image kind to `pkg/build/release/`

4. Update cluster manager to handle new service in `pkg/cluster/`

5. Test integration

## Testing

See [TESTING.md](./TESTING.md) for detailed testing guidelines.

Quick reference:
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## Code Style

See [STYLE_GUIDE.md](./STYLE_GUIDE.md) for detailed style guidelines.

Key points:
- Use `gofmt` and `go vet`
- Handle errors appropriately
- Keep packages focused
- Write meaningful comments (explain WHY, not WHAT)
- Follow Go idioms and conventions

## Pull Requests

1. Ensure all tests pass
2. Update documentation as needed
3. Add a clear PR description explaining:
   - What problem does this solve?
   - What changes were made?
   - How was it tested?
4. Link related issues if applicable

## Getting Help

- Check existing documentation in `docs/`
- Review similar implementations in the codebase
- Open an issue for discussion
- Join our community channels (if available)

## Project Structure

See [CLAUDE.md](../CLAUDE.md) for detailed project structure and package responsibilities.

## Common Development Commands

```bash
# Full development cycle
make hind-cli                           # Build CLI
./bin/hind version                      # Test basic functionality
./bin/hind start --profile=test         # Test cluster creation
./bin/hind get test                     # Get cluster details
./bin/hind list                         # List all clusters
./bin/hind rm test                      # Clean up test cluster

# Code quality
go mod tidy && go fmt ./... && go vet ./... && go test ./...

# Or use the Makefile target
make test                               # Runs fmt, vet, and test

# Debug builds
go build -race -o bin/hind              # Race condition detection
go build -ldflags="-s -w" -o bin/hind   # Optimized binary
```

## Need Help?

If you have questions or run into issues, please see [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) or open an issue.
