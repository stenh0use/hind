# Bugs

## BUG-001
- Description: `hind get`/`hind list` can panic when the cluster network is missing because `Manager.Get` dereferences a nil network pointer (severity: high)
- Repro steps or triggering condition:
  1. Use a cluster name with no existing Docker network (for example, a non-existent cluster)
  2. Run `hind get <cluster-name>` or trigger `Manager.Get` via `hind list`
- Observed result: process can crash with nil pointer dereference from `state.Network = *networkInfo`
- Expected result: command should return a controlled not-found/error response without panicking
- Status: open
- Linked work item: RE-001

## BUG-002
- Description: `hind stop` does not load persisted cluster config and may skip scaled client nodes (severity: high)
- Repro steps or triggering condition:
  1. Create/start a cluster with more than one client (e.g., `hind start demo --clients=3`)
  2. Run `hind stop demo`
- Observed result: stop iterates default in-memory config (1 client) and can leave additional client containers running
- Expected result: stop should load current cluster config from disk and stop all configured nodes
- Status: open
- Linked work item: RE-001

## BUG-003
- Description: container/network inspect errors are swallowed in stop/delete flows due conditional ordering and weak error propagation (severity: high)
- Repro steps or triggering condition:
  1. Trigger provider inspect failures (e.g., daemon permission/connectivity issues)
  2. Run `hind stop <cluster>` or `hind rm <cluster>`
- Observed result: inspect errors can be treated as "not found" and skipped, and delete may continue/report success despite provider failures
- Expected result: inspect errors should be returned to callers (except explicit not-found semantics)
- Status: open
- Linked work item: RE-001

## BUG-004
- Description: `hind list` can misclassify stopped clusters because it expects status `"stopped"` while Docker inspect returns `"exited"` (severity: medium)
- Repro steps or triggering condition:
  1. Stop a cluster so containers are in Docker `exited` state
  2. Run `hind list`
- Observed result: status may show `partial` instead of `stopped`
- Expected result: fully stopped cluster should be classified as `stopped`
- Status: open
- Linked work item: RE-001

## BUG-005
- Description: `hind get` renders inaccurate/garbled output (severity: medium)
- Repro steps or triggering condition:
  1. Run `hind get <cluster-name>` for any cluster with containers
- Observed result: status line is hardcoded to `created`; ports use `%s` with `[]string`, producing `%!s(...)` formatting artifacts
- Expected result: status should reflect actual state; ports should be formatted human-readably
- Status: open
- Linked work item: RE-001

## BUG-006
- Description: `hind list` fails for first-time users when cluster config directory does not exist (severity: medium)
- Repro steps or triggering condition:
  1. Use a fresh HOME with no `~/.config/hind/cluster` directory
  2. Run `hind list`
- Observed result: command errors on directory read instead of returning empty list
- Expected result: command should succeed and print `No clusters found`
- Status: open
- Linked work item: RE-001

## BUG-007
- Description: file/path handling permits path traversal outside configured root (severity: medium)
- Repro steps or triggering condition:
  1. Provide path-like cluster names containing traversal segments (e.g., `../../...`)
  2. Invoke commands that persist/read cluster config paths
- Observed result: `validatePath` only checks emptiness and `resolvePath` can escape root boundaries
- Expected result: reject traversal/absolute escapes for user-controlled paths and enforce root confinement
- Status: open
- Linked work item: RE-001

