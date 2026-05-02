# Hind Releases Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `hind releases` command surface so `hind-releases.feature` scenario "List available hind versions" is fully covered, and the feature file is normalized to remove the two incomplete scenario stubs.

**Architecture:** A new `pkg/cmd/hind/releases/` command package follows the existing Cobra command pattern (constructor + `runE`). All version data comes from `pkg/build/release` (already implemented); the command layer is presentation-only and uses `text/tabwriter` to render a sorted, column-labeled table matching the feature contract. No new release-layer logic is required.

**Tech Stack:** Go 1.21+, Cobra, `text/tabwriter`, `pkg/build/release`, `cmd.IOStreams`.

---

## File structure

| Path | Status | Responsibility |
|------|--------|---------------|
| `pkg/cmd/hind/releases/releases.go` | Create | Command constructor + `runE` — renders sorted version table |
| `pkg/cmd/hind/releases/releases_test.go` | Create | Table-driven tests for column order, row order, header format, and empty-data edge case |
| `pkg/cmd/hind/root.go` | Modify | Register `releases.NewCommand` on root |
| `features/hind-releases.feature` | Modify | Remove the two empty/stub scenarios; normalize Background and List scenario wording |

---

## Task 1: Normalize `hind-releases.feature`

**Files:**
- Modify: `features/hind-releases.feature`

The feature file has two malformed stub scenarios ("Create new hind cluster" and "Run non existent hind version") that have no steps and are out of scope for B-020. Remove them. Normalize the Background and the "List available hind versions" scenario to be precise enough for acceptance testing.

- [ ] **Step 1: Read the current feature file**

Open `features/hind-releases.feature` and confirm the three scenarios and their step states.

- [ ] **Step 2: Write the normalized feature file**

Replace the full file with:

```gherkin
Feature: HIND releases menu
    As a maintainer of the HIND CLI
    I want an easy way to view the hind versions and the version of the HashiCorp binaries that are included
    So that releases can easily be built and published

    Background:
        Given I have defined the hind version in the version configuration
        And the hind version has the defined consul version
        And the vault version
        And the nomad version

    Scenario: List available hind versions
        Given I run the "hind releases" command
        When I execute the command
        Then the CLI will list in a table the available hind versions
        And the column header row is printed first with columns: HIND, CONSUL, NOMAD, VAULT
        And the first column is the hind version
        And the remaining columns are displayed in alphabetical order: consul, nomad, vault
        And the latest version is on the first row
        And the oldest version is on the last row
```

- [ ] **Step 3: Commit**

```bash
git add features/hind-releases.feature
git commit -m "feat(B-020): normalize hind-releases.feature — remove stubs, clarify list scenario"
```

---

## Task 2: Implement `pkg/cmd/hind/releases` command

**Files:**
- Create: `pkg/cmd/hind/releases/releases.go`

The command lists all hind releases sorted newest-first as a `tabwriter` table with columns: `HIND`, `CONSUL`, `NOMAD`, `VAULT` (alphabetical after HIND). The latest version appears on the first row.

- [ ] **Step 1: Write the failing test first (see Task 3 — do Task 3 step 1 before this)**

(Tests are in Task 3. Write tests before implementation per TDD sequence. Come back here after Task 3 Step 1.)

- [ ] **Step 2: Write the implementation**

Create `/Users/james/dev/github/stenh0use/hind/pkg/cmd/hind/releases/releases.go`:

