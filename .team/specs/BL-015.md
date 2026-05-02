# BL-015 — Feature spec vs implementation audit

Date: 2026-04-30
Scope: `hind-releases.feature`, `hind-build.feature`, `default-cluster.feature`, `hind-start.feature`, `hind-stop.feature`

## Status matrix

- `hind-start.feature`: **partially implemented**
- `hind-stop.feature`: **partially implemented**
- `hind-build.feature`: **partially implemented**
- `default-cluster.feature`: **partially implemented**
- `hind-releases.feature`: **not implemented**

## Evidence summary

### `hind-stop.feature` — partially implemented
Implemented:
- default/positional/explicit cluster name selection behavior (with active-cluster fallback)
- stop flow with success message and preserved config semantics
- timeout flag exists
- cluster-not-found error path exists

Gaps:
- no `--force` flag behavior
- no `--verbose` flag behavior/log sequence
- no partial-stop warning/success semantics for per-container failures
- no explicit already-stopped idempotent success message contract
- no explicit unhealthy-container handling message contract

### `default-cluster.feature` — partially implemented
Implemented:
- successful `hind start` sets active cluster
- `hind set profile [name]` command exists

Gaps:
- `hind set profile [name]` does not verify cluster existence before setting active cluster
- active cluster reset behavior references `hind rm`; actual command surface is `hind rm`, and reset semantics need explicit spec alignment
- failure message contract for non-existent profile is not enforced
