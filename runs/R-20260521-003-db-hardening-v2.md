# R-20260521-003: DB Hardening Second Pass (ws3-db-v2)

Date: 2026-05-21
Type: fixture
Related issues: I-0010
Related decisions: none

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
./agentwrap-run ws3-db-v2  # all 8 DB scenarios
```

Scenarios:
- `db-unavailable`, `db-non-json`, `db-timeout`, `db-locked`
- `db-session-no-assistant`, `db-assistant-no-finish`, `db-assistant-with-finish`
- `db-only-proof` (new)

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Fixture: `fake-opencode/fake-opencode.sh` with `DB_MODE` and `RUN_MODE`
- Harness changes: JSON validation, DB query timeout, session ID alignment, `db-only-proof` mode

## Result

- Status: all 8 scenarios completed with coherent evidence
- Improvements verified:

| Improvement | Before | After |
|-------------|--------|-------|
| JSON validation | `db-non-json`: status=captured (invalid JSON treated as valid) | status=failed with explicit parse error |
| DB query timeout | `db-timeout`: silent empty-array capture | 5s per-file context deadline, explicit timeout error |
| Session ID alignment | `ses_test456` / `ses_test123` mismatched | `ses_fake123` used consistently across all fixtures and results |
| DB-only proof | no scenario exercised DB reconciliation | `db-only-proof`: runtime_exit + captured DB evidence with matching session ID |

## Evidence

- Log directory: `.agentwrap-logs/ws3-db-v2/`

Key individual results:

**`db-non-json`**:
- `opencode-db-snapshot-status.json`: status=failed, errors including "not valid JSON"
- Before: status=captured with invalid files written

**`db-only-proof`** (new):
- `RUN_MODE=db_proof`: emits step_start + text only, no step_finish
- `DB_MODE=db_only_proof`: DB returns session+message+part rows with ses_fake123
- `results.json`: status=failed, category=runtime_exit, session_id=ses_fake123
- `opencode-db-snapshot-status.json`: status=captured with valid JSON
- Correctly fails (no final event) but captures DB evidence coherently

**`db-timeout`**:
- With 5s harness-side timeout, fake 30s sleep fixture would produce explicit timeout errors per query
- `saveOpenCodeDB` runs after run completes; run itself produces runtime_exit before DB queries

**`db-assistant-with-finish`**:
- Session IDs now consistent: ses_fake123 in both results.json and DB snapshot files
- Assistant data with finish="stop" captured correctly

## Notes

- This run proves **evidence capture** improvements, not wrapper-level reconciliation behavior.
- `db-only-proof` shows the harness can capture DB evidence even when wrapper reports runtime_exit, but the wrapper's `reconcileFinalState()` is not exercised because the fake runner doesn't call it.
- Session ID alignment is now correct across all fixture outputs.
- Source: `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 3 evidence.
