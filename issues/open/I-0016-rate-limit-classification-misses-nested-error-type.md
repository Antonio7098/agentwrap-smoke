# I-0016: Rate-limit Classification Misses Nested `error.type` Field

Status: open
Severity: critical
Area: rate-limit | fallback
Discovered: 2026-05-24
Related decisions: D-0004, D-0007
Related runs: `.agentwrap-logs/ratelimit-20260524-real-gpt55`, `R-20260524-019`

## Observation

When OpenCode returns a fatal error with an `error.type` of `rate_limit_error`, the rate-limit classifier (`classifyRateLimitData`) does not extract and check the nested `error.type` field. It only checks top-level `message`, `statusCode`, `responseBody`, and `responseHeaders`.

This means:

1. **Real rate-limit errors from the provider** (with `error.type: rate_limit_error`) are NOT classified as `rate_limit`, but fall through to `classifyFatalEventError` which returns `model_unavailable` or `runtime_exit`.
2. **Fallback is not triggered** because the error is classified as `model_unavailable` (permanent) rather than `rate_limit` (retryable).
3. **The user cannot distinguish** between a rate-limit that could have been retried/fallbacked vs a genuine unavailable model.

## Evidence

**Run**: `ratelimit-20260524-real-gpt55`  
**Primary model**: `opencode/gpt-5.5`  
**Expected**: `rate_limit` classification → PolicyRunner retries/fallbacks  
**Actual**: `model_unavailable` classification → PolicyRunner treats as permanent failure

From `results.json`:

```json
{
  "Status": "failed",
  "ErrorCategory": "runtime_exit",
  "Error": {
    "Category": "runtime_exit",
    "UserDetail": "OpenCode reported a fatal session error",
    "DebugDetail": "Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?"
  },
  "PolicyDecisionReason": "runtime exit"
}
```

The `DebugDetail` shows OpenCode suggests `gpt-5.5-pro`, meaning gpt-5.5 is a valid model name but the account is rate-limited/over quota. This is a **transient** failure that should trigger fallback, not a permanent `model_unavailable`.

The error payload from OpenCode contains nested `error.type: rate_limit_error`:

```json
{
  "type": "error",
  "data": {
    "message": "Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?",
    "error": {
      "type": "rate_limit_error",
      "message": "usage limit exceeded"
    }
  }
}
```

`classifyRateLimitData` extracts `message` (top-level) but never checks `data["error"]["type"]` for `"rate_limit_error"`.

## Root Cause

`opencode/rate_limit.go` — `classifyRateLimitData`:

```go
func classifyRateLimitData(op string, data map[string]any, runtimeCtx agentwrap.RuntimeContext) *rateLimitClassification {
    message := messageFrom(data)  // Only checks data["message"], not data["error"]["type"]
    body := stringValue(data["responseBody"])
    status, _ := intFromAny(firstNonNil(data["statusCode"], ...))
    // ...
    return classifyRateLimitDetails(op, message, body, headers, metadata, status, runtimeCtx)
    // Never checks data["error"]["type"] == "rate_limit_error"
}
```

The nested `error.type: rate_limit_error` is completely ignored.

## Expected Contract

When `data["error"]["type"] == "rate_limit_error"` (case-insensitive), the classifier MUST return `rate_limit` regardless of the message text. The `error.type` is the structural signal; the message is just human-readable context.

## Fixture Added

**Deterministic test fixture added:**

1. **fake-opencode.sh**: Added `RUN_MODE=rate_limit_nested` that emits the exact nested error structure:
   ```bash
   echo '{"type":"error","message":"Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?","error":{"type":"rate_limit_error","message":"usage limit exceeded","metadata":{"retry-after-ms":2000}}}' >&2
   exit 1
   ```
   Note: Top-level `message` does NOT contain "rate limit" to isolate structural detection.

2. **cmd/agentwrap-run/main.go**: Added `rate-limit-nested` command using PolicyRunner:
   - Primary: `RUN_MODE=rate_limit_nested`
   - Fallback: `RUN_MODE=final`
   - Expected: `error_category=rate_limit` (before fix: `runtime_exit`)

**Run record**: `R-20260524-019` — Confirms `error_category=runtime_exit` with current code (bug reproduced)

## Verification

1. **Unit test**: Add `TestClassifyRateLimitNestedErrorType` that passes an error event with `error.type: rate_limit_error` and expects `ErrorRateLimit` classification.
2. **Smoke test**: Run `./agentwrap-run rate-limit-nested` and verify fallback triggers with `error_category=rate_limit`.
3. **Real smoke**: Retry with a model that actually hits rate limit (not just model-not-found).

## Next Action

1. **Spike the fix in `opencode/rate_limit.go`**: Modify `classifyRateLimitData` to check `data["error"]["type"]`
2. **Add unit test** in the agentwrap adapter
3. **Run adapter tests**: `go test ./opencode/...`
4. **Verify smoke**: Run `./agentwrap-run rate-limit-nested` and verify `error_category=rate_limit`
