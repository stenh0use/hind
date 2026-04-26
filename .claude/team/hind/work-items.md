# Work Items

| ID | Description | Assigned | Status | Blockers |
|----|-------------|----------|--------|----------|
| RE-001 | Repository-wide Go quality review and prioritized improvement backlog | team-lead, staff-engineer, qa-engineer | Completed | None |
| BL-001 | Prevent nil-pointer panic in cluster state retrieval (`hind get`/`hind list`) | engineer-A | In Progress | None |
| BL-002 | Enforce path confinement (block traversal/root escape) | engineer-B | In Progress | None |
| BL-003 | Load persisted cluster config consistently for read/stop operations | unassigned | Todo | BL-001 |
| BL-004 | Fix inspect error propagation in stop/delete flows | unassigned | Todo | BL-003 |
| BL-005 | Resolve `start --version` contract drift | engineer-C | In Progress | None |
| BL-006 | Normalize status mapping (`exited`/`stopped`) in list aggregation | unassigned | Todo | BL-003 |
| BL-007 | Correct `hind get` status/ports rendering | unassigned | Todo | BL-001 |
| BL-008 | Make first-run `hind list` return empty-state success | unassigned | Todo | BL-001 |
| BL-009 | Tighten provider/data-structure shaping and boundary clarity | unassigned | Todo | BL-003, BL-004, BL-006, BL-007 |
| BL-010 | Deepen behavioral/error-path test coverage in critical flows | unassigned | Todo | BL-001, BL-002, BL-003, BL-004 |
| BL-011 | Align docs/comments with runtime behavior | unassigned | Todo | BL-005, BL-006, BL-007 |
| BL-012 | Preserve architecture patterns during refactors | team-lead | Ongoing | None |
