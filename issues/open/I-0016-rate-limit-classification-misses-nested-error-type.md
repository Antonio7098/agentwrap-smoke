# I-0016: Rate-limit Classification Misses Nested `error.type` Field

Status: open
Severity: critical
Area: rate-limit | fallback
Discovered: 2026-05-24
Related decisions: D-0004, D-0007
Related runs: `.agentwrap-logs/ratelimit-20260524-real-gpt55`

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

## Implementation

In `classifyRateLimitData`:

1. Before any text-based matching, check if `data["error"].(map[string]any)["type"]` is `"rate_limit_error"` (case-insensitive).
2. If so, extract the error's `message` and `responseHeaders` from the nested `error` object.
3. Return `rate_limit` classification using those extracted values.

```go
// At start of classifyRateLimitData:
if errObj, ok := data["error"].(map[string]any); ok {
    if strings.EqualFold(stringValue(errObj["type"]), "rate_limit_error") {
        // Extract message from error object
        errMsg := stringValue(errObj["message"])
        errHeaders := headerMapFromAny(errObj["responseHeaders"])
        errBody := stringValue(errObj["body"])
        // Build rate-limit classification from error object data
        status, _ := intFromAny(firstNonNil(data["statusCode"], errObj["status"], errObj["code"]))
        metadata := stringMapFromAny(errObj["metadata"])
        info := rateLimitInfoFrom(errHeaders, metadata, runtimeCtx, errBody, errMsg, "error.type")
        return &rateLimitClassification{
            err: agentwrap.NewError(agentwrap.ErrorRateLimit, op, userRateLimitDetail(errBody, errMsg), nil,
                rateLimitErrorOptions(status, errHeaders, errBody, metadata, runtimeCtx, info)...),
            info: info,
        }
    }
}
```

## Verification

1. **Unit test**: Add `TestClassifyRateLimitNestedErrorType` that passes an error event with `error.type: rate_limit_error` and expects `ErrorRateLimit` classification.
2. **Smoke test**: Add a `fake-opencode` mode that emits `RUN_MODE=rate_limit_nested` with the nested structure. Run `rate-limit` command and verify fallback triggers.
3. **Real smoke**: Retry with a model that actually hits rate limit (not just model-not-found).

## Next Action

1. Spike the fix in `opencode/rate_limit.go`
2. Add unit test
3. Run adapter tests: `go test ./opencode/...`
4. Add fake-opencode rate-limit mode and smoke test
