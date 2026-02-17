# HIND CLI Package - Development Guide

## Package Overview

The `pkg/cmd` package contains all CLI command implementations for the hind tool, organized around the Cobra framework.

**Package Structure:**
```
pkg/cmd/
├── iostreams.go           # IO abstraction for testable output
├── logger.go              # Logger factory and configuration
├── hind/                  # Root and subcommands
│   ├── root.go           # Root command (hind)
│   ├── build/            # Build command for Docker images
│   ├── get/              # Get command for cluster details
│   ├── list/             # List command for all clusters
│   ├── rm/               # Remove command to delete clusters
│   ├── start/            # Start command to create/start clusters
│   ├── stop/             # Stop command to stop clusters
│   ├── set/              # Set command group for configuration
│   ├── version/          # Version command
│   └── format/           # Shared formatting utilities
```

**Key Principles:**
- **Separation of Concerns**: CLI layer handles only user interaction, delegates business logic to `pkg/cluster`, `pkg/build`
- **Testability**: Commands accept dependencies (logger, IO streams) rather than creating them
- **Consistency**: All commands follow the same structural patterns

---

## Command Structure Patterns

### Standard Command Signature

All commands use this signature:
```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command
```

### Command Types

**Leaf Command** (performs actual work):
```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List all hind clusters",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd.Context(), logger, streams)
        },
    }
    return cmd
}

func runE(ctx context.Context, logger *log.Logger, streams IOStreams) error {
    // Implementation here
}
```

**Group Command** (organizes subcommands):
```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "set",
        Short: "Set hind configuration options",
    }
    cmd.AddCommand(newProfileCommand(logger, streams))
    return cmd
}
```

### RunE Separation Pattern

**Always separate RunE logic** from NewCommand for testability:

```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd.Context(), logger, streams, args)
        },
    }
    return cmd
}

// runE contains the actual command logic
func runE(ctx context.Context, logger *log.Logger, streams IOStreams, args []string) error {
    // 1. Parse arguments
    // 2. Validate input
    // 3. Call business logic (cluster.Manager, build.Builder)
    // 4. Format output
    return nil
}
```

---

## Dependency Injection

### Logger Injection

**Production usage**:
```go
logLevel := cmd.GetLogLevelFromEnv()  // Reads HIND_LOGLEVEL
logger := cmd.NewLogger(logLevel, "text")
```

**Test usage**:
```go
logger := &log.Logger{
    Handler: discard.New(),  // No-op handler for tests
    Level:   log.ErrorLevel,
}
```

**Logger levels**: DebugLevel (verbose), InfoLevel, WarnLevel, ErrorLevel

### IOStreams Injection

**IOStreams type** (`pkg/cmd/iostreams.go`):
```go
type IOStreams struct {
    In     io.Reader  // stdin
    Out    io.Writer  // stdout - program output
    ErrOut io.Writer  // stderr - status messages
}
```

**Production**: `streams := cmd.StandardIOStreams()`
**Test**: Capture with `bytes.Buffer` for output verification

---

## IO Guidelines

### Stream Usage

- **streams.Out** - Program output (parseable, machine-readable)
  - Structured data, tables, JSON
  - Content users might pipe or parse

- **streams.ErrOut** - Status messages (human-readable)
  - Progress updates, completion messages
  - Warnings that don't fail the command

- **logger** - Internal logging (controlled by log level)
  - Debug information (requires --verbose)
  - Info/Warn for internal state
  - Never use logger.Error() in commands - return errors instead

### Output Rules

**DO:**
- ✅ Use `streams.Out` for data users might pipe or parse
- ✅ Use `streams.ErrOut` for status messages
- ✅ Use `logger` for debug/verbose information
- ✅ Use tabwriter for aligned columns

**DON'T:**
- ❌ Use `fmt.Println()` or `fmt.Printf()` directly
- ❌ Mix program output and status messages on stdout
- ❌ Use `logger.Error()` to report command failures - return errors
- ❌ Write to `os.Stdout` or `os.Stderr` directly

