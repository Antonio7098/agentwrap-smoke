# I-0013: Cancellation Final Outcome Is Not Preserved Strongly Enough

Status: resolved
Severity: medium
Area: cancellation
Discovered: 2026-05-22
Related decisions: none
Related runs: `R-20260524-008`, `R-20260524-009`

## Observation

The OpenCode lifecycle report shows that OpenCode has a structured cancellation path: runner cancellation, LLM stream abort, interrupt hooks, and interrupted assistant finalization.

The adapter path is process-oriented: `Cancel()` emits cancelled lifecycle, sends SIGTERM/SIGKILL cleanup, and the run goroutine may still be draining events or producing the final result.

## Expected Contract

Cancellation should preserve the final run outcome as far as the process boundary allows.

Minimum contract:

- explicit caller cancellation reports category `cancellation`
- cleanup errors do not replace the meaningful cancellation outcome
- late OpenCode cancellation/error events are either drained or explicitly recorded as missed
- `Cancel()` should not make it harder for callers to retrieve the final `RunResult`

## Evidence

Source report:

- `/home/antonioborgerees/coding/ultraplan/targets/agentwrap/reports/opencode-cancellation-failures-end-of-run.md`

Relevant sections:

- `1.1 OpenCode's Layered Cancellation`
- `1.2 agentwrap's Current Cancellation`
- `1.3 Gaps and Recommendations`

## Root Cause

The adapter controls OpenCode through an OS process boundary. It cannot directly invoke OpenCode's internal `Runner.cancel`, so it must decide how long to wait for final cancellation evidence before killing the process group.

## Implementation

In progress / initial preservation fix implemented in `/home/antonioborgerees/coding/agentwrap` on 2026-05-24:

- `opencode/runtime.go`: `Cancel()` preserves explicit cancellation as the primary caller-visible outcome even if process cleanup reports an error.
- `opencode/runtime.go`: `Cancel()` waits briefly for final cancellation evidence (`r.done`), bounded by 100ms or caller context cancellation.
- `opencode/runtime.go`: cleanup errors no longer overwrite completed/cancelled primary outcomes in `finalResult()`.
- `opencode/runtime.go`: cleanup failures remain recorded in `RunMetadata.Cleanup`, `RunMetadata.Errors`, warnings, and `NativeMetadata.cleanup_warning`.
- `opencode/runtime.go`: cleanup failure no longer emits a lifecycle transition to failed after the run outcome is already determined.
- `opencode/runtime_test.go`: added coverage for completed + cleanup failure and explicit cancellation + cleanup failure.

## Verification

Verified 2026-05-24 in `R-20260524-008`:

- `/home/antonioborgerees/coding/agentwrap`: `go test ./opencode`
- `/home/antonioborgerees/coding/agentwrap`: `go test ./...`
- `/home/antonioborgerees/coding/agentwrap-smoke`: `go test ./...`

Smoke evidence in `R-20260524-009`:

- `go run ./cmd/agentwrap-run cancel`: real OpenCode cancellation returned status `cancelled`, category `cancellation`, and `Cancel succeeded without error`.
- `go run ./cmd/agentwrap-run process-group-cancel-children`: fake process-group cancellation returned `cancelled`/`cancellation` and confirmed both fake OpenCode and helper child PIDs were not alive.

## Remaining Risk

Waiting too long during cancellation can make caller cancellation feel hung. Any drain window must be short and bounded.

## Next Action

Review `R-20260524-008` and `R-20260524-009`; move to resolved if accepted. Any longer native abort/final-cancellation evidence capture should be tracked as a transport-level follow-up rather than blocking this subprocess contract fix.

