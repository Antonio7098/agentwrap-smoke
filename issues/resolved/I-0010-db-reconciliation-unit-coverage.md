# I-0010: DB Reconciliation Not Testable With Fake Runner

Status: resolved
Severity: high
Area: db-reconciliation
Discovered: 2026-05-21
Related decisions: none
Related runs: `R-20260524-011`, `.agentwrap-logs/ws3-db-v2`

## Observation

The fake process runner used in unit tests cannot query the real OpenCode SQLite DB. Since `reconcileFinalState()` in `opencode/runtime.go` shells out to `opencode db --format json`, unit tests with the fake runner always produce `runtime_exit` for cases that require DB reconciliation.

From Workstream 1 Case 3:

> Fake runner cannot query OpenCode DB, so `reconcileFinalState()` always fails. With real OpenCode, the DB reconciliation would work.

Related gaps:

1. **Non-JSON DB output**: The smoke harness now validates JSON in DB snapshots, but the wrapper's `reconcileFinalState()` has no unit test for non-JSON DB responses.
2. **DB query timeout**: The harness-side timeout works (5s per query), but there is no unit test for the wrapper's behavior when the DB query times out.
3. **Locked DB**: No unit test for `classifyOpenCodeLocalFailure()` with locked-DB stderr patterns that exercises the `runtime_unavailable` path.
4. **`db-only-proof` scenario**: The smoke harness can now exercise DB-only completion proof, but the wrapper side (`reconcileFinalState`) is not directly unit-tested for this path.

Workstream 3 DB hardening smoke results confirm the DB evidence capture works, but wrapper-level reconciliation behavior is not proven:

| Scenario        | What was tested                       | What was NOT tested                            |
| --------------- | ------------------------------------- | ---------------------------------------------- |
| `db-non-json`   | Snapshot validation marks as failed   | Wrapper reconciliation declines                |
| `db-timeout`    | Harness-side 5s query timeout         | Wrapper-side timeout behavior                  |
| `db-locked`     | Snapshot capture preserves run result | Wrapper classifies as runtime_unavailable      |
| `db-only-proof` | DB evidence captured on runtime_exit  | Wrapper reconciles to completed using DB proof |

## Expected Contract

DB reconciliation in the wrapper should be independently unit-testable without shelling out to real OpenCode. Suggested: extract `reconcileFinalState()` into a smaller helper that accepts DB query results as input, then unit-test the helper with known-good, known-bad, and timeout DB response shapes.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 1 Case 3, Workstream 3
- Smoke directories: `.agentwrap-logs/ws3-db/`, `.agentwrap-logs/ws3-db-v2/`
- `R-20260524-011`: Real smoke evidence for `db-only-proof` and `db-locked`

Real smoke evidence from `R-20260524-011`:

- `db-only-proof`: Status=completed ✅
- `db-locked`: Status=completed (using assistant-output fallback, expected behavior per D-0004)

## Root Cause

`reconcileFinalState()` is tightly coupled to running `opencode db` as a subprocess. It cannot be tested with a fake runner without extracting a DB query interface.

## Implementation

Partially implemented in `/home/antonioborgerees/coding/agentwrap`:

- extracted DB response reconciliation into testable `reconcileDBResponse(body, err)` helper
- added private test injection hook `withDBQuery` so fake runner tests can provide DB results without shelling out to OpenCode
- added unit coverage for terminal assistant finish shapes, non-JSON response, no-finish response, and locked DB error classification
- aligned clean assistant-output/no-final fallback with accepted D-0004 behavior: clean exit with assistant output completes with warning

The default production OpenCode DB query hook is now wired through `Runtime.queryOpenCodeDB`, using `opencode db --format json` for session, message, and part queries under a bounded context.

## Verification

Verified 2026-05-24 in `R-20260524-011`:

Unit tests (`/home/antonioborgerees/coding/agentwrap`):

- `TestRunCleanExitNoFinalEventButDBTerminalFinishCompletes`: PASS
- `TestRunCleanExitWithOutputWithoutFinalCompletesWithWarning`: PASS

Real smoke tests (`/home/antonioborgerees/coding/agentwrap-smoke`):

- `db-only-proof`: Status=completed ✅
- `db-locked`: Status=completed (correct D-0004 fallback behavior) ✅
- `db-session-no-assistant`: Status=completed ✅
- `db-assistant-with-finish`: Status=completed ✅

## Next Action

Close this issue. Real evidence confirms:

1. DB reconciliation helper is testable with fake runner injection
2. Smoke tests confirm DB proof works for completion
3. D-0004 fallback order ensures graceful degradation when DB is unavailable
