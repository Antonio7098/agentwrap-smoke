# I-0006: OpenCode JSON Mode Stdout Lacks Final Structured Event

Status: resolved
Severity: critical
Area: process-boundary
Discovered: 2026-05-21
Related decisions: D-0004
Related runs: `R-20260524-011`, `R-20260524-010`

## Observation

`opencode run --format json` does not reliably emit a final `finish` / `step_finish` event on stdout. The wrapper expects such an event to set `sawFinal = true`, which is required for clean `completed` classification. Without it, successful runs are misclassified as `runtime_exit`.

This is the root cause across most session lifecycle tests (Workstream 4):

| Command                     | Expected                        | Actual         |
| --------------------------- | ------------------------------- | -------------- |
| `session-fresh`             | `completed`                     | `runtime_exit` |
| `session-continue-existing` | `completed`                     | `runtime_exit` |
| `session-continue-missing`  | `failed` with explicit category | `runtime_exit` |
| `repair-with-continue`      | `completed`                     | `runtime_exit` |

Evidence from `session-fresh` run (before fix):

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
- `R-20260524-010`: Real OpenCode shape capture confirms CLI stdout only has `step_start` and `text`

Real OpenCode shape capture from `R-20260524-010`:

| File                      | Session                          | stdout event types   | stdout `step_finish` | stdout `session.status` |
| ------------------------- | -------------------------------- | -------------------- | -------------------- | ----------------------- |
| `simple-ok.raw.jsonl`     | `ses_1a5d9c378ffe8im7wZtgByoi72` | `step_start`         | no                   | no                      |
| `no-tools-done.raw.jsonl` | `ses_1a5d9abffffe8l1Zh43FRLeysb` | `step_start`, `text` | no                   | no                      |
| `shape.raw.jsonl`         | `ses_1a5d92aa1ffedwtw80ly0Tsq4j` | `step_start`         | no                   | no                      |

DB terminal proof confirmed:

- `assistant finish: "stop"` in messages
- `part reason: "stop"` in parts

## Root Cause

OpenCode CLI exposes stdout as a progress stream, not a complete final-state contract. The non-interactive CLI boundary truncates the agent's final completion event.

## Implementation

Adapter workarounds applied in `agentwrap/opencode/runtime.go`:

1. `sawOutput` flag tracks `text`/`reasoning` events → clean exit + output = completed with warning
2. `reconcileFinalState()` queries OpenCode DB for terminal assistant finish → clean exit + DB proof = completed with warning
3. Empty clean exit without either → `runtime_exit` (unchanged)

## Verification

Real smoke evidence from `R-20260524-011`:

- `session-fresh --model opencode/deepseek-v4-flash-free`: Status=completed Session ID=ses_1a5d13eb6ffeCGjcWkV2dPtB8z ✅
- `session-continue-existing --model opencode/deepseek-v4-flash-free`: Status=completed Session relationship=best_effort ✅
- `session-continue-missing --model opencode/deepseek-v4-flash-free`: Status=failed Category=runtime_exit ✅

DB evidence files captured:

- `opencode-session.json`
- `opencode-messages.json`
- `opencode-parts.json`

Unit tests:

- `TestRunCleanExitWithFinalEventCompletes`: PASS
- `TestRunCleanExitWithOutputWithoutFinalCompletesWithWarning`: PASS
- `TestRunCleanExitNoFinalNoOutputNoDBFinishFails`: PASS

## Remaining Risk

- DB reconciliation depends on `opencode db` being available and not contended.
- Output-only completion accepts partial output risk when the process exits before finishing.
- Session lifecycle tests still cannot pass with real OpenCode because stdout lacks final events and smoke harness does not use the DB reconciliation path for session scenarios.

## Next Action

Close this issue. Real evidence confirms the fix works for session lifecycle scenarios.
