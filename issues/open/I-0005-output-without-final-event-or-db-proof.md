# I-0005: Assistant Output Without Final Event Or DB Proof

Status: open
Severity: medium
Area: process-boundary
Discovered: 2026-05-21
Related decisions: `D-0003`
Related runs: none yet in `runs/`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` identifies a completion-semantics gap: OpenCode can exit cleanly with assistant output but without a final structured event.

If DB reconciliation also cannot prove terminal finish, the wrapper needs an explicit contract for whether visible assistant output is enough to report completion.

## Expected Contract

Pending `D-0003`.

Current proposed contract: assistant output alone can produce completed-with-warning only when there is no contradictory process/provider evidence and the output is known useful. Otherwise prefer `runtime_exit`.

## Evidence

Source document:

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md`

Required next evidence:

- unit test for clean exit with assistant output and no final event
- real smoke run showing whether DB reconciliation is available
- `results.json`, `events.jsonl`, and DB snapshot status

## Root Cause

OpenCode stdout may omit terminal structure even when useful assistant text was emitted.

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

Decide `D-0003`, then align adapter behavior, warnings, and regression coverage with that contract.

