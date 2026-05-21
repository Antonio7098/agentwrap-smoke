# Workstream 2: Deterministic Rate-Limit And Fallback Testing - Results

## Summary

Workstream 2 tests for rate-limit classification and fallback behavior have been implemented across unit tests, policy tests, and smoke commands. Most tests pass, but there are a few failures due to timing sensitivity issues in log classification that need addressing.

## Tests Added/Modified

### Unit Tests (agentwrap/opencode/runtime_test.go - uncommitted changes)

| Test | Status | Notes |
|------|--------|-------|
| TestClassifyExitErrorDetectsJSONRateLimit | PASS | Detects JSON-structured 429 rate limits |
| TestClassifyRateLimitDataParsesOpenCodeHeaders | PASS | Parses headers like retry-after-ms |
| TestProjectNativeFatalErrorPromotesRateLimit | PASS | Fatal event with 429 promotes to rate_limit |
| TestRunFatalEventStoresRateLimitMetadata | PASS | Stores rate_limit_info in metadata |
| TestRunExitRateLimitStoresRateLimitMetadata | PASS | Exit with 429 stderr stores metadata |
| TestClassifyExitErrorDetectsMaxRetryRateLimitMessage | PASS | Detects "maximum retry attempts reached for rate limit" |
| TestClassifyRateLimitTextDetectsMiniMaxUsageLimit | FAIL | Classification issue - message extraction problem |
| TestTimeoutClassifiesRecentOpenCodeLogRateLimit | FAIL | Log cutoff timing issue (started - 1min) |
| TestTimeoutClassifiesRecentOpenCodeDBCheckpointLog | FAIL | Log cutoff timing issue |

### Policy Tests (agentwrap/policy_test.go - existing, passing)

| Test | Status | Notes |
|------|--------|-------|
| TestPolicyRunnerFallbackThenRetriesOnFallbackTarget | PASS | Primary fails -> fallback -> retry on fallback |
| TestPolicyRunnerHonorsRateLimitRetryAfter | PASS | Rate limit triggers retry with proper backoff |

### Smoke Commands (agentwrap-smoke/cmd/agentwrap-run/main.go)

| Command | Status | Notes |
|---------|--------|-------|
| rate-limit | PASS | Real OpenCode fallback - primary fails, fallback completes |
| provider-429 | PASS | Documents expected category when 429 occurs |

## Evidence

### Real smoke test output (rate-limit command):
```
PolicyRunner configured with opencode/gpt-5.5 primary and opencode/deepseek-v4-flash-free fallback
Starting run...
Run started: ID=policy-1
Final Status: completed
Attempt 1 target=0 model=opencode/gpt-5.5 status=failed error=runtime_exit
Attempt 2 target=1 model=opencode/deepseek-v4-flash-free status=completed error=
```

### Unit test evidence (passing rate-limit tests):
```
=== RUN   TestClassifyExitErrorDetectsJSONRateLimit
--- PASS: TestClassifyExitErrorDetectsJSONRateLimit (0.00s)
=== RUN   TestRunFatalEventStoresRateLimitMetadata
--- PASS: TestRunFatalEventStoresRateLimitMetadata (0.00s)
=== RUN   TestRunExitRateLimitStoresRateLimitMetadata
--- PASS: TestRunExitRateLimitStoresRateLimitMetadata (0.00s)
```

## Failing Tests - Issues Identified

### 1. TestClassifyRateLimitTextDetectsMiniMaxUsageLimit
- **Issue**: classifyRateLimitText returns nil for the MiniMax log text
- **Root cause**: The text format has a nested error JSON in responseBody that may not be parsed correctly
- **Text tested**: `ERROR service=llm providerID=minimax-coding-plan modelID=MiniMax-M2.7 error={"error":{"name":"AI_APICallError"},"statusCode":429,"responseBody":"{\"type\":\"error\",\"error\":{\"type\":\"rate_limit_error\",\"message\":\"usage limit exceeded\"}}"} stream error`
- **Needs**: The nested responseBody JSON parsing may need adjustment in rate_limit.go

### 2. TestTimeoutClassifiesRecentOpenCodeLogRateLimit
- **Issue**: Log cutoff (started - 1 minute) may exclude the test log file
- **Current behavior**: Gets `ErrorTimeout` instead of `ErrorRateLimit`
- **Note**: The test itself documents: "CURRENT BEHAVIOR: got timeout instead of rate_limit. Log classification may have timing sensitivity issue."

### 3. TestTimeoutClassifiesRecentOpenCodeDBCheckpointLog
- **Issue**: Same timing sensitivity as above

## Gaps Remaining

1. **MiniMax rate-limit detection**: The nested JSON parsing in `classifyRateLimitText` needs verification
2. **Log classification timing**: The cutoff in `classifyRecentLogFailure` uses `started.Add(-1 * time.Minute)` which may be too aggressive
3. **Fake opencode fixture**: The plan mentioned creating a fake opencode executable in `/home/antonioborgerees/coding/agentwrap-smoke/fake-opencode/` - this was started but not completed

## Classification Status

From `rate_limit.go` classifyRateLimitDetails:
- HTTP 429 with quota exceeded -> nil (skip)
- HTTP 429 with rate_limit_error in body/message -> rate_limit
- "gousagelimiterror", "too_many_requests", "rate_limit_error" in body/message -> rate_limit
- Status 503/504/529 -> rate_limit (retryable)

The MiniMax text "usage limit exceeded" appears to not match the detection patterns, which is why TestClassifyRateLimitTextDetectsMiniMaxUsageLimit fails.

## Recommendations

1. **Fix MiniMax test**: The message "usage limit exceeded" doesn't match current patterns. Consider adding to detection or adjust test expectation.

2. **Fix log timing**: Extend the log cutoff window in `classifyRecentLogFailure` or adjust test log timestamp to be more recent.

3. **Complete fake-opencode fixture**: Create the deterministic fake opencode for smoke tests that need specific rate-limit shapes.

4. **Add policy fallback test for rate_limit specifically**: There's no direct test for "primary returns rate_limit, fallback completes" - the closest is `TestPolicyRunnerHonorsRateLimitRetryAfter` which tests retry, not fallback.

## Conclusion

Rate-limit classification and fallback infrastructure is in place and mostly working. Real smoke tests pass. Unit tests for standard rate-limit shapes pass. There are 3 failing tests that need either code fixes or test adjustments due to timing sensitivity in log classification.
