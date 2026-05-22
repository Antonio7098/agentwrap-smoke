# R-20260521-002: MiniMax Targeted Rate-Limit Run

Date: 2026-05-21
Type: smoke
Related issues: I-0002, I-0008, I-0011
Related decisions: none

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run rate-limit \
  --primary-model minimax-coding-plan/MiniMax-M2.7 \
  --fallback-model opencode/deepseek-v4-flash-free
```

Prior observation before MiniMax account reset (run on 2026-05-20):

```bash
./agentwrap-run usage --model minimax-coding-plan/MiniMax-M2.7
# Result: timeout (not rate_limit) — the trigger for this investigation
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Models: `minimax-coding-plan/MiniMax-M2.7` (primary), `opencode/deepseek-v4-flash-free` (fallback)

## Result

- Status: completed (inconclusive for fallback testing)
- Category: N/A
- Exit code: 0
- Fallback exercised: no (primary completed before rate limit hit)

Evidence from MiniMax rate-limit event captured before account reset:

```
service=llm providerID=minimax-coding-plan modelID=MiniMax-M2.7
statusCode=429
responseBody={"type":"error","error":{"type":"rate_limit_error","message":"usage limit exceeded, 5-hour usage limit reached for Token Plan Starter (1500/1500 used), resets at 2026-05-20T20:00:00Z (2056)"}}
```

Observed wrapper behavior before fix:

```
./agentwrap-run usage --model minimax-coding-plan/MiniMax-M2.7
Wait error: opencode run: timeout: OpenCode run timed out
Final Status: failed
```

Diagnosis: OpenCode wrote provider 429 to its internal log. The CLI process did not exit promptly. The wrapper hit timeout and reported `timeout`, losing the actionable `rate_limit` cause. Fix applied: timeout path now scans recent OpenCode logs for rate-limit shapes.

## Evidence

- Log directory: `.agentwrap-logs/ratelimit-20260521-120501`
- `results.json`: final status completed, attempt 1 completed, no fallback
- MiniMax rate-limit evidence: captured in `AGENTWRAP_REPORTING.md` from prior run
- Unit coverage: `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` (FAIL — nested JSON not parsed)

## Notes

- **Inconclusive for fallback**: MiniMax account reset, so this run did not exercise the fallback path.
- **Unit gap confirmed**: `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` fails because nested JSON `responseBody` is not parsed.
- **Timeout fix verified**: The timeout→log-scanning path is unit-tested but not exercised in this real run.
- Source: `AGENTWRAP_REPORTING.md` MiniMax section, `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 2.
