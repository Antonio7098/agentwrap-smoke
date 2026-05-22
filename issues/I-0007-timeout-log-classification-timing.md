# I-0007: Timeout Log Classification Timing Sensitivity

Status: open
Severity: medium
Area: rate-limit
Discovered: 2026-05-21
Related decisions: none
Related runs: none yet in `runs/`

## Observation

`classifyRecentLogFailure()` in `opencode/runtime.go` uses `started.Add(-1 * time.Minute)` as a log file modification-time cutoff when scanning for provider rate-limit or OpenCode DB checkpoint evidence on timeout.

Tests that create log files near this cutoff boundary produce timing-sensitive failures:

| Test | Expected | Actual | Status |
|------|----------|--------|--------|
| `TestTimeoutClassifiesRecentOpenCodeLogRateLimit` | `rate_limit` | `timeout` | FAIL (log cutoff timing) |
| `TestTimeoutClassifiesRecentOpenCodeDBCheckpointLog` | `runtime_unavailable` | `timeout` | FAIL (log cutoff timing) |

The 1-minute cutoff is too tight for tests where log modtime is set programmatically near the boundary. In real environments, OpenCode logs have real filesystem timestamps, but the cutoff still risks missing logs when runs complete very quickly or when the system clock and log timestamps disagree.

## Expected Contract

The log classifier should match by content (session ID, model, provider) rather than relying primarily on modtime. The modtime cutoff should be a safety bound, not the primary filter.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 1 Case 7, Workstream 2 Key Findings
- Unit test files in `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go`

## Root Cause

`classifyRecentLogFailure()` was designed to avoid matching stale logs from unrelated runs. The modtime cutoff was chosen too tightly. The function should match by session ID or model name first, then use modtime as a fallback filter.

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

Fix the classification timing by either extending the modtime window (e.g., from `-1 minute` to `-5 minutes`) or by adding content-based matching (session ID, model name) as the primary filter before timestamp.
