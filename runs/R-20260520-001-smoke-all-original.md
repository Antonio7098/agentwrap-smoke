# R-20260520-001: Original Smoke-All Run (Pre-Fixes)

Date: 2026-05-20
Type: smoke
Related issues: I-0001, I-0005, I-0006, I-0009
Related decisions: D-0001, D-0003

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run smoke-all --model opencode/deepseek-v4-flash-free
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: `opencode/deepseek-v4-flash-free`
- Adapter: pre-sawOutput, pre-cancellation-precedence fixes

## Result

- Status: 17 passed, 3 passed:false (harness classification artifacts)
- Exit code: 0

## Evidence

- Log directory: `.agentwrap-logs/smoke-all-20260520-212857`
- Summary: `summary.json` with line-delimited results

Three `passed:false` entries:

1. **`validate-repair-exhaust`**: genuine unresolved — timed out before exhaustion.
2. **`health-fail`**: harness classification artifact — expected health failures were reported.
3. **`session-fork`**: harness classification artifact — expected unsupported cap error was reported.

Confirmed good scenarios (17):

- `smoke-text`: completed with usage, warning for missing final event
- `smoke-reasoning`: completed
- `smoke-file-write`: completed
- `usage`: completed with DB-projected usage
- `artifacts`: completed
- `fixed-backoff`: completed
- `validate-json`, `validate-md`, `custom-validator`: completed
- `validate-fail`: failed with `validation`
- `validate-repair`: completed (1 repair attempt)
- `cancel`: cancelled with `cancellation`
- `timeout`: failed with `timeout`
- `fallback-invalid-model`: completed via fallback
- `fallback-invalid-provider`: completed via fallback
- `fallback-all-fail`: failed
- `health-model`: completed

## Notes

- This is the **authoritative pre-fix baseline**. The 2026-05-21 rerun was partially invalidated by OpenCode DB checkpoint failures.
- `validate-repair-exhaust` artifact shows 1 repair attempt and category `timeout`, not `repair_exhausted`. The original report text over-interpreted this scenario's behavior.
- Source: `AGENTWRAP_REPORTING.md` "Real OpenCode Robustness Test Results (2026-05-20)" section.
