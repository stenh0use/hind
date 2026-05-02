# pkg/cmd Claude Guide

Use this file for command-layer work in `pkg/cmd`.

## Scope

- `pkg/cmd` handles CLI interaction only.
- Delegate business logic to `pkg/cluster`, `pkg/build`, and related packages.

## Package layout

- `iostreams.go` — standard input/output abstraction.
- `logger.go` — logger setup.
- `hind/` — root command + subcommands.

## Command structure

- Standard constructor for command packages under `pkg/cmd/hind/*`:

```go
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command
```

- Keep constructor focused on wiring flags/args.
- Put command behavior in `runE(...)` for testability.
- Use Cobra arg validators (`cobra.NoArgs`, `cobra.ExactArgs`, etc.).

## Flags

- Use local vars for a few flags.
- Use a `flagpole` struct when a command has many flags (typically 4+).
- Prefer stable defaults and explicit flag descriptions.

## IO and logging rules

- `streams.Out`: program output users may parse.
- `streams.ErrOut`: status/progress messages.
- `logger`: debug/internal logs.
- Return errors from command logic; do not report command failures via `logger.Error()`.
- Avoid direct `fmt.Println` / `os.Stdout` / `os.Stderr` in commands.

## Error handling

- Wrap errors with context using `%w`.
- Include useful identifiers (cluster name, operation) in user-facing error context.

## Active cluster behavior

- For optional cluster args, prefer:
  1. explicit arg,
  2. active cluster from `cluster.GetActiveCluster()`,
  3. fallback default.
- Commands that select/create a cluster should set active cluster when appropriate.
- Removing the active cluster should clear it.

## Testing essentials

- Prefer table-driven tests.
- Use `t.Parallel()` only when safe.
- In parallel table tests, capture range var (`tt := tt`).
- Verify output by injecting `cmd.IOStreams` with `bytes.Buffer`.

## Canonical examples

- `pkg/cmd/hind/list/list.go`
- `pkg/cmd/hind/start/start.go`
- `pkg/cmd/hind/set/set.go`
- `pkg/cmd/hind/rm/rm.go`

## Validation

- Run `make test` after command-layer changes.
