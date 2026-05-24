# I-0011: Deterministic Rate-Limit Fallback Fixture Missing

Status: open
Severity: medium
Area: fallback
Discovered: 2026-05-21
Related decisions: none
Related runs: `.agentwrap-logs/ratelimit-20260521-120501`

## Observation

Real provider rate limits are stateful and cannot be forced deterministically. The MiniMax targeted run on 2026-05-21 completed successfully without rate-limiting, so fallback was not exercised.

Current fallback evidence relies on:
1. **Unit tests**: Rate-limit text classification is covered, but `TestPolicyRunnerHonorsRateLimitRetryAfter` tests retry on the same target, not fallback to an alternative.
2. **Real smoke with invalid model/provider**: `fallback-invalid-model` and `fallback-invalid-provider` use invalid models, not real rate limits.
3. **Opportunistic MiniMax**: Only works when the MiniMax account is actually exhausted.

There is no deterministic smoke test for: "primary returns `rate_limit`, fallback completes with `completed`".

## Expected Contract

The harness should have a deterministic fixture (fake OpenCode executable on PATH) that can emit 429 / rate-limit shapes without relying on live provider account state. This fixture should be clearly labeled as a wrapper/process-contract test, not a real provider test.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 2
- `AGENTWRAP_REPORTING.md` "Missing Features" section
- `.agentwrap-logs/ratelimit-20260521-120501/ratelimit.log`: MiniMax returned `completed`, no rate limit exercised

## Root Cause

Testing fallback from real provider rate limits requires either:
- A controllable test provider that can be configured to return 429, or
- A fake `opencode` executable that produces rate-limit shapes deterministically.

Neither exists in the current harness. The `fake-opencode.sh` fixture exists but does not yet have a rate-limit emulation mode.

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

1. Add a `RUN_MODE=rate_limit` mode to `fake-opencode.sh` that emits final events on stdout and writes rate-limit shapes to stderr.
2. Add a `cmdFakeRateLimitFallback` smoke command that uses the fake fixture with `PolicyRunner`.
3. Add unit test: `TestPolicyRunnerFallbackFromRateLimit` — primary returns `rate_limit`, fallback completes.
