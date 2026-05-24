# I-0011: Deterministic Rate-Limit Fallback Fixture Missing

Status: verified
Severity: medium
Area: fallback
Discovered: 2026-05-21
Related decisions: none
Related runs: R-20260524-016

## Observation

Real provider rate limits are stateful and cannot be forced deterministically. The MiniMax targeted run on 2026-05-21 completed successfully without rate-limiting, so fallback was not exercised.

## Expected Contract

The harness should have a deterministic fixture (fake OpenCode executable on PATH) that can emit 429 / rate-limit shapes without relying on live provider account state. This fixture should be clearly labeled as a wrapper/process-contract test, not a real provider test.

## Implementation (Spike R-20260524-016)

### 1. fake-opencode.sh additions

Added two new `RUN_MODE` options:

```
"rate_limit"        - emits HTTP 429 rate-limit shape via stderr, exits 1 (triggers fallback)
"rate_limit_nested" - emits nested error.type: rate_limit_error shape (I-0016 test)
```

**rate_limit mode** emits:

```json
{
  "type": "error",
  "statusCode": 429,
  "message": "rate limit exceeded",
  "error": {
    "type": "rate_limit_error",
    "message": "OpenCode provider rate limit reached"
  }
}
```

**rate_limit_nested mode** emits:

```json
{
  "type": "error",
  "message": "rate limit exceeded",
  "error": {
    "type": "rate_limit_error",
    "metadata": { "retry-after-ms": 2000 }
  }
}
```

### 2. cmd/agentwrap-run/main.go addition

Added `rate-limit-fixture` command and `cmdRateLimitFixture` function.

**Command registration:**

```go
rateLimitFixtureCmd := &cobra.Command{
    Use:   "rate-limit-fixture",
    Short: "Test deterministic rate-limit fallback using fake-opencode (I-0011)",
    Args:  cobra.ExactArgs(0),
    Run:   cmdRateLimitFixture,
}
```

**PolicyRunner configuration:**

```go
runner := agentwrap.PolicyRunner{
    Runtime: primaryRuntime, // RUN_MODE=rate_limit
    Policy: agentwrap.BasicPolicy{
        MaxAttemptsPerTarget: 1,
        RetryRateLimits:      true,
    },
    Alternatives: []agentwrap.FallbackAlternative{
        {
            Name:    "rate-limit-fallback",
            Runtime: fallbackRuntime, // RUN_MODE=final
            Request: agentwrap.RunRequest{...},
        },
    },
}
```

## Verification (R-20260524-016)

```
=== I-0011: RATE-LIMIT FALLBACK FIXTURE TEST ===
Testing: Primary fake-opencode returns rate_limit -> fallback completes

Final Status: completed
Session ID: ses_fake123

Attempt Summary:
  Attempt 1: model=failed status=rate_limit error_category=rate_limit
    RateLimit info: provider=opencode model=opencode/fake-primary detail=rate limit exceeded
  Attempt 2: model=completed status=completed error_category=

=== VERIFICATION ===
PASS: Final status=completed (fallback succeeded)
PASS: 2 attempts recorded (primary + fallback)
PASS: First attempt error_category=rate_limit
PASS: Second attempt status=completed (fallback succeeded)

=== RESULT: ALL CHECKS PASSED ===
```

## Evidence

- Run ID: R-20260524-016
- Log directory: `.agentwrap-logs/rate-limit-fixture-20260524-...`
- Results file: `results.json`

## Next Action

1. Add `rate-limit-fixture` to smoke-all suite in `cmdSmokeAll`
2. Add `RUN_MODE=rate_limit_nested` test for I-0016 classifier fix
3. Consider adding nested error.type detection to rate_limit.go (I-0016)
