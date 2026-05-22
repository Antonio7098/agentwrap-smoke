# I-0001: Non-Zero Exit Versus Final Event

Status: open
Severity: high
Area: process-boundary
Discovered: 2026-05-21
Related decisions: `D-0001`
Related runs: none yet in `runs/`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` records a suspected adapter bug where a non-zero OpenCode process exit can override a previously observed final structured event.

Recorded finding:

> Non-zero exit overrides `sawFinal` because `proc.ExitCode != 0` is checked before `!r.sawFinal` in `finalResult()`.

## Expected Contract

Pending `D-0001`.

Candidate contract: a strong final structured completion event should be preserved as completion, while the non-zero process exit is retained as warning/native metadata.

## Evidence

Source document:

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md`
- `/home/antonioborgerees/coding/ultraplan/targets/agentwrap/reports/opencode-cancellation-failures-end-of-run.md`

Additional evidence from the OpenCode lifecycle report:

- Section 2.2 describes `finalResult()` as intending to tolerate non-zero exit after `sawFinal` when the classified error is only `runtime_exit`.
- Section 3.2 confirms that `step_finish` is the adapter's current final signal.
- This sharpens the next check: verify actual current code/tests, because the reports disagree on whether non-zero-exit-overrides-final is still current or already fixed.

Required next evidence:

- unit test name and result from `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go`
- exact adapter code path in `/home/antonioborgerees/coding/agentwrap/opencode/runtime.go`
- relevant real or fixture smoke run if one exists

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

Confirm current adapter behavior with the narrowest unit test. If current code already preserves final events across `runtime_exit`, close this as verified and move attention to richer final-signal handling.
