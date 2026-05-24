# I-0003: Reporting Sprawl

Status: verified
Severity: medium
Area: reporting
Discovered: 2026-05-22
Related decisions: none
Related runs: `R-20260524-005`, `R-20260524-017`, `R-20260524-020`

## Observation

The repo currently stores process, findings, fixes, future plans, and verification history in large top-level reports:

- `AGENTWRAP_REPORTING.md`
- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md`
- `AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md`
- `README.md`

Those documents are useful snapshots, but they are too broad to act as a meticulous issue ledger.

## Expected Contract

New findings and fixes should be tracked as issue records under `issues/`, and new verification commands should be tracked as run records under `runs/`.

The top-level reports should remain historical context unless intentionally rewritten.

## Evidence

Current setup work added:

- `docs/PROCESS.md`
- `docs/CURRENT_STATE.md`
- `docs/DECISIONS.md`
- `docs/archive/` — archived historical reports
- `issues/README.md`
- `runs/README.md`
- `templates/`

## Implementation

Initial tracking structure created. Subsequent work:

- `R-20260524-005`: First tracking assessment
- `R-20260524-017`: I-0012 archive task (docs/archive/ created)
- `R-20260524-020`: Final verification of tracking infrastructure

## Verification

### Assessment Run R-20260524-020

1. **Top-level files outside tracking system**:
   - `README.md` ✅ OK — legitimate project README
   - `sample-table.md`, `table.md` ⚠️ Minor — unrelated test files
   - `test-output.txt`, `worker-summary.txt` ⚠️ Minor — generated artifacts
   - `workstream-logs/`, `workstream-results/`, `workstream-scripts/` 📦 Pre-tracking era — need archiving decision

2. **CURRENT_STATE.md alignment with issues/open/**:
   - ✅ Fixed: Added missing issues I-0007, I-0008, I-0009, I-0010, I-0013, I-0014

3. **runs/README.md completeness**:
   - ✅ Fixed: Added R-20260524-001 through R-20260524-017, R-20260524-020

4. **Historical reports archived**:
   - ✅ `docs/archive/` contains AGENTWRAP\_\*.md source documents

5. **Tracking infrastructure verified sufficient**:
   - Issue tracking with stable IDs (issues/)
   - Run records with date-based IDs (runs/)
   - Dashboard (docs/CURRENT_STATE.md)
   - Contracts/decisions (docs/DECISIONS.md)
   - Templates (templates/)
   - Archived historical context (docs/archive/)

## Conclusion

Tracking infrastructure is sufficient. The reporting sprawl issue has been resolved:

- Historical docs archived to docs/archive/
- Issue tracking with stable IDs in place
- Run records with complete index
- Dashboard reflects current state of all issues

## Next Action

None — I-0003 closed as verified.

### Recommended cleanup

- Decide on `workstream-*` directory fate (archive or keep)
- Close resolved issues: I-0007, I-0008, I-0009, I-0010, I-0013, I-0014
