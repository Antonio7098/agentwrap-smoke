# I-0013: Cancellation Final Outcome Is Not Preserved Strongly Enough

Status: open
Severity: medium
Area: cancellation
Discovered: 2026-05-22
Related decisions: none
Related runs: none yet in `runs/`

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

No implementation recorded in this repo yet.

Candidate implementation work:

- make `Cancel()` wait briefly on `r.done` after signaling cleanup
- preserve cleanup errors as metadata/warnings instead of primary result when cancellation is already established
- add a cancellation smoke/unit run record showing final result stability

## Verification

Not verified.

## Remaining Risk

Waiting too long during cancellation can make caller cancellation feel hung. Any drain window must be short and bounded.

## Next Action

Inspect current `agentwrap/opencode/runtime.go` cancellation code and add the smallest unit test for `Cancel()` returning while the run goroutine is still finalizing.

