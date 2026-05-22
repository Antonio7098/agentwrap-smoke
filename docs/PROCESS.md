# agentwrap Smoke Tracking Process

This repo is an evidence ledger for `agentwrap` behavior against real and simulated OpenCode process boundaries.

The rule is simple: every important observation gets a stable issue ID, and every verification run gets a stable run ID. Do not bury new facts in long narrative reports without adding or updating the corresponding issue/run record.

## Tracking Model

- `issues/`: unified finding, decision, fix, and verification records.
- `runs/`: unit, smoke, manual, and provider run records.
- `docs/CURRENT_STATE.md`: compact dashboard for the latest known state.
- `docs/DECISIONS.md`: behavior contracts that affect future classification.
- `templates/`: copyable record formats.

Use "issue" broadly. An issue can be a bug, ambiguity, contract decision, test gap, harness gap, OpenCode behavior, or monitoring item.

## Standard Loop

1. Run the narrowest scenario that shows the behavior.
2. Create a run record in `runs/`.
3. If the behavior is wrong, ambiguous, or worth monitoring, create or update an issue in `issues/`.
4. Attach raw evidence paths: `results.json`, scenario log, `events.jsonl`, DB snapshots, OpenCode logs.
5. Decide the expected contract if it is unclear.
6. Implement the smallest fix or regression test.
7. Re-run unit tests and the relevant real smoke command.
8. Update the issue status, verification section, and `docs/CURRENT_STATE.md`.

## IDs

Use monotonically increasing IDs:

- Issues: `I-0001`, `I-0002`, ...
- Runs: `R-YYYYMMDD-NNN`, for example `R-20260522-001`.
- Decisions: `D-0001`, `D-0002`, ...

Do not recycle IDs, even if an issue is closed or deleted from scope.

## Issue Statuses

- `open`: observed and not yet fixed or contractually resolved.
- `in_progress`: implementation or test work has started.
- `blocked`: cannot proceed without external state, provider state, or a contract decision.
- `monitoring`: mitigated but needs more real-run evidence.
- `verified`: fixed and verified by at least one relevant run.
- `wontfix`: understood and intentionally not changed.

## Evidence Standard

An issue is not actionable unless it names the evidence that created it.

Minimum evidence:

- command
- timestamp or run ID
- log directory
- observed status/category
- expected status/category or contract question

When stdout is incomplete or contradictory, also record:

- `opencode-session.json`
- `opencode-messages.json`
- `opencode-parts.json`
- `opencode-db-snapshot-status.json`
- relevant OpenCode log excerpt location

## What Belongs In Current State

`docs/CURRENT_STATE.md` is a dashboard, not a report. Keep it short:

- latest verified baseline
- open high-priority issues
- active contract decisions
- recently verified fixes
- next recommended run

Move detailed investigation into issue and run records.

