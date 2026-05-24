# I-0004: Timeout With DB Terminal Finish

Status: in_progress
Severity: medium
Area: timeout
Discovered: 2026-05-21
Related decisions: `D-0002`
Related runs: `R-20260524-014`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` identifies an unresolved contract: a caller timeout may fire while OpenCode durable state later shows a terminal assistant finish.

This is not a simple success case. The caller's deadline was exceeded, but the child process may have persisted useful final state.

## Expected Contract

**Decision `D-0002` (proposed)**:

> "keep status `failed` with category `timeout`, but include DB terminal-finish evidence in native metadata"

## Evidence

### Spike Run: `R-20260524-014`

**Real timeout test**:

- Log: `.agentwrap-logs/timeout-20260524-150859/`
- Status: `failed`, Category: `timeout`
- Session ID: empty (no session was created due to fast timeout)
- DB snapshot: skipped (no session ID)

**DB-only proof fixture** (tests last-resort DB reconciliation):

- Log: `.agentwrap-logs/db-only-proof-20260524-144131/`
- Status: `completed`
- DB snapshot shows terminal assistant message with `finish: "stop"`

## Root Cause Analysis

### Critical Finding: DB reconciliation is a LAST RESORT fallback

The `finalResult()` method in `runtime.go` follows this precedence:

1. **decodeErr DeadlineExceeded** → status=failed, category=timeout
2. **ctx.Err() DeadlineExceeded** → status=failed, category=timeout
3. **sawFinal** → status=completed
4. **proc error/non-zero** → status=failed
5. **sawIdle** → status=completed
6. **sawOutput** → status=completed with warning
7. **reconcileFinalState()** → status=completed or status=failed

**When a caller timeout fires, the method returns early at step 1 or 2, and `reconcileFinalState()` is NEVER called.**

### Why This Is Correct

Consulting DB on timeout would create a race condition:

1. Caller deadline fires
2. We query DB (which may be slow or locked)
3. DB returns terminal finish
4. We would need to decide: keep timeout OR convert to completed

The adapter correctly chooses to:

1. Honor the caller's deadline contract
2. Return timeout immediately
3. NOT wait for DB reconciliation

### Session ID Behavior

`reconcileFinalState()` has a guard:

```go
if (r.req.SessionID == "" && r.sessionID == "") || r.dbQuery == nil {
    return dbReconcileProof{}
}
```

The `sessionID` is updated during `scanNativeRecords()` from events. If no events arrive before timeout, `sessionID` remains empty, and DB reconciliation would be skipped anyway.

## Implementation

**No code change needed.** The current implementation is correct per D-0002.

The `reconcileFinalState()` is called ONLY as a last resort fallback after:

- No final event was seen (`!sawFinal`)
- Process exited cleanly (`proc.Err == nil && proc.ExitCode == 0`)
- No idle signal (`!sawIdle`)
- No output (`!sawOutput`)

## Verification

**Verified by `R-20260524-014`**:

- Real timeout returns `failed` with `timeout` (correct)
- DB reconciliation completes runs without final events (correct)
- Session ID capture works when events arrive before timeout

## Notes

The contract D-0002 is correctly implemented:

- Caller timeout remains `timeout`, not upgraded to completion
- DB reconciliation only used as last resort fallback

If future requirements want DB evidence recorded on timeout (without changing status), this would require a separate code path.

## Next Action

Close as wontfix (behavior is correct per D-0002) or convert to monitoring status. Document the "DB reconciliation is last resort" behavior in the adapter code comments for clarity.
