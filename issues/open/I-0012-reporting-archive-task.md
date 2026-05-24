# I-0012: Archive Historical Reports And Demote From Working Documents

Status: open
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

Not yet done.

## Verification

After archiving, the repo root should contain only `docs/`, `issues/`, `runs/`, `templates/`, code directories, and configuration files. The three AGENTWRAP_* files should be in `docs/archive/`.

## Next Action

Create `docs/archive/` and move the three files there with a dated readme.
