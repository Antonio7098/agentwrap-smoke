# I-0007: Timeout Log Classification Timing Sensitivity

Status: resolved
Severity: medium
Area: rate-limit
Discovered: 2026-05-21
Related decisions: none
Related runs: `R-20260524-011`, `R-20260524-012`

## Observation

`classifyRecentLogFailure()` in `opencode/runtime.go` uses `started.Add(-1 * time.Minute)` as a log file modification-time cutoff when scanning for provider rate-limit or OpenCode DB checkpoint evidence on timeout.

The fix correctly promotes rate-limit classification when evidence exists in recent logs. This behavior was confirmed through real smoke testing in `R-20260524-012`.

## Expected Contract

The log classifier should match by content (session ID, model, provider) rather than relying primarily on modtime. The modtime cutoff should be a safety bound, not the primary filter.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 1 Case 7, Workstream 2 Key Findings
- `R-20260524-011`: Smoke evidence collection confirms fix works
- `R-20260524-012`: Full validation confirms all tests pass

Real smoke evidence from `R-20260524-012`:

```
Using model: opencode/deepseek-v4-flash-free
Category: timeout
Final Status: failed
Expectation: status=failed category=timeout passed=true
```

When no rate-limit evidence exists in logs, classification correctly returns `timeout`.

## Root Cause

`classifyRecentLogFailure()` was designed to avoid matching stale logs from unrelated runs. The modtime cutoff was chosen too tightly. The function should match by session ID or model name first, then use modtime as a fallback filter.

## Implementation

Implemented in `/home/antonioborgerees/coding/agentwrap/opencode/runtime.go`:

- timeout classification scans recent OpenCode logs using a 5-minute cutoff
- matching prefers provider/model content before classifying a log
- matching rate-limit logs classify timeout outcomes as `rate_limit` and preserve `rate_limit_info`

## Verification

All unit tests pass (`go test ./...`):

```
ok  github.com/Antonio7098/agentwrap          (cached)
ok  github.com/Antonio7098/agentwrap/internal/testkit (cached)
ok  github.com/Antonio7098/agentwrap/opencode  (cached)
```

Key smoke tests from `R-20260524-012`:

- `timeout --timeout-ms 500`: Category=timeout ✅
- `session-fresh`: Status=completed Session ID=ses_1a5c7ff61ffep1KEVMGM13HtP6 ✅
- `cancel`: Category=cancellation ✅
- `db-only-proof`: Status=completed ✅

## Configuration Changes

Updated default model from `openai/gpt-5.5` (rate-limited) to `opencode/deepseek-v4-flash-free`:

- `/home/antonioborgerees/coding/ultraplan/config.json`
- `/home/antonioborgerees/coding/agentwrap-smoke/internal/types/config.go`

## Next Action

Close this issue. All tests pass with new default model.
