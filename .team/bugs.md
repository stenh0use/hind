# Bugs

Closed items: `.team/done/bugs/`

---

<!-- Template:
## W-xxx — Title

**Severity:** P1 / P2 / P3
**Status:** open / in-progress / resolved
**Source:** where it came from

**Repro steps:**
1. ...

**Expected:** ...
**Actual:** ...

**Spec:** `W-xxx.md` (if one exists)
-->

---

## BUG-001 — `hind get` succeeds silently when cluster has no config file on disk

**Severity:** P2
**Status:** open
**Source:** QA audit of assigned commands (get, list, stop, set profile)

**Root cause file:** `pkg/cluster/manager.go:325-331` (`LoadPersistedConfig`)

**Repro steps:**
1. Ensure no cluster named "ghost" has ever been created (no config file at `~/.config/hind/cluster/ghost/cluster.json`).
2. Run `hind get ghost`.
3. Observe exit code and output.

**Expected:** Command exits non-zero with a "cluster not found" error message.

**Actual:** `LoadPersistedConfig` (called by `Manager.Get`) falls through the `!m.ConfigFileExists()` branch without returning an error because `m.config.Name` is non-empty (it was set by `cluster.New` from the supplied arg). The manager then calls `provider.InspectNetwork` and `provider.InspectContainer` using the freshly-synthesised default config, both return `nil` (no such resources), and `Get` returns an empty `ClusterInfo` with no error. `get.runE` renders a table showing `Status: N/A`, `Network: ` (empty), and exits 0. The user receives no indication that the cluster does not exist.

**Contrast with `hind stop`:** `stop.runE` explicitly calls `clusterMgr.ConfigFileExists()` before proceeding and returns an error if false. `get.runE` has no equivalent guard.

---

## BUG-002 — `hind list` swallows `tabwriter.Flush` error

**Severity:** P3
**Status:** open
**Source:** QA audit of assigned commands (get, list, stop, set profile)

**Root cause file:** `pkg/cmd/hind/list/list.go:110`

**Repro steps:**
1. Any `hind list` invocation that reaches the table-printing path (at least one cluster exists).
2. Observe: the return value of `w.Flush()` is discarded.

**Expected:** The error returned by `w.Flush()` (e.g. a broken-pipe or closed writer) is propagated and causes `runE` to return a non-zero exit code, consistent with how `get.runE` handles `w.Flush()` at line 85-87 of `get.go`.

**Actual:** `w.Flush()` is called as a bare statement with no error check (`w.Flush()` on line 110, no `if err :=` wrapper). If the flush fails the error is silently dropped and the command exits 0, potentially producing truncated output.

---

## BUG-003 — `hind stop` exits 0 even when containers fail to stop

**Severity:** P2
**Status:** open
**Source:** QA audit of assigned commands (get, list, stop, set profile)

**Root cause file:** `pkg/cmd/hind/stop/stop.go:115-121`

**Two related issues in the same function:**

**Issue A — `FailedCount > 0` branch always returns nil (line 121)**
When one or more containers fail to stop (without `--force`), `runE` prints the partial-stop warning to `ErrOut` but returns `nil` — exit 0. Scripts cannot detect the failure.

**Issue B — `--force` branch (line 115) evaluated before `FailedCount` check (line 119)**
When `force=true` and containers failed to kill, the `if force` branch fires first and returns `nil` with "force stopped", so the `FailedCount` path is never reached at all. Individual failure messages are still printed to `ErrOut` (lines 103-105) but the exit code and final status message are incorrect.

**Repro steps (Issue A):**
1. Start a cluster with at least two containers.
2. Arrange for one container's stop to fail (e.g. Docker daemon error on that container).
3. Run `hind stop <cluster-name>` (no --force).
4. Observe exit code is 0 despite partial failure.

**Repro steps (Issue B):**
1. Same setup but run `hind stop --force <cluster-name>`.
2. Observe "force stopped" message and exit 0 — `FailedCount` path is never reached.

**Expected:** Both paths should return a non-nil error when `result.FailedCount > 0`.

---

## BUG-004 — `hind rm` succeeds silently when cluster does not exist

**Severity:** P2
**Status:** open
**Source:** QA audit of assigned commands (rm, start, build)

**Root cause file:** `pkg/cluster/manager.go:221-276` (`Delete`)

**Repro steps:**
1. Ensure no cluster named `ghost` exists.
2. Run `hind rm ghost`.
3. Observe output and exit code.

**Expected:** Command exits non-zero with an error such as `cluster 'ghost' does not exist`.

**Actual:** Command exits 0 and prints `Cluster 'ghost' deleted successfully`. `Delete()` does not fail when config/resources are absent, so non-existent deletion is treated as success.

---

## BUG-005 — `hind start` never returns `StartResultAlreadyRunning`

**Severity:** P3
**Status:** open
**Source:** QA audit of assigned commands (rm, start, build)

**Root cause file:** `pkg/cluster/manager.go:72-108` (`Start`)

**Repro steps:**
1. Run `hind start mycluster`.
2. Run `hind start mycluster` again while already fully running.

**Expected:** Already-running no-op path should return `StartResultAlreadyRunning` and avoid redundant connection-info output.

**Actual:** Existing-cluster path always returns `StartResultResumed`, even when reconcile has no actions. `StartResultAlreadyRunning` appears to be dead/unused.

---

## BUG-006 — `hind build` wrong-arg error message uses slice formatting

**Severity:** P3
**Status:** open
**Source:** QA audit of assigned commands (rm, start, build)

**Root cause file:** `pkg/cmd/hind/build/build.go:39`

**Repro steps:**
1. Run `hind build` (no args), or `hind build a b`.

**Expected:** Error should report argument count cleanly (e.g. `received 0` / `received 2`) or use Cobra's default exact-args error.

**Actual:** Custom message uses `%s` with `[]string`, producing messages like `accepts 1 arg, received []` or `[a b]`.

---

## BUG-007 — `hind set profile` writes success message to stderr

**Severity:** P3
**Status:** open
**Source:** QA audit of assigned commands (get, list, stop, set profile)

**Root cause file:** `pkg/cmd/hind/set/set.go:42`

**Repro steps:**
1. Run `hind set profile <existing-cluster>`.
2. Redirect stderr to `/dev/null`.

**Expected:** Success output should be on stdout if treated as user-facing command result.

**Actual:** Success message is written to `streams.ErrOut`, so it disappears when stderr is redirected.
