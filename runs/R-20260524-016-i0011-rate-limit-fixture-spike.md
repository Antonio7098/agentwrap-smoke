# R-20260524-016: I-0011 Rate-Limit Fallback Fixture Spike

**Date**: 2026-05-24
**Command**: `agentwrap-run rate-limit-fixture`
**Run ID**: R-20260524-016
**Related Issue**: I-0011
**Status**: PASS

## Summary

Spike implementation of deterministic rate-limit fallback fixture. Added `RUN_MODE=rate_limit` and `RUN_MODE=rate_limit_nested` to `fake-opencode.sh`, and implemented `rate-limit-fixture` command in `cmd/agentwrap-run/main.go`. Test passes: primary fake-opencode returns rate_limit, PolicyRunner falls back to secondary fake-opencode which completes successfully.

## Configuration

- **Primary Runtime**: fake-opencode with `RUN_MODE=rate_limit` (HTTP 429 stderr)
- **Fallback Runtime**: fake-opencode with `RUN_MODE=final` (step_start, text, step_finish)
- **Policy**: BasicPolicy with RetryRateLimits=true, MaxAttemptsPerTarget=1

## Run Command

```bash
PATH="$PWD/fake-opencode:$PATH" ./agentwrap-run rate-limit-fixture --log-dir /tmp/rate-limit-fixture-test
```

## Results

| Check                  | Expected   | Actual     | Status |
| ---------------------- | ---------- | ---------- | ------ |
| Final status           | completed  | completed  | PASS   |
| Attempt count          | 2          | 2          | PASS   |
| First attempt category | rate_limit | rate_limit | PASS   |
| Second attempt status  | completed  | completed  | PASS   |

## Evidence

- Log directory: `/tmp/rate-limit-fixture-test`
- Results file: `/tmp/rate-limit-fixture-test/results.json`
- Log file: `/tmp/rate-limit-fixture-test/rate-limit-fixture.log`

## Changes Made

1. **fake-opencode/fake-opencode.sh**: Added `rate_limit` and `rate_limit_nested` RUN_MODE options
2. **cmd/agentwrap-run/main.go**: Added `rate-limit-fixture` command and `cmdRateLimitFixture` function
3. **issues/open/I-0011-...**: Updated with spike implementation details

## Notes

- Rate-limit classification uses adapter's `classifyRateLimitDetails` function (read-only)
- Nested error.type: rate_limit_error shape added for I-0016 classifier fix testing
- Test demonstrates deterministic fallback without real provider rate-limit state
