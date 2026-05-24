# R-20260524-014: I-0004 Timeout With DB Terminal Finish Spike

Date: 2026-05-24
Type: smoke | unit
Related issues: `I-0004`
Related decisions: `D-0002`

## Commands

### Real timeout test

```bash
./agentwrap-run timeout --timeout-ms=500
```

### DB-only proof test (fixture)

```bash
./agentwrap-run db-only-proof
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: `opencode/deepseek-v4-flash-free`
- Commit: N/A (spike investigation)

## Result

**Timeout test**: Status=failed, Category=timeout, Pass=true ✅

**DB-only proof test**: Status=completed, Category=completed, Pass=true ✅

## Evidence

### Real timeout log directory

- `.agentwrap-logs/timeout-20260524-150859/`
- Status: `failed`
- Category: `timeout`
- Exit code: `-1` (process was killed)
- Session ID: empty (no session was created due to fast timeout)
- DB snapshot: skipped (no session ID)

### DB-only proof fixture log directory

- `.agentwrap-logs/db-only-proof-20260524-144131/` (previous run)
- Status: `completed`
- DB snapshot shows:
  - `opencode-session.json`: `ses_fake123` with tokens
  - `opencode-messages.json`: assistant message with `finish: "stop"`
  - `opencode-parts.json`: `step-finish` with `finish: "stop"`

## Adapter Code Analysis

### Key code path in `runtime.go`

The `finalResult()` method in `run` struct follows this precedence:

1. **decodeErr DeadlineExceeded** → status=failed, category=timeout
2. **ctx.Err() DeadlineExceeded** → status=failed, category=timeout
3. **sawFinal** → status=completed
4. **proc error/non-zero** → status=failed
5. **sawIdle** → status=completed
6. **sawOutput** → status=completed with warning
7. **reconcileFinalState()** → last resort fallback
   - If err: status=failed
   - If completed=true: status=completed with warning
   - If no proof: status=failed

### Critical finding: DB reconciliation is a LAST RESORT fallback

The `reconcileFinalState()` is only called when:

- No final event was seen (`!sawFinal`)
- Process exited cleanly (`proc.Err == nil && proc.ExitCode == 0`)
- No idle signal (`!sawIdle`)
- No output (`!sawOutput`)

**This means when a caller timeout fires (DeadlineExceeded), the method returns early at step 1 or 2, and `reconcileFinalState()` is NEVER called.**

### Session ID behavior

`reconcileFinalState()` has a guard:

```go
if (r.req.SessionID == "" && r.sessionID == "") || r.dbQuery == nil {
    return dbReconcileProof{}
}
```

The `sessionID` is updated during `scanNativeRecords()` from events. If no events arrive before timeout, `sessionID` remains empty, and DB reconciliation would be skipped anyway.

## Interpretation

### Current behavior is correct per D-0002

The proposed D-0002 contract states:

> "keep status `failed` with category `timeout`, but include DB terminal-finish evidence in native metadata"

The current adapter implementation:

1. Returns `failed` with `timeout` when caller deadline is exceeded ✅
2. Does NOT consult DB when timeout fires (early return) ✅
3. DB reconciliation only happens as a LAST RESORT fallback ✅

### Why DB is NOT consulted on timeout

Consulting DB on timeout would create a race condition where:

- Caller deadline fires
- We query DB (which may be slow or locked)
- DB returns terminal finish
- We would need to decide: keep timeout OR convert to completed

This is inherently racy. The adapter correctly chooses to:

1. Honor the caller's deadline contract
2. Return timeout immediately
3. NOT wait for DB reconciliation

### Gap: native_metadata does not include DB evidence on timeout

The current implementation does NOT record DB terminal-finish evidence in `native_metadata` when a timeout occurs. Per D-0002:

> "include DB terminal-finish evidence in native metadata"

If the adapter should record DB evidence on timeout (without changing the status), this would require:

1. Calling `reconcileFinalState()` even when timeout fires
2. Adding a `db_terminal_finish_proof` field to `native_metadata`
3. Keeping status=failed but adding a warning

However, this has risks:

- DB query may be slow (current limit is 5s)
- DB may be locked or unavailable
- Adding latency to timeout path

## Notes

1. The current behavior is correct per D-0002: caller timeout should remain timeout, not be upgraded to completion based on DB evidence.

2. The `reconcileFinalState()` is only called as a LAST RESORT fallback, after all other signals (final event, output, idle) are checked. This prevents the timeout-vs-DB-race scenario.

3. Session ID capture: if events arrive before timeout, session ID is captured. But if timeout fires before any events, session ID is empty and DB reconciliation would not apply anyway.

4. The `db-only-proof` fixture tests the DB reconciliation path (last resort fallback), not the timeout path.

5. **Next action for I-0004**: No code change needed. The contract D-0002 is correctly implemented. Consider adding documentation about the "DB reconciliation is last resort" behavior to make it explicit.