---

## Flag Management

### Flagpole Pattern (4+ flags)

```go
type flagpole struct {
    hindVersion string
    timeout     time.Duration
    clients     int
}

func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    flags := &flagpole{}

    cmd := &cobra.Command{
        Use:   "start [cluster-name]",
        Short: "Start or create a hind cluster",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd, cmd.Context(), logger, streams, flags, args)
        },
    }

    cmd.Flags().StringVar(&flags.hindVersion, "version", "latest", "Hind version to use")
    cmd.Flags().DurationVar(&flags.timeout, "timeout", DefaultStartTimeout, "Timeout for startup")
    cmd.Flags().IntVar(&flags.clients, "clients", 1, "Number of client nodes")

    return cmd
}
```

### Simple Pattern (0-3 flags)

```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    var timeout time.Duration

    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd.Context(), logger, streams, timeout)
        },
    }

    cmd.Flags().DurationVar(&timeout, "timeout", DefaultTimeout, "Operation timeout")
    return cmd
}
```

### Flag Conventions

- Use lowercase with hyphens: `--cluster-name`, `--timeout`
- Always provide sensible defaults
- Use constants for default values: `DefaultStartTimeout`
- Check explicit vs default with `cmd.Flags().Changed("flag-name")`

---

## Error Handling

### Error Wrapping

**Always wrap business logic errors** with user-facing context:

```go
func runE(cmd *cobra.Command, ctx context.Context, logger *log.Logger,
          streams IOStreams, args []string) error {
    mgr, err := cluster.New(logger, clusterName)
    if err != nil {
        return fmt.Errorf("failed to initialize cluster manager: %w", err)
    }

    if err := mgr.Start(ctx); err != nil {
        return fmt.Errorf("failed to start cluster %q: %w", clusterName, err)
    }

    return nil
}
```

### Typed Errors

```go
import "errors"

func runE(...) error {
    if err := mgr.Delete(ctx); err != nil {
        var notFoundErr *cluster.NotFoundError
        if errors.As(err, &notFoundErr) {
            logger.Warnf("Cluster %q not found, nothing to delete", notFoundErr.Name)
            return nil
        }
        return fmt.Errorf("failed to delete cluster: %w", err)
    }
    return nil
}
```

### Error Conventions

- ✅ Wrap errors with `fmt.Errorf("context: %w", err)`
- ✅ Include relevant identifiers in messages (cluster names, etc.)
- ✅ Return errors from runE - let app layer format them
- ❌ Don't log errors with logger.Error() - return them instead
- ❌ Don't panic or call os.Exit() in command code

---

## Testing Patterns

### Command Construction Tests

```go
func TestNewCommand(t *testing.T) {
    logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
    streams := IOStreams{In: strings.NewReader(""), Out: io.Discard, ErrOut: io.Discard}

    cmd := NewCommand(logger, streams)

    if cmd == nil {
        t.Fatal("NewCommand() returned nil")
    }

    if cmd.Use != "list" {
        t.Errorf("Expected Use to be 'list', got '%s'", cmd.Use)
    }
}
```

### Table-Driven Tests

```go
func TestCommandArgs(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        args      []string
        wantError bool
    }{
        {"no args is valid", []string{}, false},
        {"one arg is valid", []string{"dev"}, false},
        {"two args is invalid", []string{"dev", "extra"}, true},
    }

    for _, tt := range tests {
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // Test implementation
        })
    }
}
```

### Output Verification

```go
func TestListCommand_Output(t *testing.T) {
    var buf bytes.Buffer
    streams := IOStreams{Out: &buf, ErrOut: io.Discard}

    cmd := NewCommand(logger, streams)
    cmd.SetArgs([]string{})

    if err := cmd.Execute(); err != nil {
        t.Fatalf("Command execution failed: %v", err)
    }

    output := buf.String()
    if !strings.Contains(output, "NAME") {
        t.Error("Expected header 'NAME' in output")
    }
}
```

### Parallel Tests