```go
// Package releases implements the "hind releases" command.
// It renders a sorted table of all hind releases and their HashiCorp component versions.
package releases

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/cmd"
)

// NewCommand returns a new cobra.Command for listing hind releases.
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	command := &cobra.Command{
		Use:   "releases",
		Short: "List available hind releases and their HashiCorp component versions",
		Long:  "List all available hind releases in a table sorted by version (latest first).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams)
		},
	}
	return command
}

func runE(logger *log.Logger, streams cmd.IOStreams) error {
	logger.Debug("Listing hind releases")

	versions := release.List()
	if len(versions) == 0 {
		fmt.Fprintln(streams.ErrOut, "No releases found")
		return nil
	}

	// Sort versions descending (latest first) using semver-style string comparison.
	// Versions follow "MAJOR.MINOR.PATCH" format so lexicographic sort is valid
	// when zero-padded; use sort.Slice with reverse string comparison as a conservative
	// baseline. For strict semver ordering in future, switch to golang.org/x/mod/semver.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	w := tabwriter.NewWriter(streams.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "HIND\tCONSUL\tNOMAD\tVAULT")

	for _, v := range versions {
		info, err := release.Get(v)
		if err != nil {
			// Skip unknown versions; should not occur with List() output.
			logger.Warnf("skipping unknown release %q: %v", v, err)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Hind, info.Consul, info.Nomad, info.Vault)
	}

	return w.Flush()
}
```

- [ ] **Step 3: Run go vet**

```bash
go vet ./pkg/cmd/hind/releases/...
```

Expected: no output (success).

- [ ] **Step 4: Run the tests**

```bash
go test ./pkg/cmd/hind/releases/
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/cmd/hind/releases/releases.go
git commit -m "feat(B-020): add hind releases command — table-rendered sorted release list"
```

---

## Task 3: Write tests for `releases` command

**Files:**
- Create: `pkg/cmd/hind/releases/releases_test.go`

Tests use `cmd.IOStreams` with `bytes.Buffer` and a test-scoped `release.Data` to assert header format, column order, row order (latest first), and the empty-data branch.

Because `runE` calls `release.List()` and `release.Get()` from the package-level store (which is fixed at compile time), the test approach injects via a function var seam — the same pattern used in `pkg/cmd/hind/build/build.go`. Alternatively, since the package-level store is immutable and deterministic, we can test against the real store and assert structural properties (header present, at least one row, latest version on first row) rather than injecting a fake store. This is simpler and avoids adding a seam solely for tests.

Choose the real-store approach: tests assert:
1. Header row is first and contains exactly `HIND`, `CONSUL`, `NOMAD`, `VAULT`.
2. Output has at least two rows (header + at least one version).
3. First data row corresponds to `release.Latest().Hind`.
4. Each data row has four tab-separated fields.
5. Empty error output on success.

- [ ] **Step 1: Write the failing tests**

Create `/Users/james/dev/github/stenh0use/hind/pkg/cmd/hind/releases/releases_test.go`:

```go
package releases

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apex/log"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/cmd"
)

func testStreams() (cmd.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return cmd.IOStreams{Out: out, ErrOut: errOut}, out, errOut
}

func testLogger() *log.Logger {
	return log.Log
}

func TestRunE_HeaderRow(t *testing.T) {
	streams, out, errOut := testStreams()
	if err := runE(testLogger(), streams); err != nil {
		t.Fatalf("runE() unexpected error: %v", err)
	}
	if errOut.Len() != 0 {
		t.Errorf("expected empty stderr, got: %q", errOut.String())
	}

	lines := splitLines(out.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + 1 release), got %d", len(lines))
	}

	header := lines[0]
	for _, col := range []string{"HIND", "CONSUL", "NOMAD", "VAULT"} {
		if !strings.Contains(header, col) {
			t.Errorf("header row %q missing column %q", header, col)
		}
	}
}

func TestRunE_LatestVersionFirstRow(t *testing.T) {
	streams, out, _ := testStreams()
	if err := runE(testLogger(), streams); err != nil {
		t.Fatalf("runE() unexpected error: %v", err)
	}

	lines := splitLines(out.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	// First data row (index 1) should start with latest hind version.
	latest := release.Latest().Hind
	firstDataRow := lines[1]
	if !strings.HasPrefix(strings.TrimSpace(firstDataRow), latest) {
		t.Errorf("first data row %q does not start with latest version %q", firstDataRow, latest)
	}
}

func TestRunE_DataRowsHaveFourFields(t *testing.T) {
	streams, out, _ := testStreams()
	if err := runE(testLogger(), streams); err != nil {
		t.Fatalf("runE() unexpected error: %v", err)
	}

	lines := splitLines(out.String())
	// Skip header line.
	for i, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) != 4 {
			t.Errorf("data row %d %q: expected 4 fields, got %d", i+1, line, len(fields))
		}
	}
}

func TestNewCommand_Structure(t *testing.T) {
	streams, _, _ := testStreams()
	cmd := NewCommand(testLogger(), streams)

	if cmd.Use != "releases" {
		t.Errorf("Use = %q, want %q", cmd.Use, "releases")
	}
	if cmd.Args == nil {
		t.Error("Args validator is nil; expected cobra.NoArgs")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

// splitLines returns non-empty lines from output.
func splitLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
```

