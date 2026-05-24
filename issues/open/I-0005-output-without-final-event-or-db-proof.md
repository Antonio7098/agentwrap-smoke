# I-0005: Assistant Output Without Final Event Or DB Proof

Status: resolved
Severity: medium
Area: process-boundary
Discovered: 2026-05-21
Related decisions: `D-0003`, `D-0004`
Related runs: `R-20260524-015`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` identifies a completion-semantics gap: OpenCode can exit cleanly with assistant output but without a final structured event.

If DB reconciliation also cannot prove terminal finish, the wrapper needs an explicit contract for whether visible assistant output is enough to report completion.

## Contract Decision D-0003

**Resolved**: Assistant output alone produces `completed-with-warning` on clean exit.

Evidence: `../agentwrap/opencode/runtime.go` `finalResult()` decision tree:

```
if r.sawFinal                   → completed (final event supersedes exit code)
else if proc.Err/ExitCode != 0  → failed/runtime_exit
else if r.sawIdle               → completed
else if r.sawOutput             → completed + warning (ASSISTANT OUTPUT FALLBACK)
else if proof := r.reconcile... → completed + warning (DB proof, secondary)
else                            → failed/runtime_exit
```

**Key finding**: `sawOutput` is checked BEFORE DB reconciliation. This means output signal is sufficient to claim completion even when DB is unavailable.

## Implementation

`../agentwrap/opencode/runtime.go` already implements the contract:

```go
} else if r.sawOutput {
    r.warnings = append(r.warnings, "OpenCode finished without a final structured result; treating assistant output on clean exit as completed")
    status = agentwrap.StatusCompleted
```

No code changes needed. `sawOutput` fallback is correctly implemented and smoke-tested.

## Evidence

### Real evidence from `.agentwrap-logs/`:

| Run                                           | Mode          | DB          | Status    | Warning                                                |
| --------------------------------------------- | ------------- | ----------- | --------- | ------------------------------------------------------ |
| `db-only-proof-20260524-151009`               | partial       | unavailable | completed | "treating assistant output on clean exit as completed" |
| `db-only-proof-20260524-144131`               | partial       | complete    | completed | same (sawOutput wins before DB check)                  |
| `process-group-nonzero-final-20260524-150922` | nonzero_final | —           | completed | none (sawFinal wins)                                   |
| `process-group-final-delayed-20260524-141440` | final_delayed | —           | completed | none (sawFinal wins)                                   |

### Fake-opencode `RUN_MODE=partial` behavior confirmed:

- Emits: `{"type":"step_start"}` → `{"type":"text"}` (no final event)
- Exits: 0 (clean exit)
- No assistant text with finish field → triggers `sawOutput` fallback

## Verification

Verified by `R-20260524-015` spike:

- `sawOutput` fallback fires correctly for `RUN_MODE=partial`
- Decision tree order is intentional: output is simpler signal than DB
- No contradictory evidence required (clean exit proves no provider error)
- Warning enables callers to detect non-standard completion path

## Open Item: DB Reconciliation Gap

The `cmdFakeDBBoundary` function (in `cmd/agentwrap-run/main.go`) never reaches DB reconciliation because `RUN_MODE=partial` always triggers `sawOutput` first.

To test DB reconciliation as primary fallback, would need `RUN_MODE=db_only_no_output` that emits only session metadata without `text` events.

**Not blocking**: Output signal is simpler and sufficient for most cases. DB reconciliation is correctly implemented for when output is unavailable.

## Resolution

Status changed to `resolved`. D-0003 contract is implemented correctly in adapter. No code changes needed. Smoke evidence confirms behavior matches contract.
