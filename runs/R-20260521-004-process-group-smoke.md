# R-20260521-004: Process-Group Smoke Tests

Date: 2026-05-21
Type: fixture
Related issues: I-0001, I-0005
Related decisions: D-0001

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run process-group-nonzero-final      # Non-zero exit (7) with final event → completed
./agentwrap-run process-group-nonzero-ratelimit  # Non-zero exit + rate-limit stderr → rate_limit
./agentwrap-run process-group-cancel-children    # Cancel with helper child → cancelled, no survivors
./agentwrap-run process-group-malformed-before   # Malformed before final event → malformed_event
./agentwrap-run process-group-malformed-after    # Final event then malformed → completed
./agentwrap-run process-group-final-delayed      # Delayed step_finish → completed
```

Fake OpenCode modes used:

```bash
RUN_MODE=nonzero_final         # Final events, exit 7
RUN_MODE=nonzero_ratelimit     # Final events + rate-limit stderr, exit 7
RUN_MODE=final_delayed         # Step start, text, sleep, step_finish
RUN_MODE=malformed_before_final # Malformed before final events
RUN_MODE=malformed_after_final  # Final events, then malformed
HELPER_CHILD=1                 # Spawn helper that exits on SIGTERM
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Fixture: `fake-opencode/fake-opencode.sh` with new RUN_MODEs and HELPER_CHILD
- Adapter: `/home/antonioborgerees/coding/agentwrap` with final-state precedence fix

## Result

| Scenario | Expected | Actual | Status |
|----------|----------|--------|--------|
| Non-zero exit + final event | `completed` | `completed` | PASS |
| Non-zero exit + rate-limit stderr | `rate_limit` | `rate_limit` | PASS |
| Cancel with child processes | `cancelled`, no survivors | `cancelled`, no survivors | PASS |
| Malformed before final event | `malformed_event` | `malformed_event` | PASS |
| Final event then malformed | `completed` | `completed` | PASS |
| Delayed final event | `completed` | `completed` | PASS |

## Evidence

- Process-group evidence from PID-file checks confirms no surviving child processes after cancellation.
- Malformed-after-final: wrapper preserves valid final event when malformed JSON appears later in stream. Scanner remains strict; run loop normalizes post-final decode errors after `sawFinal` is true.
- Final-state precedence confirmed: `sawFinal` takes precedence over `proc.ExitCode != 0`.

## Notes

- These tests use the fake-opencode fixture, not real OpenCode. They prove **wrapper logic correctness**, not real OpenCode integration.
- Source: `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 7.
