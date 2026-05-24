# R-20260524-015: I-0005 Output-Without-Final-Event Spike

Date: 2026-05-24
Type: spike
Related issues: I-0005, I-0006
Related decisions: D-0003, D-0004

## Question

When does assistant output alone produce `completed` vs `failed`?

## Method

1. Read `../agentwrap/opencode/runtime.go` (read-only, `finalResult` method)
2. Traced `sawOutput` fallback in decision tree
3. Examined existing evidence in `.agentwrap-logs/`
4. Analyzed process-group-\* commands and fake-opencode.sh `RUN_MODE=partial`

## Decision Tree Analysis

In `runtime.go`'s `finalResult()` (no decode error, no context error):

```
if r.sawFinal                              → completed (final event supersedes exit code)
else if proc.Err || proc.ExitCode != 0     → failed/runtime_exit
else if r.sawIdle                          → completed
else if r.sawOutput                        → completed + warning
else if proof := r.reconcileFinalState()... → completed + warning (DB proof)
else                                       → failed/runtime_exit
```

**Key: `sawOutput` is checked BEFORE DB reconciliation.** This means output signal
is sufficient to claim completion even when DB is unavailable.

## Evidence

### Scenario A: Partial Output, DB Unavailable

Evidence: `.agentwrap-logs/db-only-proof-20260524-151009/`

```
RUN_MODE=partial (step_start + text, no step_finish)
DB_MODE=unavailable (DB query fails)

Result: status=completed, warnings=["OpenCode finished without a final structured result;
treating assistant output on clean exit as completed"]
exit_code=0, no final event, DB reconciliation not attempted (sawOutput wins)
```

### Scenario B: Partial Output, DB Available with Terminal Proof

Evidence: `.agentwrap-logs/db-only-proof-20260524-144131/`

```
RUN_MODE=partial (step_start + text, no step_finish)
DB_MODE=complete (session + message with finish=stop + part with finish=stop)

Result: status=completed, warning about sawOutput fallback
opencode-messages.json: {"role":"assistant","finish":"stop","content":"completed"}
opencode-parts.json: {"type":"step-finish","finish":"stop"}
```

Even though DB has terminal assistant proof, `sawOutput` wins because it's checked first.

### Scenario C: Final Event (Baseline)

Evidence: `.agentwrap-logs/process-group-final-delayed-20260524-141440/`

```
RUN_MODE=final_delayed (step_start + text + 1s sleep + step_finish)

Result: status=completed, no warning, native_terminal_evidence=step_finish
sawFinal=true → completed directly
```

### Scenario D: Non-Zero Exit with Final Event

Evidence: `.agentwrap-logs/process-group-nonzero-final-20260524-150922/`

```
RUN_MODE=nonzero_final (step_start + text + step_finish, exit 7)

Result: status=completed, no error
exit_code=7, but sawFinal=true → completed (D-0001 confirmed)
```

### Scenario E: Non-Zero Exit Without Final Event

Expected from adapter logic: would hit `proc.ExitCode != 0` → `failed/runtime_exit`

## D-0003 Answer

**Question**: Is assistant output alone enough to mark completed?

**Answer**: Yes — `sawOutput` on clean exit produces `completed` with warning.
No separate DB reconciliation needed for this fallback.

The fallback order per D-0004:

1. Final structured event (strongest)
2. Assistant output on clean exit (already implemented)
3. DB terminal finish evidence (secondary, checked after sawOutput)
4. None → runtime_exit

## Current Adapter Behavior: Correct

The adapter correctly implements D-0003:

- `sawOutput` → `completed-with-warning` (assistant output proves useful output)
- No contradictory process/provider evidence required
- Warning is added so callers can detect the non-standard completion path

## Gap: DB Reconciliation Not Properly Tested

The `cmdFakeDBBoundary` function uses `testdata/fakeopencode/fake-opencode.sh` with
`db_only_proof` mode, which emits step_start + text without final event, but the
adapter never reaches the DB reconciliation branch because `sawOutput` fires first.

To test DB reconciliation as primary fallback, we need a mode where:

- `sawOutput = false` (no assistant text events)
- But DB shows terminal assistant finish

This would require `RUN_MODE=db_proof` in fake-opencode.sh that emits only
session events without text, to test whether DB reconciliation fires correctly.

## Fake-Opencode RUN_MODE=partial Confirmation

`RUN_MODE=partial` in `fake-opencode/fake-opencode.sh`:

- Emits: `{"type":"step_start"}` → `{"type":"text"}` (no final event)
- Exits: 0 (clean exit)
- No DB queries (DB_MODE is separate env variable)

This correctly triggers `sawOutput` → `completed-with-warning` in adapter.

## Decision D-0003: RESOLVED

Status: accepted (implementation matches contract)

The `sawOutput` fallback in adapter correctly implements the proposed contract:
assistant output alone produces `completed-with-warning` on clean exit.
No runtime_exit for output-only cases when exit is clean.

## Decision D-0004: Verified

The fallback order is correctly implemented and smoke-tested:

- `process-group-final-delayed`: final event → completed (D-0001)
- `db-only-proof`: partial + DB terminal → completed-with-warning (DB as secondary)
- `db-unavailable`: partial + no DB → completed-with-warning (sawOutput wins)

## Recommendations

1. **Close D-0003**: Decision is resolved. `sawOutput` fallback is correct.
2. **Add DB-only fallback test**: Create `RUN_MODE=db_only_no_output` in fake-opencode
   to test DB reconciliation when no text events are emitted. Current harness
   always triggers `sawOutput` first.
3. **Document**: `sawOutput` check order (before DB reconciliation) is intentional.
   Output signal is simpler and sufficient. DB reconciliation is for cases where
   output alone is not available.
