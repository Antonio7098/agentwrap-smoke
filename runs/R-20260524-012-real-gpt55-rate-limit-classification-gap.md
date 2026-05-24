# R-20260524-012: Real Rate-Limit From `gpt-5.5` — Classification Gap Evidence

Date: 2026-05-24
Type: smoke | real
Related issues: `I-0016` (new), `I-0002`, `I-0011`
Related decisions: `D-0004`, `D-0007`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run rate-limit \
  --log-dir .agentwrap-logs/ratelimit-20260524-real-gpt55 \
  --primary-model "opencode/gpt-5.5" \
  --fallback-model "opencode/deepseek-v4-flash-free"
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Primary model: `opencode/gpt-5.5`
- Fallback model: `opencode/deepseek-v4-flash-free`
- Timestamp: 2026-05-24T14:36:46+01:00

## Result

| Attempt | Model                             | Status    | Error Category                          | Policy Decision      |
| ------- | --------------------------------- | --------- | --------------------------------------- | -------------------- |
| 1       | `opencode/gpt-5.5`                | failed    | `runtime_exit` (should be `rate_limit`) | fallback to deepseek |
| 2       | `opencode/deepseek-v4-flash-free` | completed | —                                       | fallback succeeded   |

**Overall**: `completed` (fallback worked)  
**Classification**: `runtime_exit` (should be `rate_limit`)  
**Fallback triggered**: yes (but for wrong reason — `runtime_exit`, not `rate_limit`)

## Evidence

- `.agentwrap-logs/ratelimit-20260524-real-gpt55/results.json`
- `.agentwrap-logs/ratelimit-20260524-real-gpt55/ratelimit.log`

Key finding from `results.json` attempt 1:

```json
{
  "Status": "failed",
  "ErrorCategory": "runtime_exit",
  "Error": {
    "Category": "runtime_exit",
    "Operation": "opencode event",
    "UserDetail": "OpenCode reported a fatal session error",
    "DebugDetail": "Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?"
  }
}
```

The `DebugDetail` suggests `gpt-5.5-pro`, confirming gpt-5.5 is a valid model name. The actual error is a **rate limit / quota exhaustion**, not a missing model. The OpenCode error payload contained nested:

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

The `error.type: rate_limit_error` was ignored by `classifyRateLimitData` because it only checks top-level fields, not nested `data["error"]["type"]`.

## Gap

`opencode/rate_limit.go` — `classifyRateLimitData` does not check `data["error"]["type"]` for `"rate_limit_error"`. The nested structure from OpenCode is silently ignored, causing the error to fall through to `classifyFatalEventError` which returns `runtime_exit` (and eventually `model_unavailable` due to the "Model not found" text).

This means:

1. PolicyRunner treats the failure as permanent (`model_unavailable`) rather than transient (`rate_limit`).
2. Fallback still triggers (because PolicyRunner has fallback configured), but for wrong semantic reasons.
3. Real rate-limit errors that could be retried are not correctly classified for diagnostic or retry-policy purposes.

## Notes

- Fallback succeeded despite wrong classification — PolicyRunner's fallback-alternative config triggers on any non-success, not just rate-limit.
- The semantic wrongness matters for: retry policy, observability, D-0007 auth-vs-rate-limit distinction, and user diagnostics.
- I-0016 created to track the root cause fix.
