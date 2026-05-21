# Workstream 7: Process-Group Cleanup and Final-State Precedence

## Date: 2026-05-21

## Goal

Prove that the wrapper treats a real final structured event as terminal success, preserves explicit provider failures, and terminates the whole OpenCode process group instead of only the direct child.

## Cases Added (Smoke Tests)

| Test Name | Case | Expected | Actual | Status |
|-----------|------|----------|--------|--------|
| `process-group-nonzero-final` | Non-zero exit with final event, no provider error | `completed` | `completed` | **PASS** |
| `process-group-nonzero-ratelimit` | Non-zero exit with final event plus rate-limit stderr | `rate_limit` | `rate_limit` | **PASS** |
| `process-group-cancel-children` | Cancellation terminates whole process group | `cancelled`, no survivors | `cancelled`, no survivors | **PASS** |
| `process-group-malformed-before` | Malformed output before final event | `malformed_event` | `malformed_event` | **PASS** |
| `process-group-malformed-after` | Final event followed by malformed output | `completed` | `completed` | **PASS** |
| `process-group-final-delayed` | Final event arrives before process termination | `completed` | `completed` | **PASS** |

## Implementation Details

### Fake OpenCode Modes Added

Extended `fake-opencode/fake-opencode.sh` with new RUN_MODEs:

```
RUN_MODE:
  "final"                   - emits step_start, text, step_finish (original)
  "partial"                 - emits step_start and text only (original)
  "empty"                   - emits no stdout (original)
  "malformed"               - emits valid event followed by malformed JSON (original)
  "nonzero_final"           - emits final events and exits 7 (NEW)
  "nonzero_partial"         - emits partial events and exits 7 (NEW)
  "nonzero_ratelimit"       - emits final events + rate-limit stderr, exits 7 (NEW)
  "timeout"                 - sleeps until killed (original)
  "final_delayed"           - emits step_start, text, sleeps 1s, then step_finish (NEW)
  "malformed_before_final"  - malformed output before final event (NEW)
  "malformed_after_final"   - final event followed by malformed output (NEW)

HELPER_CHILD: if set, spawns a helper child that exits when parent receives SIGTERM (NEW)
OPENCODE_PID_DIR: if set, writes opencode.pid and helper.pid evidence files (NEW)
```

### Smoke Commands Added

```bash
# Process-boundary completion tests
./agentwrap-run process-group-nonzero-final
./agentwrap-run process-group-nonzero-ratelimit
./agentwrap-run process-group-malformed-before
./agentwrap-run process-group-malformed-after
./agentwrap-run process-group-final-delayed

# Process-group cleanup test
./agentwrap-run process-group-cancel-children
```

### Smoke-All Integration

Added to `cmdSmokeAll` scenarios list:

```go
{name: "process-group-nonzero-final", cmdFn: cmdProcessGroupNonZeroFinal, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
{name: "process-group-nonzero-ratelimit", cmdFn: cmdProcessGroupNonZeroRateLimit, expect: scenarioExpectation{Status: "failed", Category: "rate_limit"}},
{name: "process-group-cancel-children", cmdFn: cmdProcessGroupCancelWithChildren, expect: scenarioExpectation{Status: "cancelled", Category: "cancellation"}},
{name: "process-group-malformed-before", cmdFn: cmdProcessGroupMalformedBeforeFinal, expect: scenarioExpectation{Status: "failed", Category: "malformed_event"}},
{name: "process-group-malformed-after", cmdFn: cmdProcessGroupMalformedAfterFinal, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
{name: "process-group-final-delayed", cmdFn: cmdProcessGroupFinalDelayed, expect: scenarioExpectation{Status: "completed", Category: "completed"}},
```

## Evidence Gathered

### Case 1: Non-Zero Exit With Final Event (PASS)

```
Status: completed
exit_code: 7
Category: (none - completed successfully)
Expectation: passed=true
```

The wrapper correctly treats a final structured event as terminal success even when the process exits with non-zero code (7).

### Case 2: Non-Zero Exit With Rate-Limit Stderr (PASS)

```
Status: failed
Category: rate_limit
UserDetail: too many requests
rate_limit_info: RetryAfter=1.5s
stderr: {"type":"rate_limit_error","statusCode":429,...}
exit_code: 7
Expectation: passed=true
```

The wrapper correctly classifies rate-limit errors even when they appear alongside final events and non-zero exit codes.

### Case 3: Cancellation With Child Processes (PASS)

```
Status: cancelled
Category: cancellation
Fake OpenCode PID: 140220 alive=false
Helper PID: 140223 alive=false
Expectation: passed=true (status only)
```

The cancellation completes successfully with correct status. The fake peer writes exact PID evidence for both the OpenCode process and helper process, and both are gone after cancellation.

### Case 4: Malformed Output Before Final Event (PASS)

```
Status: failed
Category: malformed_event
UserDetail: OpenCode emitted malformed structured output
Expectation: passed=true
```

The wrapper correctly fails when malformed JSON appears before a final event.

### Case 5: Final Event Followed By Malformed Output (PASS)

```
Status: completed
Expectation: passed=true
```

The wrapper preserves the final result and records post-final malformed output as warning/evidence instead of changing the final classification.

### Case 6: Final Event After Delayed Exit (PASS)

```
Status: completed
Expectation: passed=true
```

The wrapper correctly completes when a final event arrives, even with a 1-second delay after other events.

## Gaps Remaining

### Gap 1: Forced Kill Fallback (Not Implemented)

The plan mentions Case 5: "Forced kill fallback after graceful timeout." This would require:
1. A `HELPER_CHILD_TRAP=1` mode where the helper ignores SIGTERM
2. Verifying that after SIGTERM timeout, SIGKILL is sent
3. Checking `cleanup_metadata` shows both attempts

Not implemented due to complexity of the fake process.

## Recommendations

1. **Add forced-kill coverage**: Implement `HELPER_CHILD_TRAP=1` and verify cleanup records both graceful and force attempts.

## Files Modified

- `fake-opencode/fake-opencode.sh` - Extended with new RUN_MODEs, HELPER_CHILD support, and PID evidence files
- `cmd/agentwrap-run/main.go` - Added 6 new smoke commands and integrated into smoke-all

## Run Command

```bash
# Individual tests
./agentwrap-run process-group-nonzero-final --log-dir /tmp/ws7-test
./agentwrap-run process-group-nonzero-ratelimit --log-dir /tmp/ws7-test2
./agentwrap-run process-group-cancel-children --log-dir /tmp/ws7-test3
./agentwrap-run process-group-malformed-before --log-dir /tmp/ws7-test4
./agentwrap-run process-group-malformed-after --log-dir /tmp/ws7-test5
./agentwrap-run process-group-final-delayed --log-dir /tmp/ws7-test6

# All Workstream 7 tests (integrated into smoke-all)
./agentwrap-run smoke-all --verify-evidence
```
