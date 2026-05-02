# Work Items

Active queue only (assigned or in-flight).

| ID | Description | Assigned role | Status | Blockers |
|----|-------------|---------------|--------|----------|
| BL-012 | Preserve architecture patterns during refactors | team-lead | done | None (closure based on archived audit + preservation guidance confirmed in active workstream reviews) |
| BL-013 | Define migration requirements from `internal/docker` to `pkg/provider` in image builds | staff-engineer | done | None (discovery/spec complete; canonical spec: `.claude/team/hind/spec-BL-013.md`) |
| BL-014 | Define release versioning requirements with discoverable versions | staff-engineer | done | None (discovery/spec complete; canonical spec: `.claude/team/hind/spec-BL-014.md`) |
| BL-015 | Audit feature specs versus implementation status | team-lead | done | None (audit complete; canonical spec: `.claude/team/hind/spec-BL-015.md`; follow-up backlog items BL-016..BL-020 created) |
| BL-016 | Close `hind-start.feature` behavior gaps | engineer | done | None (implementation complete, merged to refactor-cleanup) |
| BL-017 | Close `hind-stop.feature` behavior gaps (force/verbose/partial failure/idempotent) | engineer | done | None (implementation complete, merged to refactor-cleanup) |
| BL-018 | Close `hind-build.feature` version/dependency messaging gaps | engineer | done | None (staff plan approved, implementation complete, staff review approved, QA no-findings) |
| BL-019 | Enforce `default-cluster.feature` profile-selection contracts | staff-engineer | done | None (implementation complete, merged to refactor-cleanup) |
| BL-020 | Normalize and implement `hind-releases.feature` behavior | staff-engineer | done | None (implementation complete, merged to refactor-cleanup at 5f62b20) |
