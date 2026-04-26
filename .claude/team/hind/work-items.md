# Work Items

| ID | Description | Assigned | Status | Blockers |
|----|-------------|----------|--------|----------|
| RE-001 | Repository-wide Go quality review and prioritized improvement backlog | team-lead, staff-engineer, qa-engineer | Completed | None |
| BL-001 | Prevent nil-pointer panic in cluster state retrieval (`hind get`/`hind list`) | engineer-A | Completed | None |
| BL-002 | Enforce path confinement (block traversal/root escape) | engineer-B | Completed | None |
| BL-003 | Load persisted cluster config consistently for read/stop operations | engineer-A | Completed | BL-001 |
| BL-004 | Fix inspect error propagation in stop/delete flows | engineer | Completed | BL-003 |
| BL-005 | Resolve `start --version` contract drift | engineer-C | Completed | None |
| BL-006 | Normalize status mapping (`exited`/`stopped`) in list aggregation | team-lead | Completed | BL-003 |
| BL-007 | Correct `hind get` status/ports rendering | engineer | Completed | BL-001 |
| BL-008 | Make first-run `hind list` return empty-state success | engineer-C | Completed | BL-001 |
| BL-009 | Tighten provider/data-structure shaping and boundary clarity | unassigned | Todo | BL-003, BL-004, BL-006, BL-007 |
| BL-010 | Deepen behavioral/error-path test coverage in critical flows | unassigned | Todo | BL-001, BL-002, BL-003, BL-004 |
| BL-011 | Align docs/comments with runtime behavior | unassigned | Todo | BL-005, BL-006, BL-007 |
| BL-012 | Preserve architecture patterns during refactors | team-lead | Ongoing | None |
| BL-013 | Inject provider.Client into cluster.New() via parameter (remove hardcoded dockercli.New) | unassigned | Todo | None |
| BL-014 | Extract client node factory function to eliminate drift and fix numbering collision bug | unassigned | Todo | None |
| BL-015 | Populate or remove unused ContainerInfo fields (Ports, Network, Address, Image) | unassigned | Todo | BL-004, BL-006, BL-007 |
| BL-016 | Remove or complete dead CNI sub-package (pkg/cluster/cni) | unassigned | Todo | None |
| BL-026 | Fix `hind build` "path must be relative" error (BUG-009) | unassigned | Todo | None |
| BL-017 | Define provider.ContainerSpec to decouple dockercli from config.Node | unassigned | Todo | BL-013 |
| BL-018 | Move provider.ClusterInfo to pkg/cluster to clean layer boundary | unassigned | Todo | BL-015 |
| BL-019 | Fix minor correctness issues: unused ctx, wrong error text, Ports double-assign, bad image fallback, timer leak | unassigned | Todo | None |
| BL-020 | Define and implement image surface on provider.Client (BuildImage, TagExists, PullImage) | unassigned | Todo | BL-013 |
| BL-021 | Remove or implement dockercli/build.go stub (no-op BuildImage) | unassigned | Todo | BL-020 |
| BL-022 | Prune spurious fields from NetworkInfo; remove empty ContainerSummary/NetworkSummary types | unassigned | Todo | BL-015 |
| BL-023 | Add executor seam to internal/docker for unit testing BuildImage/TagExists/checkDependencies | unassigned | Todo | None |
| BL-024 | Harden metadata file path in build/image: use filepath.Join, extract constant, add test | unassigned | Todo | None |
| BL-025 | Normalize container status in dockercli provider (single source of truth for exited→stopped) | unassigned | Todo | BL-013 |
