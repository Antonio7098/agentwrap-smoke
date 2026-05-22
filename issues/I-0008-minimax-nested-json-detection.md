# I-0008: MiniMax Nested JSON Rate-Limit Text Not Classified

Status: open
Severity: medium
Area: rate-limit
Discovered: 2026-05-21
Related decisions: none
Related runs: `.agentwrap-logs/ratelimit-20260521-120501`

## Observation

`classifyRateLimitText()` in `opencode/runtime.go` does not detect MiniMax rate-limit signals when they are nested inside a JSON `responseBody` field.

Real MiniMax evidence captured before provider reset:

```
service=llm providerID=minimax-coding-plan modelID=MiniMax-M2.7
statusCode=429
responseBody={"type":"error","error":{"type":"rate_limit_error","message":"usage limit exceeded, 5-hour usage limit reached for Token Plan Starter (1500/1500 used), resets at 2026-05-20T20:00:00Z (2056)"}}
```

The relevant test `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` fails because the nested JSON format has `responseBody` containing escaped JSON, and `classifyRateLimitText` only checks top-level strings for patterns like `usage limit exceeded`.

## Expected Contract

MiniMax `rate_limit_error` with `usage limit exceeded` should classify as `rate_limit` and populate `rate_limit_info` metadata including `RetryAfter`.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 2 Key Findings
- MiniMax OpenCode log excerpt (above)
- Test: `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` — FAIL

## Root Cause

`classifyRateLimitText()` uses simple string matching against stderr lines. When the rate-limit shape is embedded in a JSON `responseBody` field, the function does not extract or scan the nested JSON.

## Implementation

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

Fix by either:
1. Extracting `responseBody` JSON and scanning its decoded fields for rate-limit patterns, or
2. Adding the specific MiniMax error text patterns ("usage limit exceeded", "rate_limit_error") directly to the line-level pattern list (simpler but may produce false positives).

Re-run `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` after fix.
