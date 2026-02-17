# Go Code Style Guide

## Code Style Principles

Go code in this project follows standard Go conventions and idioms. The goal is to write clear, maintainable code that other Go developers can easily understand.

## Automated Tools

Run these before every commit:

```bash
go fmt ./...      # Format all Go code
go vet ./...      # Lint code for common mistakes
make test         # Run fmt, vet, and all tests
```

## General Rules

- **Follow Go conventions** - Use `gofmt`, `golint`, and `go vet`
- **Package organization** - Keep packages focused and well-named
- **Error handling** - Always handle errors appropriately
- **No global state** - Use dependency injection patterns
- **Interfaces over structs** - Keep interfaces small and focused
- **120 char line limit** - Keep code readable

## Comment Guidelines

### The Golden Rule: Comments Explain WHY, Not WHAT

Code should be self-explanatory. Comments should provide context that isn't obvious from reading the code itself.

### When to DELETE Comments

Ask yourself:

- **Is the code self-explanatory?** If yes, DELETE the comment.
- **Does the comment just restate what the code does?** If yes, DELETE the comment.
- **Is the comment out of date or incorrect?** If yes, UPDATE or DELETE the comment.

### Examples of Comments to DELETE

```go
// Create a new slice
nodes := make([]string, 0)

// Set the status to running
cluster.Status = StatusRunning

// Return the error
return err

// Close the file
file.Close()
```

These comments add no value. The code clearly shows what's happening.

### Examples of Comments to KEEP

```go
// Regex must be compiled only once at package initialization to avoid performance degradation.
// Compiling on each validation call caused 40% CPU increase in production.
var containerNamePattern = regexp.MustCompile(`^hind\.[a-z0-9-]+\.[a-z]+\.\d{2}$`)

// Thread safety: this method is called concurrently from multiple goroutines.
// The mutex prevents race conditions when multiple workers attempt to create
// the same Docker network simultaneously.
c.mu.Lock()
defer c.mu.Unlock()
if !c.networksCreated[name] {
    if err := c.provider.CreateNetwork(ctx, name); err != nil {
        return err
    }
    c.networksCreated[name] = true
}
```

These comments provide valuable context:
- Performance considerations
- Concurrency concerns
- Non-obvious design decisions
- Business logic rationale

### When Comments Are Valuable

Comments should:
- **Provide additional context** that isn't obvious from the code itself
- **Explain WHY** a particular approach was chosen
- **Warn about gotchas** or non-obvious behavior
- **Document complex algorithms** at a high level
- **For large code blocks**, allow readers to quickly grasp the intent without reading every line

Comments should NOT:
- Narrate the code line by line
- Repeat what's already clear from variable/function names
- State the obvious
