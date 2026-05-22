# I-0003: Reporting Sprawl

Status: open
Severity: medium
Area: reporting
Discovered: 2026-05-22
Related decisions: none
Related runs: none

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
- `issues/README.md`
- `runs/README.md`
- `templates/`

## Implementation

Initial tracking structure created.

## Verification

Manual file review only.

## Next Action

Backfill the most important open gaps from `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` as issue records before doing further adapter work.

