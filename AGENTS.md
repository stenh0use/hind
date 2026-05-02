# hind Claude Guide

Use this file as fast project context for working in `hind`.

## Project at a glance

`hind` is a Go CLI (Cobra) for running HashiCorp components locally in Docker for development/testing.

## Core commands

```bash
# Build CLI
make hind-cli
# or
go build -o bin/hind

# Build images
./bin/hind build all
./bin/hind build nomad

# Cluster lifecycle
./bin/hind start [cluster-name]
./bin/hind start --clients=3
./bin/hind list
./bin/hind get <cluster-name>
./bin/hind stop [cluster-name]
./bin/hind rm <cluster-name>

# Quality checks
make test
```

## Required workflow

- After code changes, run `make test`.
- If CLI behavior changed, also validate manually (for example: `make hind-cli && ./bin/hind --help`).

## Architecture map

- `cmd/hind/` — CLI entrypoint.
- `pkg/cmd/hind/` — Cobra commands and formatting.
- `pkg/cluster/` — cluster lifecycle/orchestration.
- `pkg/provider/` — container runtime abstraction (`dockercli` implementation).
- `pkg/build/image/` — image definitions/build logic.
- `pkg/build/release/` — service versions/metadata.
- `pkg/config/` — config types.
- `pkg/file/` — file/path utilities.

## High-signal rules

- Container names follow: `hind.<cluster-name>.<service>.<number>`.
- In cluster/business logic, go through `pkg/provider` interfaces; avoid direct Docker command usage there.
- Delete containers before deleting networks.
- Do not hardcode HashiCorp versions in cluster code; use `pkg/build/release`.
- Tests creating clusters should clean up (`defer cluster.Delete(ctx)`) and use unique cluster names.

## References

- `docs/CONTRIBUTING.md` — workflow and implementation checklist.
- `docs/STYLE_GUIDE.md` — style and conventions.
- `docs/TESTING.md` — testing patterns.
- `docs/TROUBLESHOOTING.md` — debugging guidance.