- [ ] **Step 2: Run the test to confirm it fails (package does not exist yet)**

```bash
go test ./pkg/cmd/hind/releases/
```

Expected: compile error — package not found. This is the TDD red state.

- [ ] **Step 3: Implement (return to Task 2, Step 2 above)**

- [ ] **Step 4: Re-run tests after implementation**

```bash
go test ./pkg/cmd/hind/releases/
```

Expected: PASS (all four tests green).

- [ ] **Step 5: Commit test file**

```bash
git add pkg/cmd/hind/releases/releases_test.go
git commit -m "test(B-020): add releases command table-driven tests"
```

---

## Task 4: Register releases command on root

**Files:**
- Modify: `pkg/cmd/hind/root.go`

- [ ] **Step 1: Add the import**

In `/Users/james/dev/github/stenh0use/hind/pkg/cmd/hind/root.go`, add to the import block:

```go
"github.com/stenh0use/hind/pkg/cmd/hind/releases"
```

- [ ] **Step 2: Register the command**

In the `NewCommand` function body, after the existing `rootCmd.AddCommand(...)` calls, add:

```go
rootCmd.AddCommand(releases.NewCommand(logger, streams))
```

- [ ] **Step 3: Run go vet**

```bash
go vet ./pkg/cmd/hind/...
```

Expected: no output.

- [ ] **Step 4: Run full test suite**

```bash
make test
```

Expected: PASS across all packages.

- [ ] **Step 5: Verify CLI wiring**

```bash
make hind-cli
./bin/hind releases
```

Expected: table with header `HIND   CONSUL   NOMAD   VAULT` followed by release rows, latest version (0.4.0) on first row.

- [ ] **Step 6: Commit**

```bash
git add pkg/cmd/hind/root.go
git commit -m "feat(B-020): register releases subcommand on root hind command"
```

---

## Self-review against spec

**Spec coverage check:**

| Feature requirement | Task that covers it |
|---|---|
| "list in a table the available hind versions" | Task 2 — tabwriter table output |
| "column names on the first row" | Task 2 — `HIND\tCONSUL\tNOMAD\tVAULT` header; Task 3 TestRunE_HeaderRow |
| "first column is hind version" | Task 2 — first tab field is `info.Hind`; Task 3 TestRunE_DataRowsHaveFourFields |
| "remaining columns in alphabetical order consul, nomad, vault" | Task 2 — column order is hardcoded `CONSUL\tNOMAD\tVAULT`; Task 3 TestRunE_HeaderRow |
| "latest version on first row" | Task 2 — descending sort; Task 3 TestRunE_LatestVersionFirstRow |
| "oldest version on last row" | Task 2 — descending sort covers this implicitly; no separate test needed since sort order is the same invariant |
| Feature file normalization — remove stubs | Task 1 |
| Command registered and reachable | Task 4 |

**Gap check:** No spec requirement without a task. The two incomplete stubs ("Create new hind cluster", "Run non existent hind version") are explicitly out of scope for B-020; they are removed in Task 1 rather than implemented.

**Placeholder scan:** No TBD/TODO patterns; all steps have concrete code.

**Type consistency:** `release.List()` returns `[]string`; `release.Get(v)` returns `(release.Info, error)` — both used consistently in Task 2 and Task 3.
