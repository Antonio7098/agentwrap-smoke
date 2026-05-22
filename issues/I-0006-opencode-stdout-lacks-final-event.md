# I-0006: OpenCode JSON Mode Stdout Lacks Final Structured Event

Status: open
Severity: critical
Area: process-boundary
Discovered: 2026-05-21
Related decisions: D-0004
Related runs: none yet in `runs/`

## Observation

`opencode run --format json` does not reliably emit a final `finish` / `step_finish` event on stdout. The wrapper expects such an event to set `sawFinal = true`, which is required for clean `completed` classification. Without it, successful runs are misclassified as `runtime_exit`.

This is the root cause across most session lifecycle tests (Workstream 4):

| Command | Expected | Actual |
|---------|----------|--------|
| `session-fresh` | `completed` | `runtime_exit` |
| `session-continue-existing` | `completed` | `runtime_exit` |
| `session-continue-missing` | `failed` with explicit category | `runtime_exit` |
| `repair-with-continue` | `completed` | `runtime_exit` |

Evidence from `session-fresh` run:

```
Status: failed
Category: runtime_exit
Event types: lifecycle.transition, session.relationship, step_start
No finish event received
```

Source analysis (`ultraplan/studies/go-cli-study/sources/opencode`): OpenCode's `CoderAgent.Run()` returns a final event with `Done: true`, but the CLI boundary in `internal/app/app.go` consumes that event and prints only formatted assistant content through `format.FormatOutput`. The structured completion/error information is not exposed as a stable JSON stdout event.

## Expected Contract

The wrapper must not depend on `step_finish` being present in stdout. Fallback mechanisms should handle the gap:

1. Assistant output (`text`/`reasoning`) on clean exit → `completed` with warning (partially applied)
2. DB terminal finish on clean exit → `completed` with DB-recovered usage (partially applied)
3. Empty clean exit without either → `runtime_exit` (applied)

## Evidence

- `AGENTWRAP_REPORTING.md` "Bug: step_finish Event Not Reliably Emitted by opencode CLI"
- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 4 findings
- Source analysis: `ultraplan/studies/go-cli-study/sources/opencode`

## Root Cause

OpenCode CLI exposes stdout as a progress stream, not a complete final-state contract. The non-interactive CLI boundary truncates the agent's final completion event.

## Implementation

Adapter workarounds applied in `agentwrap/opencode/runtime.go`:

1. `sawOutput` flag tracks `text`/`reasoning` events → clean exit + output = completed with warning
2. `reconcileFinalState()` queries OpenCode DB for terminal assistant finish → clean exit + DB proof = completed with warning
3. Empty clean exit without either → `runtime_exit` (unchanged)

## Verification

- `TestRunCleanExitWithFinalEventCompletes`: PASS (final event path)
- `TestRunCleanExitWithOutputWithoutFinalCompletesWithWarning`: PASS (sawOutput path)
- `TestRunCleanExitNoFinalNoOutputNoDBFinishFails`: PASS (empty exit path)
- Real smoke: `./agentwrap-run usage --model opencode/deepseek-v4-flash-free` → completed with DB-projected usage

## Remaining Risk

- DB reconciliation depends on `opencode db` being available and not contended.
- Output-only completion accepts partial output risk when the process exits before finishing.
- Session lifecycle tests still cannot pass with real OpenCode because stdout lacks final events and smoke harness does not use the DB reconciliation path for session scenarios.

## Next Action

Decide D-0004 (which fallback signals are sufficient for completion), then ensure every smoke test that needs completion from DB/output is explicitly configured for it.
