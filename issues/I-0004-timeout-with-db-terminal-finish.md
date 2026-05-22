# I-0004: Timeout With DB Terminal Finish

Status: open
Severity: medium
Area: timeout
Discovered: 2026-05-21
Related decisions: `D-0002`
Related runs: none yet in `runs/`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` identifies an unresolved contract: a caller timeout may fire while OpenCode durable state later shows a terminal assistant finish.

This is not a simple success case. The caller's deadline was exceeded, but the child process may have persisted useful final state.

## Expected Contract

Pending `D-0002`.

Current proposed contract: preserve status `failed` with category `timeout`, and include DB terminal-finish evidence in native metadata or warnings.

## Evidence

Source document:

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md`

Required next evidence:

- focused unit test for timeout with session/DB terminal finish
- real or fixture run if OpenCode can reproduce the race
- DB snapshot showing terminal assistant message and usage

## Root Cause

External process boundary race between caller deadline, subprocess lifecycle, stdout event delivery, and OpenCode SQLite persistence.

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

Decide `D-0002`, then ensure adapter metadata records durable completion evidence without converting the timeout into silent success.

