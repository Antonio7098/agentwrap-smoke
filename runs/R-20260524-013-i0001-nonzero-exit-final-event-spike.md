# R-20260524-013: I-0001 Non-Zero Exit Versus Final Event Spike

Date: 2026-05-24
Type: smoke | fixture | unit-backed
Related issues: `I-0001`
Related decisions: `D-0001`

## Commands

**Unit test:**

```bash
cd /home/antonioborgerees/coding/agentwrap && go test -v ./opencode -run 'NonZeroExitWithFinalEventStillCompletes'
```

**Smoke test:**

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run process-group-nonzero-final
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: `opencode/fake` (fixture)
- Fake opencode: `fake-opencode/fake-opencode.sh` with `RUN_MODE=nonzero_final`

## Result

- **Status**: `completed`
- **Category**: (none)
- **Exit code**: 7 (preserved in `NativeMetadata.exit_code`)
- **Pass/fail**: ✅ PASS

### Evidence

- **Log directory**: `.agentwrap-logs/process-group-nonzero-final-20260524-150922/`
- **Results file**: `results.json`
- **Events**: 6 events including `step_finish` projected as `final_result`
- **Native metadata**:
  - `exit_code`: 7 (preserved, not discarded)
  - `native_terminal_evidence`: `step_finish`
  - `event_count`: 6
  - `event_categories`: `final_result=1, lifecycle=2, message=1, progress=1, session=1`
  - `native_event_types`: `lifecycle.transition=2, session.relationship=1, step_finish=1, step_start=1, text=1`

### Adapter Code Path (confirmed)

The `finalResult()` function in `/home/antonioborgerees/coding/agentwrap/opencode/runtime.go` handles this correctly:

```go
} else if r.sawFinal {
    if proc.Err != nil || proc.ExitCode != 0 {
        if exitErr := classifyExitError(proc, r.stderrBuffer.String()); exitErr.Category != agentwrap.ErrorRuntimeExit {
            sdkErr = exitErr
            status = agentwrap.StatusFailed
        } else {
            status = agentwrap.StatusCompleted   // ← non-runtime_exit wins
        }
    } else {
        status = agentwrap.StatusCompleted
    }
}
```

Since `classifyExitError` for a non-zero exit with no rate-limit/error text returns `ErrorRuntimeExit`, the status remains `completed` when `sawFinal=true`.

### Unit Test Evidence

`TestRunNonZeroExitWithFinalEventStillCompletes` in `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go` (line 1289):

- Uses `final.ndjson` fixture with `processResult{ExitCode: 1, Err: errors.New("exit status 1")}`
- Asserts `result.Status == agentwrap.StatusCompleted`
- Asserts `result.Err == nil`
- Asserts `waitErr == nil`
- Asserts `hasFinalResult == true`
- **Result**: ✅ PASS

## Findings

### I-0001 is VERIFIED (not a bug)

The suspected adapter bug from `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` — "non-zero exit overrides `sawFinal`" — is **already fixed**. The `finalResult()` function in `runtime.go` correctly gives `sawFinal` precedence over non-runtime-exit process failures.

The relevant logic flow when `sawFinal=true`:

1. If process error OR non-zero exit code exists
2. AND `classifyExitError` does NOT return `ErrorRuntimeExit` (e.g., it's a rate-limit or model error) → `failed`
3. OR if `classifyExitError` returns `ErrorRuntimeExit` (simple non-zero exit with no actionable error) → `completed`
4. If no process error and exit code 0 → `completed`

### D-0001 is CONFIRMED

The contract decision (pending D-0001) aligns with current adapter behavior:

- A **strong final structured completion event** (`sawFinal=true`) wins over a non-zero exit
- The non-zero exit is **preserved** as warning/native metadata (`exit_code`, `stderr`)
- Only if the non-zero exit has a **stronger classification** (rate-limit, model unavailable, auth failure) does the final event get overridden

### Contract Summary

| Scenario                                       | Exit Code | sawFinal | stderr          | Result                    |
| ---------------------------------------------- | --------- | -------- | --------------- | ------------------------- |
| Final event, clean exit                        | 0         | true     | —               | `completed`               |
| Final event, non-zero exit (runtime_exit only) | 7         | true     | empty           | `completed` (verified ✅) |
| Final event, non-zero exit, rate-limit stderr  | 7         | true     | rate-limit JSON | `failed` (rate_limit)     |
| No final event, non-zero exit                  | 7         | false    | —               | `failed` (runtime_exit)   |

## Notes

The non-zero exit with no provider error is preserved as `NativeMetadata` (`exit_code: 7`) rather than producing an error. This means callers can still observe the process exit code even when the status is `completed`.

The `D-0001` decision should be marked **accepted** and the issue `I-0001` should be updated to **verified** status.