Use `t.Parallel()` for faster test runs:
- ✅ Pure logic tests (no shared state)
- ✅ Tests that don't modify environment variables
- ❌ Tests that modify global state
- ❌ Integration tests with real Docker operations

**CRITICAL**: Always capture range variables in parallel tests:
```go
for _, tt := range tests {
    tt := tt  // Required for parallel tests
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // Test logic
    })
}
```

---

## Active Cluster Management

### Pattern

```go
func runE(cmd *cobra.Command, ctx context.Context, logger *log.Logger,
          streams IOStreams, args []string) error {
    var clusterName string

    if len(args) > 0 {
        clusterName = args[0]
    } else {
        activeCluster, err := cluster.GetActiveCluster()
        if err != nil {
            logger.Debugf("Failed to get active cluster: %v", err)
        }

        if activeCluster == "" {
            clusterName = "default"
            logger.Debugf("No active cluster, using default")
        } else {
            clusterName = activeCluster
            logger.Debugf("Using active cluster: %s", clusterName)
        }
    }

    // ... rest of command logic
}
```

### Setting Active Cluster

Commands that create/start clusters should set the active cluster:
```go
if err := cluster.SetActiveCluster(clusterName); err != nil {
    logger.Warnf("Failed to set active cluster: %v", err)
}
```

### Clearing Active Cluster

Commands that delete clusters should clear if deleting the active cluster:
```go
activeCluster, _ := cluster.GetActiveCluster()
if activeCluster == clusterName {
    if err := cluster.ClearActiveCluster(); err != nil {
        logger.Warnf("Failed to clear active cluster: %v", err)
    }
}
```

---

## Implementation Checklist

When implementing a new command:

- [ ] **Signature**: `func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command`
- [ ] **Flagpole**: Use flagpole struct if command has 4+ flags
- [ ] **RunE separation**: Extract logic to separate `runE()` function
- [ ] **IO streams**: All output through `streams.Out` or `streams.ErrOut`
- [ ] **Error wrapping**: Wrap business logic errors with context
- [ ] **Active cluster**: Handle active cluster logic (if applicable)
- [ ] **Args validation**: Set appropriate `Args` validator (`NoArgs`, `ExactArgs`, etc.)
- [ ] **Documentation**: Provide clear `Short` and `Long` descriptions
- [ ] **Test file**: Create `<command>_test.go` with table-driven tests
- [ ] **Output tests**: Verify output format using buffer streams
- [ ] **Flag tests**: Verify flags exist and have correct defaults
- [ ] **Parallel tests**: Add `t.Parallel()` where safe, capture range variables

---

## Reference Examples

For complete working examples, see:
- **Simple command**: [pkg/cmd/hind/list/list.go](../hind/list/list.go)
- **Complex command with flags**: [pkg/cmd/hind/start/start.go](../hind/start/start.go)
- **Group command**: [pkg/cmd/hind/set/set.go](../hind/set/set.go)
- **Testing patterns**: [pkg/cmd/hind/list/list_test.go](../hind/list/list_test.go)

---

## Quick Reference Templates

### Minimal Command
```go
func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "command-name",
        Short: "Brief description",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd.Context(), logger, streams)
        },
    }
    return cmd
}

func runE(ctx context.Context, logger *log.Logger, streams IOStreams) error {
    // Implementation
    return nil
}
```

### Command with Flags
```go
type flagpole struct {
    flag1 string
    flag2 int
}

func NewCommand(logger *log.Logger, streams IOStreams) *cobra.Command {
    flags := &flagpole{}

    cmd := &cobra.Command{
        Use:   "command-name",
        Short: "Brief description",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runE(cmd, cmd.Context(), logger, streams, flags, args)
        },
    }

    cmd.Flags().StringVar(&flags.flag1, "flag1", "default", "description")
    cmd.Flags().IntVar(&flags.flag2, "flag2", 0, "description")

    return cmd
}
```

---

This guide documents hind-specific patterns. For general Go/Cobra best practices, refer to the official documentation.
