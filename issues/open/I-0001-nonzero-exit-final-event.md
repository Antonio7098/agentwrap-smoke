# I-0001: Non-Zero Exit Versus Final Event

Status: verified
Severity: high
Area: process-boundary
Discovered: 2026-05-21
Related decisions: `D-0001` (confirmed)
Related runs: `R-20260524-013`

## Observation

`AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` recorded a suspected adapter bug where a non-zero OpenCode process exit could override a previously observed final structured event.

The suspected bug: "Non-zero exit overrides `sawFinal` because `proc.ExitCode != 0` is checked before `!r.sawFinal` in `finalResult()`."

## Investigation Result

**NOT A BUG.** The current adapter code correctly gives `sawFinal` precedence over non-runtime-exit process failures.

The relevant logic in `/home/antonioborgerees/coding/agentwrap/opencode/runtime.go` (`finalResult()`, around line 252):

```go
} else if r.sawFinal {
    if proc.Err != nil || proc.ExitCode != 0 {
        if exitErr := classifyExitError(proc, r.stderrBuffer.String()); exitErr.Category != agentwrap.ErrorRuntimeExit {
            sdkErr = exitErr
            status = agentwrap.StatusFailed
        } else {
            status = agentwrap.StatusCompleted
        }
    } else {
        status = agentwrap.StatusCompleted
    }
}
```

When `sawFinal=true`:

1. If process error OR non-zero exit code exists
2. AND `classifyExitError` does NOT return `ErrorRuntimeExit` (e.g., rate-limit, model error) → `failed`
3. OR if `classifyExitError` returns `ErrorRuntimeExit` (plain non-zero exit with no actionable error) → `completed`
4. If no process error and exit code 0 → `completed`

## Verification

### Unit Test: `TestRunNonZeroExitWithFinalEventStillCompletes`

- **File**: `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go` (line 1289)
- **Fixture**: `final.ndjson` with `processResult{ExitCode: 1, Err: errors.New("exit status 1")}`
- **Assertions**:
  - `result.Status == agentwrap.StatusCompleted` ✅
  - `result.Err == nil` ✅
  - `waitErr == nil` ✅
  - `hasFinalResult == true` (final_result event observed) ✅
- **Result**: ✅ PASS

### Smoke Test: `process-group-nonzero-final`

- **Command**: `go run ./cmd/agentwrap-run process-group-nonzero-final`
- **Fake opencode**: `RUN_MODE=nonzero_final` — emits `step_start`, `text`, `step_finish`, then exits 7
- **Expected**: `completed`
- **Actual**: `completed` ✅
- **Log dir**: `.agentwrap-logs/process-group-nonzero-final-20260524-150922/`
- **Native metadata**:
  - `exit_code`: 7 (preserved, not discarded)
  - `native_terminal_evidence`: `step_finish`
  - `stderr`: `""` (empty)
  - `event_count`: 6
  - `event_categories`: `final_result=1, lifecycle=2, message=1, progress=1, session=1`

## Contract Confirmation (D-0001)

The contract decision aligns with current adapter behavior:

| Scenario                                       | Exit Code | sawFinal | stderr          | Result                       |
| ---------------------------------------------- | --------- | -------- | --------------- | ---------------------------- |
| Final event, clean exit                        | 0         | true     | —               | `completed`                  |
| Final event, non-zero exit (runtime_exit only) | 7         | true     | empty           | `completed` ✅ verified      |
| Final event, non-zero exit, rate-limit stderr  | 7         | true     | rate-limit JSON | `failed` (rate_limit)        |
| No final event, non-zero exit                  | 7         | false    | —               | `failed` (runtime_exit)      |
| Final event, non-zero exit, model error stderr | 7         | true     | model error     | `failed` (model_unavailable) |

**Decision**: Report `completed` when the final event is strong enough to prove terminal success, include a warning plus native process-exit metadata. Do not silently discard the non-zero exit.

**Recommendation**: Mark `D-0001` as **accepted** (was: proposed).

## Implementation

No change needed — behavior is already correct.

## Next Action

Close this issue as **verified**. No further action required for the non-zero exit vs final event case. Monitor the related open issues for other process-boundary concerns.
