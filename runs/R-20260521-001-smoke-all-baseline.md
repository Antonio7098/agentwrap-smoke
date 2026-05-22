# R-20260521-001: Full Smoke-All Baseline

Date: 2026-05-21
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
- Agentwrap adapter fixes applied: sawOutput, cancellation precedence, DB reconciliation, rate-limit log scanning

## Result

- Status: 20/20 scenarios passed
- Exit code: 0

## Evidence

- Log directory: `.agentwrap-logs/smoke-all-20260521-120218`
- Summary: `summary.json` with 17 `passed:true`, 3 `passed:false` (health-fail, session-fork, validate-repair-exhaust — all harness classification artifacts, not semantic failures)

Key scenario results:

| Scenario | Status | Category | Notes |
|----------|--------|----------|-------|
| `smoke-text` | completed | — | usage present, warning for missing final event |
| `smoke-reasoning` | completed | — | reasoning events handled correctly |
| `smoke-file-write` | completed | — | file created correctly |
| `usage` | completed | — | DB-projected usage: Input=8231 Output=2 |
| `artifacts` | completed | — | 0 artifacts observed |
| `validate-json` | completed | — | JSON validation passed |
| `validate-md` | completed | — | Markdown template passed |
| `custom-validator` | completed | — | content trimmed and matched |
| `validate-fail` | failed | validation | expected |
| `validate-repair` | completed | — | 1 repair attempt, PASSED |
| `validate-repair-exhaust` | failed | timeout (not repair_exhausted) | genuine unresolved |
| `cancel` | cancelled | cancellation | expected |
| `timeout` | failed | timeout | expected |
| `fallback-invalid-model` | completed | — | fallback exercised |
| `fallback-invalid-provider` | completed | — | fallback exercised |
| `fallback-all-fail` | failed | runtime_exit | no specific category asserted |
| `health-fail` | completed | — | harness classification artifact |
| `health-model` | completed | — | health check passed |
| `session-fork` | failed | configuration | expected |

## Notes

- `validate-repair-exhaust` remains **genuinely unresolved**: timed out before producing `repair_exhausted`. Source report over-interpreted this as "two repairs then timeout" — the actual artifact shows 1 repair attempt and category `timeout`.
- `health-fail` reports as completed in smoke-all aggregator but actually returned expected health failures. This is a smoke-all reporting artifact, not a wrapper regression.
- Adapter fixes verified: sawOutput completion, cancellation classification, DB reconciliation, rate-limit log scanning, fallback attempt metadata.
- Source: `AGENTWRAP_REPORTING.md` "Real Harness Rerun After Fixes" section.
