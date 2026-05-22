# R-20260521-005: Verify-Evidence Against Smoke-All Baseline

Date: 2026-05-21
Type: fixture
Related issues: I-0003
Related decisions: none

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run verify-evidence .agentwrap-logs/smoke-all-20260521-120218 [--strict]
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Verified against: `.agentwrap-logs/smoke-all-20260521-120218`

## Result

**Errors found (13):**

| Issue | Count | Details |
|-------|-------|---------|
| Missing `opencode-db-snapshot-status.json` | 7 scenarios | artifacts, custom-validator, fallback-all-fail, fixed-backoff, usage, validate-json, validate-md |
| Missing status in `results.json` | 3 scenarios | health-fail, health-model, session-fork (no RunResult produced) |
| Missing DB snapshot status file | same 3 scenarios | health-fail, health-model, session-fork |

**Warnings (6):**

| Issue | Count | Details |
|-------|-------|---------|
| Missing native_metadata | 3 | health-fail, health-model, session-fork |
| Missing cleanup_metadata | 3 | health-fail, health-model, session-fork |

**Info (16):**

| Issue | Count | Details |
|-------|-------|---------|
| `events.jsonl` missing | most scenarios | ObservingRuntime used but no event capture to disk (except artifacts) |

## Notes

- Root causes identified:
  1. **health-fail, health-model, session-fork**: call `saveResults()` with empty `RunResult{}` — they don't create runs.
  2. **Missing DB snapshots**: `saveOpenCodeDB()` called only in subset of scenarios (smoke-text, smoke-reasoning, smoke-file-write, cancel, timeout, validate-fail, validate-repair, validate-repair-exhaust, fallback-invalid-model, fallback-invalid-provider).
  3. **events.jsonl not written**: harness only writes events from `cmdRun()` goroutine; individual smoke commands don't set up event capture.

- These findings motivated Workstream 6 (observability and evidence assertions).
- Source: `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 6.
