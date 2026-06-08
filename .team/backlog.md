# Team Backlog

Closed items: `.team/done/`

Priority order is top-to-bottom within this table (highest first).

|  ID   | Title | Type | Priority | Size | Status | Source | Spec |
|-------|-------|------|----------|------|--------|--------|------|
| W-048 | Audit all mock test cases and reduce duplication | refactor | P2 | M | approved-spec | User | `/Users/james/dev/github/stenh0use/hind/.team/specs/0003-mocks-audit.md` |
| W-049 | Audit provider/dockercli naming consistency and options patterns | refactor | P2 | M | approved-spec | User | `/Users/james/dev/github/stenh0use/hind/.team/specs/0004-provider-dockercli-naming-audit.md` |
| W-031 | Add open subcommand to open the web ui of a component | feature | P3 | M | needs-spec | User | `W-031.md` |
| W-032 | Add login subcommand to exec into an interactive shell in a node | feature | P3 | M | needs-spec | User | `W-032.md` |
| W-030 | Build and publish releases to brew for macos install | feature | P3 | L | needs-spec | User | `W-030.md` |
| W-044 | Image digest pinning for cluster start | feature | P3 | M | needs-spec | Architecture review | — |
| W-025 | Publish container images to an OCI registry on version update | feature | P4 | XL | needs-spec | User | `W-025.md` |
| W-029 | Add ingress controller for routing traffic to the internal network | feature | P4 | XL | needs-spec | User | `W-029.md` |


Notes:
- still lots of duplication in cluster package eg. here: /Users/james/dev/github/stenh0use/hind/pkg/cluster/domain/helpers.go
- Need to review build package, is the abstraction still right, and file package
- cmd mocks/tests should go in testing _test files
- dockercli inspect/list struct naming normalization
- docker impl has too much business logic ingrained there, should be at the application layer.
- docker tmfs need cleaning up otherwise they fill the vm disk
- e2e test cases shouldn't need to clean up first if dynamic ports are used.
- update go / references to Go 1.26
- change commands to list alias ls, and remove alias rm
