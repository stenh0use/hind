# Testing Guide

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run tests for a specific package
go test ./pkg/cluster/...

# Run specific test
go test ./pkg/cluster -run TestClusterManager_Create

# Verbose output
go test -v ./...
```

## Test Organization

- Place tests in `*_test.go` files alongside the code they test
- Use package-level test files: `pkg/cluster/manager_test.go`
- Keep test files focused on a single type or related functionality

## Go Testing Patterns

### Table-Driven Tests

Use table-driven tests for testing multiple scenarios:

```go
// pkg/cluster/manager_test.go
func TestClusterManager_Create(t *testing.T) {
    tests := []struct {
        name    string
        config  ClusterConfig
        want    error
        setup   func()
        cleanup func()
    }{
        {
            name: "creates single node cluster",
            config: ClusterConfig{
                Name: "test-cluster",
                Nodes: 1,
            },
            want: nil,
        },
        {
            name: "fails with invalid name",
            config: ClusterConfig{
                Name: "",
                Nodes: 1,
            },
            want: ErrInvalidClusterName,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setup != nil {
                tt.setup()
            }
            defer func() {
                if tt.cleanup != nil {
                    tt.cleanup()
                }
            }()

            manager := NewClusterManager()
            got := manager.Create(tt.config)

            if got != tt.want {
                t.Errorf("Create() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing Best Practices

1. **Use descriptive test names** - Test name should describe what's being tested
2. **Test one thing per test** - Keep tests focused and simple
3. **Use t.Run for subtests** - Better test organization and isolation
4. **Always clean up resources** - Use defer for cleanup
5. **Test error cases** - Don't just test the happy path
6. **Use table-driven tests** - For testing multiple scenarios efficiently
7. **Mock external dependencies** - Use interfaces for testability

### Test Cleanup

Always clean up resources created during tests:

```go
func TestCluster_Create(t *testing.T) {
    ctx := context.Background()
    cluster := NewCluster("test-cluster")

    // Create cluster
    err := cluster.Create(ctx)
    if err != nil {
        t.Fatalf("failed to create cluster: %v", err)
    }

    // Always clean up
    defer func() {
        if err := cluster.Delete(ctx); err != nil {
            t.Errorf("failed to cleanup cluster: %v", err)
        }
    }()

    // Run your tests...
}
```

### Integration Tests

For tests that interact with Docker or other external systems:

```go
func TestIntegration_ClusterLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Use unique names to avoid conflicts
    clusterName := fmt.Sprintf("test-%d", time.Now().Unix())

    // Test implementation...
}
```

Run integration tests:
```bash
# Run all tests including integration tests
go test ./...

# Skip integration tests (short mode)
go test -short ./...
```

## Mocking and Interfaces

Use interfaces to make code testable:

```go
// Define interface
type ContainerProvider interface {
    CreateContainer(ctx context.Context, config ContainerConfig) error
    DeleteContainer(ctx context.Context, name string) error
}

// Mock implementation for tests
type MockProvider struct {
    CreateContainerFunc func(ctx context.Context, config ContainerConfig) error
    DeleteContainerFunc func(ctx context.Context, name string) error
}

func (m *MockProvider) CreateContainer(ctx context.Context, config ContainerConfig) error {
    if m.CreateContainerFunc != nil {
        return m.CreateContainerFunc(ctx, config)
    }
    return nil
}

// Use in tests
func TestCluster_WithMockProvider(t *testing.T) {
    mock := &MockProvider{
        CreateContainerFunc: func(ctx context.Context, config ContainerConfig) error {
            return nil // Controlled behavior
        },
    }

    cluster := NewClusterWithProvider(mock)
    // Test cluster operations...
}
```

## Test Coverage

Check test coverage:

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Show coverage by function
go tool cover -func=coverage.out
```

Aim for:
- **80%+ coverage** for critical paths
- **100% coverage** for core business logic
- Focus on meaningful tests, not just coverage numbers
