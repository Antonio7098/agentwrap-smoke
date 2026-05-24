# R-20260524-019: I-0016 Nested error.type Fixture Spike

Date: 2026-05-24
Type: smoke | fixture | deterministic
Related issues: `I-0016` (nested error.type rate-limit classification)
Related decisions: `D-0004`, `D-0007`

## Command

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run rate-limit-nested --log-dir .agentwrap-logs/rate-limit-nested-20260524-002
```

## Result

| Attempt | Model                    | Status      | Error Category | Policy Decision |
| ------- | ------------------------ | ----------- | -------------- | --------------- |
| 1       | `opencode/fake-primary`  | `failed`    | `runtime_exit` | fallback        |
| 2       | `opencode/fake-fallback` | `completed` | —              | succeeded       |

**Overall**: `completed` (fallback succeeded, but for wrong reason)  
**Classification**: `runtime_exit` (should be `rate_limit`)  
**Fallback triggered**: yes (but PolicyRunner fallback triggers on any non-success, not just `rate_limit`)

## Evidence

Log: `.agentwrap-logs/rate-limit-nested-20260524-002/rate-limit-nested.log`

Fake-opencode emitted the exact nested error structure from real R-20260524-012:

```json
{
  "type": "error",
  "message": "Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?",
  "error": {
    "type": "rate_limit_error",
    "message": "usage limit exceeded",
    "metadata": {
      "retry-after-ms": 2000
    }
  }
}
```

Note: The top-level `message` does NOT contain "rate limit" text — it says "Model not found". This isolates the structural `error.type` signal from text-based matching.

## Gap Confirmed

`opencode/rate_limit.go` — `classifyRateLimitData` function:

1. **Extracts `message` from top-level `data["message"]`** — gets "Model not found..." text
2. **Does NOT check `data["error"]["type"]`** for `"rate_limit_error"`
3. **Falls through to `classifyRateLimitDetails`** which only does text-based matching
4. **Result**: `runtime_exit` because "Model not found" text doesn't match rate-limit patterns

The nested `error.type: rate_limit_error` is completely ignored.

## Why Fallback Still Succeeded

PolicyRunner's fallback triggers on ANY non-success status from the primary runtime, not just `rate_limit` classification. This masks the classification bug in real runs (as seen in R-20260524-012).

However, the semantic wrongness matters for:

1. **Observability**: `error_category=runtime_exit` instead of `rate_limit` in logs/metrics
2. **Retry policy**: PolicyRunner should retry transient `rate_limit` errors before fallback
3. **D-0007 distinction**: Cannot distinguish auth failures from rate-limit errors
4. **User diagnostics**: UserDetail shows "Model not found" instead of "usage limit exceeded"

## Fixture Added

1. **fake-opencode.sh**: Added `RUN_MODE=rate_limit_nested` that emits nested `error.type: rate_limit_error`
2. **cmd/agentwrap-run/main.go**: Added `rate-limit-nested` command using PolicyRunner with deterministic fixtures

The fixture now reliably reproduces the bug for regression testing.

## Next Action

**Fix `classifyRateLimitData` in `/home/antonioborgerees/coding/agentwrap/opencode/rate_limit.go`**:

1. Before text-based matching, check if `data["error"].(map[string]any)["type"]` is `"rate_limit_error"` (case-insensitive)
2. If so, extract the nested error's `message`, `headers`, `body`, `metadata`
3. Return `rate_limit` classification using those extracted values
