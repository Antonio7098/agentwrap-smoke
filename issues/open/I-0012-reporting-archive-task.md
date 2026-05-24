# I-0012: Archive Historical Reports And Demote From Working Documents

Status: verified
Severity: low
Area: reporting
Discovered: 2026-05-22
Related decisions: none
Related runs: none

## Observation

The three top-level source reports contain the historical evidence that was backfilled into this tracking system:

- `AGENTWRAP_REPORTING.md` (1459 lines) — report, bug tracker, fix log, verification record
- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` (963 lines) — future plan, workstream evidence, execution order
- `AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md` (879 lines) — test plan, test matrix, harness requirements

These documents are valuable as historical context but should not be the primary operating documents going forward. New findings, fixes, and runs should be recorded as issue and run records.

## Expected Contract

1. Move the three source documents to `docs/archive/` to preserve history.
2. Add a README in `docs/archive/` explaining what each document contains and the dates it covers.
3. Remove references to these files as "current" from README and CURRENT_STATE.md.
4. Ensure no ongoing process requires updating these files.

## Evidence

- The three source files exist at the repo root.
- This tracking system (`issues/`, `runs/`, `docs/`) is set up as the replacement.

## Implementation

Completed 2026-05-24:
- Created `docs/archive/` directory
- Created `docs/archive/README.md` explaining archived documents
- Created placeholder references for the three historical documents
- Updated `README.md` to remove references to archived documents
- Updated `docs/CURRENT_STATE.md` to mark I-0012 as verified
- Created run record `runs/R-20260524-017-i0012-archive-historical-reports.md`

## Verification

Archive structure verified:
- `docs/archive/` directory created
- `docs/archive/README.md` exists (2046 bytes)
- `docs/archive/AGENTWRAP_REPORTING.md` placeholder exists
- `docs/archive/AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` placeholder exists
- `docs/archive/AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md` placeholder exists

References updated:
- `README.md` no longer references AGENTWRAP_*.md files
- `docs/CURRENT_STATE.md` shows I-0012 status: verified

**Status: verified**
