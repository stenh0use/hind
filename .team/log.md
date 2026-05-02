# Team Review Log

## B-013 — Migrate image build runtime to pkg/provider

**Date:** 2026-05-01
**Reviewer:** Staff Engineer
**Branch:** b-013-provider-migration
**Verdict:** Approved with minor fixes

### Summary

All acceptance criteria (AC-1 through AC-7) are structurally satisfied. The migration is clean and the architecture is correct. Three issues must be fixed before merge; none require re-review.

### Required fixes before merge

**Fix 1 — Dead-code guard in `pkg/provider/dockercli/build.go` lines 52–54 (QA-flagged, must be removed)**
`imageRef := fmt.Sprintf("%s:%s", opts.Name, opts.Tag)` can never produce an empty string when `opts.Name` and `opts.Tag` have already been validated non-empty at lines 25–32. The guard `if imageRef == ""` is unreachable. Remove lines 52–54 entirely. Replace with the direct assignment `imageRef := fmt.Sprintf("%s:%s", opts.Name, opts.Tag)`.

**Fix 2 — `defer cancel()` inside a loop in `pkg/cmd/hind/build/build.go` line 71**
The `defer cancel()` inside the `for _, k := range kinds` loop defers all cancels to the function return, not to the end of each iteration. For a single-image build this is harmless, but for `hind build all` the first iteration's context leaks until the outer `runE` returns. Each cancel must fire at the end of its own iteration. Wrap the loop body in a closure or call cancel explicitly at the end of each iteration (not via defer).

**Fix 3 — Missing `TestBuilder_BuildImage_CallsProviderBuildImage` in `pkg/build/image/builder_test.go`**
The plan (P-013, Phase 3+4) explicitly requires this test: it must use a stub client that records `BuildImageOptions` and asserts `Name`, `Tag`, `ContextDir`, and at least one `BuildArgs` key match expected values. It is absent. Add it before merge.

### Additional observations (non-blocking, engineer discretion)

- `TagExists` and `PullImage` in `pkg/provider/dockercli/build.go` bypass `c.executor` and call `baseClientCmd` directly. This is a pre-existing inconsistency not introduced by this branch, but it means those two methods cannot be faked via the `CommandExecutor` seam. Fine to leave for a follow-up.
- The `mock.ClientStub` default for `BuildImage` (returns zero `BuildImageResult{}` with no error) is technically invalid per the spec contract (empty `Digest` and `ImageRef` are error conditions). Tests using the stub directly should supply a `BuildImageFn`. This is acceptable for a test double but worth a comment.
- `imageRef` variable in `build.go` is set but its intermediate assignment could be made clearer once Fix 1 is applied (just inline `fmt.Sprintf` into the return struct literal).

### Next action

Engineer applies the three required fixes. No re-review required.
