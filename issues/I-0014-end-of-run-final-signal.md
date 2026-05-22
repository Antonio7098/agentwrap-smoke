# I-0014: End-Of-Run Detection Uses Too Coarse A Final Signal

Status: open
Severity: high
Area: end-of-run
Discovered: 2026-05-22
Related decisions: `D-0001`, `D-0003`, `D-0004`
Related runs: none yet in `runs/`

## Observation

The OpenCode lifecycle report shows that OpenCode's run completion is more nuanced than a single `step_finish` event:

- prompt loop completion depends on assistant finish, pending tool calls, tool results, and assistant ordering
- session idle is the authoritative bus-level completion event
- `step_finish` can have finish reasons that may not mean final completion
- trailing usage, artifacts, warnings, or errors may arrive around the final boundary

The adapter currently treats projected final events, especially `step_finish`, as the main end-of-run signal.

## Expected Contract

End-of-run detection should distinguish:

- normal final completion
- finish that still has pending tool calls
- permission/error stop
- cancellation/abort
- idle session signal, if OpenCode emits it
- trailing structured evidence after final

The adapter should not mark a run completed solely because a `step_finish` was seen if the native payload says more work or an error state remains.

## Evidence

Source report:

- `/home/antonioborgerees/coding/ultraplan/targets/agentwrap/reports/opencode-cancellation-failures-end-of-run.md`

Relevant sections:

- `3.1 OpenCode's Completion Detection`
- `3.2 agentwrap's Current End-of-Run Detection`
- `3.3 Gaps and Recommendations`

## Root Cause

The adapter compresses OpenCode's richer lifecycle into `sawFinal`. That is pragmatic, but it risks false completion or dropped metadata when native events contain finish reasons, idle events, or trailing data.

## Implementation

No implementation recorded in this repo yet.

Candidate implementation work:

- inspect real `.agentwrap-logs/*/events.jsonl` files for `step_finish` payload shape and possible `session.status idle`
- update `projector.go` so only terminal finish reasons set final
- preserve native finish reason in metadata
- add tests for `finish == "tool-calls"` not marking final completion if that shape exists
- consider a short post-final drain window for trailing events

## Verification

Not verified.

## Remaining Risk

OpenCode event schema may vary by version. Tests should use captured native event fixtures where possible.

## Next Action

Audit existing `.agentwrap-logs/*/events.jsonl` for `step_finish`, `session.status`, finish reasons, usage, and artifacts around the final boundary.

