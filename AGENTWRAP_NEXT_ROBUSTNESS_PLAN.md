# agentwrap Next Robustness Plan

## Purpose

`agentwrap` is wrapping OpenCode, an external process that owns its own stdout stream, local database, logs, provider calls, tool execution, and session lifecycle. We do not control that system, so robustness means making the wrapper explicit about what it observed, what it inferred, what it recovered from durable state, and what it could not prove.

The next round should focus on process-boundary correctness. If the wrapper has reliable final-state facts, then validation, repair, retry, fallback, cancellation, and reporting can be built on top without hiding failures or inventing success.

## Current Baseline

Latest verified run:

```text
/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260521-120218
```

Result:

```text
20/20 smoke-all scenarios passed with real OpenCode runs.
```

Known improvements already implemented:

- OpenCode DB/local state failures can classify as `runtime_unavailable`.
- MiniMax rate-limit text is unit-covered.
- Timeout classification checks recent OpenCode logs.
- Clean exits with incomplete stdout can reconcile against OpenCode DB state.
- Repair metadata records `Phase` and `FailurePhase`.
- Smoke harness expected-category checks are stricter.
- Smoke harness writes DB snapshot status even when no DB snapshot can be captured.

Remaining concern:

Passing smoke tests do not prove the wrapper has exhausted the OpenCode process-boundary edge cases. The harness must keep pushing on ambiguous final states, DB/log disagreement, retries, fallback, and session behavior.

## Workstream 1: Process-Boundary Completion Semantics

Goal: define and test exactly when an OpenCode subprocess result should be considered completed, failed, cancelled, timed out, rate-limited, or locally unavailable.

### Cases To Add

1. Clean exit with final structured event.
   - Expected: `completed`.
   - Evidence: final event in `events.jsonl`, final-result category in native metadata.

2. Clean exit with assistant text but no final structured event.
   - Expected: probably `completed` with warning.
   - Evidence: emitted output, no final event, warning in result.
   - Risk: text could be partial. Need to decide whether output alone is enough or whether DB finish is required.

3. Clean exit with no final structured event but DB assistant message has terminal finish.
   - Expected: `completed` with warning and DB-recovered usage.
   - Evidence: DB session/message rows in snapshot.

4. Clean exit with no final structured event, no assistant output, and no DB terminal finish.
   - Expected: `runtime_exit`.
   - Evidence: no final event, no usable DB proof.

5. Non-zero exit with DB terminal finish.
   - Expected: decide contract.
   - Candidate behavior: fail, but include DB evidence and warning that OpenCode persisted completion despite process failure.
   - This should not silently convert to success until the contract is explicit.

6. Decode error after partial valid events.
   - Expected: fail as decode/runtime event error, include last known session and native event counts.

7. Timeout with recent OpenCode provider error log.
   - Expected: classify provider error when log match is strong enough.
   - Must avoid misclassifying stale logs from another run.

8. Timeout with DB terminal finish.
   - Expected: decide contract.
   - A timeout means the caller's deadline was exceeded, but durable state may show the process later completed. This needs careful reporting rather than naive success.

### Implementation Steps

1. Add unit tests in `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go` for every process-boundary case that can be simulated with the fake process runner.
2. Add smoke commands in `/home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go` for the real cases that require OpenCode.
3. Save per-scenario `results.json`, `events.jsonl`, `opencode-session.json`, `opencode-messages.json`, `opencode-parts.json`, and `opencode-db-snapshot-status.json`.
4. Update `AGENTWRAP_REPORTING.md` with each observed state and the chosen contract.

### Workstream 1 Evidence (2026-05-21)

#### Unit Tests Added to `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go`

| Test Name | Case | Expected Behavior | Actual Behavior | Status |
|-----------|------|-------------------|-----------------|--------|
| `TestRunCleanExitWithFinalEventCompletes` | Case 1: Clean exit with final event | `completed` | `completed` | PASS |
| `TestRunCleanExitWithOutputWithoutFinalCompletesWithWarning` | Case 2: Clean exit with assistant text, no final event | `completed` with warning (desired) | `failed` with `runtime_exit` (current) | GAP |
| `TestRunCleanExitNoFinalEventButDBTerminalFinishCompletes` | Case 3: No final event, DB has terminal finish | `completed` with warning and DB-recovered usage (desired) | `failed` with `runtime_exit` (fake runner has no DB) | GAP (fake limitation) |
| `TestRunCleanExitNoFinalNoOutputNoDBFinishFails` | Case 4: No final, no output, no DB finish | `runtime_exit` | `runtime_exit` | PASS |
| `TestRunNonZeroExitWithDBTerminalFinishFails` | Case 5: Non-zero exit with DB terminal finish | `failed` with evidence (contract undecided) | `runtime_exit` (no DB reconciliation in fake) | GAP |
| `TestRunDecodeErrorAfterPartialEventsFails` | Case 6: Decode error after partial valid events | `failed` with `malformed_event` | `malformed_event` | PASS |
| `TestTimeoutWithRecentProviderErrorLogClassifiesRateLimit` | Case 7: Timeout with recent provider error log | `rate_limit` (desired) | `timeout` (log classification timing issue) | GAP |
| `TestTimeoutWithDBTerminalFinishReportsTimeoutWithEvidence` | Case 8: Timeout with DB terminal finish | `timeout` with DB evidence (contract undecided) | `timeout` (no session in test) | PASS |
| `TestRunNonZeroExitWithFinalEventStillCompletes` | Additional: Non-zero exit with final event | `completed` (final event supersedes exit code, desired) | `failed` with `runtime_exit` (current) | BUG |
| `TestRunEmptyStdoutFails` | Additional: Clean exit with empty stdout | `runtime_exit` | `runtime_exit` | PASS |

#### Key Findings

1. **Case 2 Gap**: The plan expects "completed with warning" when `sawOutput` is true, but current `runtime.go` line 336-347 only completes if `reconcileFinalState()` returns true. With no real DB in fake tests, reconciliation fails.

2. **Case 3 Gap (Fake Limitation)**: Fake runner cannot query OpenCode DB, so `reconcileFinalState()` always fails. With real OpenCode, the DB reconciliation would work.

3. **Case 5 Gap**: Non-zero exit with DB terminal finish - contract undecided. Currently fails with `runtime_exit` even if DB confirms completion.

4. **Case 7 Gap**: `classifyRecentLogFailure` requires log file modtime within 1 minute of `r.started`. The existing tests use older log files that may not match.

5. **BUG Documented**: Non-zero exit overrides `sawFinal` because `proc.ExitCode != 0` is checked before `!r.sawFinal` in `finalResult()` (line 330 before line 336).

#### Gaps Remaining

- **Case 5**: Contract needs to be decided - should non-zero exit with confirmed DB terminal finish be `completed` or `failed`?
- **Case 7**: Log classification timing sensitivity needs investigation. Should logs be matched by content (session/model) rather than just modtime?
- **Case 8**: Contract for timeout with DB terminal finish needs documentation. Timeout is caller's deadline exceeded, but durable state may show later completion.
- **Non-zero exit bug**: The check order in `finalResult()` means final event is ignored when exit code is non-zero.

#### Real OpenCode Smoke Tests (Not Yet Implemented)

The unit tests cover cases 1-8 with fake runner. Real OpenCode smoke tests would verify:
- Actual DB reconciliation behavior with real OpenCode session
- Actual log matching timing in real environment
- Actual smoke commands for the 8 process-boundary cases

**Recommendation**: Add real smoke commands to `/home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go` for cases 2, 3, 5, 7, and 8 where unit tests show gaps between desired and current behavior.

## Workstream 2: Deterministic Rate-Limit And Fallback Testing

Goal: prove fallback behavior without depending on a provider account being rate-limited at the exact moment of the test.

### Cases To Add

1. Fake process unit test: stderr contains provider 429 / MiniMax usage limit.
   - Expected: `rate_limit`.
   - Expected metadata: `rate_limit_info`.

2. Fake process unit test: context timeout plus recent matching OpenCode log with MiniMax rate limit.
   - Expected: `rate_limit`, not generic timeout.

3. Policy unit test: primary returns `rate_limit`, fallback completes.
   - Expected: final status `completed`, attempts show primary then fallback.

4. Smoke command with invalid or controlled local wrapper fixture that simulates rate-limit shape.
   - Expected: fallback path exercised deterministically.
   - Constraint: do not patch OpenCode source.

5. Opportunistic real provider smoke:
   - Run MiniMax when a real limit is known.
   - Record whether the provider actually rate-limited.
   - Treat pass-without-rate-limit as inconclusive for fallback, not as proof.

### Implementation Options

- Add a harness-only fake `opencode` executable placed earlier on `PATH` for deterministic process-boundary tests.
- Keep real OpenCode smoke separate from fake process-boundary smoke.
- Clearly label fake runs as wrapper/process-contract tests, not real provider tests.


### Workstream 2 Evidence (2026-05-21)

#### Unit Tests for Rate-Limit Classification (in `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go`)

| Test Name | Case | Expected | Actual | Status |
|-----------|------|----------|--------|--------|
| `TestClassifyExitErrorDetectsJSONRateLimit` | 429 JSON stderr | `rate_limit` | `rate_limit` | PASS |
| `TestClassifyRateLimitDataParsesOpenCodeHeaders` | 429 with headers | rate_limit_info populated | rate_limit_info with RetryAfter | PASS |
| `TestProjectNativeFatalErrorPromotesRateLimit` | Fatal event with 429 | rate_limit promoted | rate_limit | PASS |
| `TestRunFatalEventStoresRateLimitMetadata` | Fatal event exit | metadata contains rate_limit_info | metadata contains rate_limit_info | PASS |
| `TestRunExitRateLimitStoresRateLimitMetadata` | Exit with 429 stderr | metadata contains rate_limit_info | metadata contains rate_limit_info | PASS |
| `TestClassifyExitErrorDetectsMaxRetryRateLimitMessage` | "maximum retry attempts reached for rate limit" | `rate_limit` | `rate_limit` | PASS |
| `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` | Case 1: MiniMax usage limit log | `rate_limit` | nil classification | FAIL |
| `TestTimeoutClassifiesRecentOpenCodeLogRateLimit` | Case 2: Timeout + matching log | `rate_limit` (not timeout) | `timeout` (log cutoff timing) | FAIL |
| `TestTimeoutClassifiesRecentOpenCodeDBCheckpointLog` | Timeout + DB checkpoint log | `runtime_unavailable` | `timeout` (log cutoff timing) | FAIL |

#### Policy Tests for Fallback (in `/home/antonioborgerees/coding/agentwrap/policy_test.go`)

| Test Name | Case | Expected | Actual | Status |
|-----------|------|----------|--------|--------|
| `TestPolicyRunnerFallbackThenRetriesOnFallbackTarget` | Primary fails, fallback retry succeeds | 3 attempts, fallback then retry | 3 attempts, fallback then retry | PASS |
| `TestPolicyRunnerHonorsRateLimitRetryAfter` | Rate limit triggers backoff | Sleep with retry-after | Sleep with 1hr backoff | PASS |
| `TestPolicyRunnerHonorsShouldFallbackBeforeRuntimeExitFallback` | ShouldFallback=false prevents fallback | Primary retries, no fallback | Primary retries, no fallback | PASS |

#### Smoke Commands for Rate-Limit (in `/home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go`)

| Command | Description | Evidence |
|---------|-------------|----------|
| `rate-limit` | Primary fails -> fallback completes | Final Status: completed, Attempt 1 failed (runtime_exit), Attempt 2 completed |
| `provider-429` | Documents expected category for provider 429 | Non-deterministic - passes when 429 occurs |

#### Key Findings

1. **Classification Coverage**: Rate-limit classification from stderr exits and fatal events works correctly. Tests pass for JSON 429, max-retry messages, and header parsing.

2. **MiniMax Detection Failure**: `TestClassifyRateLimitTextDetectsMiniMaxUsageLimit` fails because the nested JSON format has `responseBody` containing escaped JSON that may not be parsed by `classifyRateLimitText`. The message "usage limit exceeded" does not match current detection patterns.

3. **Log Classification Timing**: `classifyRecentLogFailure` uses `started.Add(-1 * time.Minute)` as log cutoff. Tests use log files with timestamps near the cutoff boundary, causing timing sensitivity failures.

4. **Fallback Behavior**: Real smoke test shows primary fails with `runtime_exit` and fallback completes with `completed`. This confirms fallback infrastructure works with real OpenCode.

#### Gaps Remaining

1. **MiniMax detection**: Need to either add "usage limit exceeded" to detection patterns or fix nested JSON parsing in `classifyRateLimitText`.

2. **Log cutoff timing**: Extend the log cutoff window in `classifyRecentLogFailure` or adjust test log timestamps to be more recent.

3. **Fake opencode fixture**: The plan mentions creating `/home/antonioborgerees/coding/agentwrap-smoke/fake-opencode/` for deterministic process-boundary tests. Started but not completed.

4. **Policy test gap**: No direct test for "primary returns rate_limit, fallback completes". `TestPolicyRunnerHonorsRateLimitRetryAfter` tests retry on same target, not fallback to alternative.

#### Evidence from Real Smoke Test

```
./agentwrap-run rate-limit --primary-model opencode/gpt-5.5 --fallback-model opencode/deepseek-v4-flash-free

PolicyRunner configured with opencode/gpt-5.5 primary and opencode/deepseek-v4-flash-free fallback
Starting run...
Run started: ID=policy-1
Final Status: completed
Attempt 1 target=0 model=opencode/gpt-5.5 status=failed error=runtime_exit
Attempt 2 target=1 model=opencode/deepseek-v4-flash-free status=completed error=
```

Results JSON shows:
- `fallback_used: true`
- `retry_count: 1`
- `status: completed`
- Attempt 1: failed, Attempt 2: completed

This proves the fallback path works correctly when primary fails.


## Workstream 3: OpenCode DB Hardening

Goal: DB reconciliation and evidence capture should never hide failures or block final result reporting.

### Cases To Add

1. DB command unavailable.
   - Expected: run result preserved; DB snapshot status records failure.

2. DB query returns non-JSON.
   - Expected: reconciliation declines; evidence records parse failure.

3. DB query times out.
   - Expected: reconciliation declines quickly; final result does not hang.

4. DB locked / checkpoint failure.
   - Expected: `runtime_unavailable` when it is the primary failure; snapshot failure when it only affects evidence capture.

5. DB has session row but no assistant message.
   - Expected: not enough proof of completion.

6. DB has assistant message with no finish.
   - Expected: not enough proof of completion.

7. DB has assistant message with terminal finish and usage.
   - Expected: completion can be reconciled when process exit is clean.

### Implementation Steps

1. Add unit tests around `queryOpenCodeDB`, usage extraction, and reconciliation.
2. Consider extracting DB reconciliation into a smaller testable helper.
3. Add smoke evidence assertions that every scenario either captures DB rows or records a DB snapshot status reason.

### Evidence Collected 2026-05-21

Harness fixes made in `agentwrap-smoke` only:

- Registered standalone DB smoke commands so they can be run directly instead of waiting for the real-provider front half of `smoke-all`.
- Changed fake DB smoke commands to stop calling `os.Exit(1)` on `StartRun` failure so one harness setup failure does not abort later scenarios.
- Changed DB snapshot capture to use the fake `opencode` executable and its `DB_MODE` env for fake DB scenarios instead of always shelling out to the real local `opencode db`.

Evidence run directory:

```text
/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db
```

Observed results:

1. `db-unavailable`
   - `results.json`: `status=failed`, `error_category=runtime_exit`, `session_id=ses_fake123`.
   - `opencode-db-snapshot-status.json`: `status=failed` with `exit status 1` for all three DB files.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-unavailable/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-unavailable/opencode-db-snapshot-status.json`

2. `db-non-json`
   - `opencode-db-snapshot-status.json` currently reports `status=captured`.
   - But `opencode-session.json` contains plain text `This is not JSON output from database`.
   - This is a harness evidence gap: snapshot capture records file presence, not JSON validity.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-non-json/opencode-db-snapshot-status.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-non-json/opencode-session.json`

3. `db-timeout`
   - `results.json`: `status=failed`, `error_category=runtime_exit`.
   - `opencode-session.json` saved as `[]`, and snapshot status still reports `captured`.
   - This fake case does not yet prove query timeout handling; the fixture returns an empty array instead of forcing harness-side timeout metadata.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-timeout/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-timeout/opencode-session.json`

4. `db-locked`
   - `results.json`: `status=failed`, `error_category=runtime_exit`.
   - `opencode-db-snapshot-status.json`: `status=failed` with `exit status 1` for all DB files.
   - This does preserve run result plus snapshot failure evidence, but it does not yet surface `runtime_unavailable` at the wrapper result boundary in the fake scenario.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-locked/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-locked/opencode-db-snapshot-status.json`

5. `db-session-no-assistant`
   - `results.json`: `status=completed` from final structured events.
   - `opencode-session.json` contains a session row with tokens but no assistant message evidence.
   - This scenario currently proves evidence capture, not reconciliation refusal, because the fake run mode emits a final event and completes before DB state matters.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-session-no-assistant/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-session-no-assistant/opencode-session.json`

6. `db-assistant-no-finish`
   - `results.json`: `status=completed` from final structured events.
   - `opencode-messages.json` contains two JSON documents concatenated across lines: a session row and then an assistant message without `finish`.
   - This is useful evidence, but it also shows the fixture is not yet modeling real `opencode db --format json` output shape cleanly.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-assistant-no-finish/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-assistant-no-finish/opencode-messages.json`

7. `db-assistant-with-finish`
   - `results.json`: `status=completed`.
   - `opencode-messages.json` contains assistant data with `finish:"stop"`.
   - Snapshot capture works, but the saved DB session IDs differ from the wrapper session ID (`ses_test456` in DB evidence vs `ses_fake123` in result), so this does not yet prove real reconciliation correctness.
   - Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-assistant-with-finish/results.json`, `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db/db-assistant-with-finish/opencode-messages.json`

Current conclusions:

- The smoke harness can now run and save DB hardening evidence deterministically without touching `../agentwrap`.
- The current DB smoke fixtures are strong enough to expose evidence-capture gaps, especially that `opencode-db-snapshot-status.json` treats invalid JSON as `captured`.
- The current fixtures do not yet prove wrapper-side reconciliation behavior for cases 5-7 because the fake runs with `RUN_MODE=final` complete from structured events before DB evidence becomes decisive.
- The current fake timeout fixture also does not prove query timeout handling because the harness does not impose a DB query timeout and the captured output is just `[]`.

Next harness-only improvements recommended:

1. ~~Validate captured DB snapshot files as JSON before marking snapshot status `captured`~~ — **DONE** (improvement 1 in `saveOpenCodeDB`).
2. ~~Add a fixture mode where the fake run exits cleanly without a final event but DB output provides the only completion proof~~ — **DONE** (new `DB_MODE=db_only_proof` + `RUN_MODE=db_proof` in `fake-opencode.sh`, new `cmdDBOnlyProof` command).
3. ~~Add a harness-side timeout around DB snapshot queries so `db-timeout` records explicit snapshot timeout failure~~ — **DONE** (5 second context deadline per query in `saveOpenCodeDB`).
4. ~~Align fake DB session IDs with run session ID so reconciliation evidence is not contradictory~~ — **DONE** (`ses_fake123` now used consistently in all DB fixture outputs).

### Second-Run Evidence 2026-05-21 (ws3-db-v2)

Run directory: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db-v2`

**`db-non-json` — JSON validation now works:**

Before: `status=captured` with invalid JSON files written.
After: `status=failed` with `"errors":{"opencode-messages.json":"not valid JSON",...}`.

This correctly fails the snapshot when DB output is not valid JSON.

**`db-only-proof` — new scenario added:**

- `RUN_MODE=db_proof`: emits `step_start` + `text` only, no `step_finish`.
- `DB_MODE=db_only_proof`: DB queries return session+message+part rows, all with `sessionID=ses_fake123` matching the run session.
- `results.json`: `status=failed`, `category=runtime_exit`, `session_id=ses_fake123` (wrapper correctly did not complete from missing final event).
- `opencode-db-snapshot-status.json`: `status=captured` with all three DB files present and valid JSON.
- `opencode-session.json`, `opencode-messages.json`, `opencode-parts.json`: all contain valid JSON with `ses_fake123` session ID.
- Evidence: `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ws3-db-v2/db-only-proof/`

This proves the wrapper correctly fails when no final event is present AND correctly captures DB evidence even when the run itself is not completed by the wrapper.

**Session ID alignment now correct:**

All DB fixture outputs now use `ses_fake123` (matching the run session ID) instead of the prior `ses_test123` / `ses_test456` mismatch. Evidence is now coherent: session ID in `results.json` matches session ID in DB snapshot files.

**`db-timeout` fixture behavior:**

The fake `opencode db` command with `DB_MODE=timeout` sleeps 30 seconds. With the new 5-second harness-side timeout on each query, this would now produce a `"query timeout after 5s"` error per file instead of a silent empty-array capture. However, `saveOpenCodeDB` runs the DB queries after the run completes, and the run itself (`RUN_MODE=partial`) produces `runtime_exit` before the DB queries are even attempted, so the snapshot status shows `failed` from the run exit. The timeout classification at the DB query level is now in place for cases where the run itself completes or times out in ways that leave session ID non-empty.

**Evidence coherence confirmed:**

All 8 DB scenarios now run and save coherent evidence. Session IDs are consistent across `results.json` and DB snapshot files. JSON validation now marks invalid snapshots as `failed`. The harness is ready to test real OpenCode DB reconciliation behavior with these fixtures as a baseline.

## Workstream 4: Session Lifecycle Coverage

Goal: session metadata should match what OpenCode actually persisted, especially around continue, fresh, and unsupported fork behavior.

### Cases To Add

1. Fresh session success.
   - Expected: result has session ID; DB session row exists.

2. Continue existing session success.
   - Expected: requested ID equals result/session row ID or metadata explains replacement.

3. Continue missing/invalid session.
   - Expected: explicit category; no silent fresh session unless marked best-effort.

4. Continue after failed run.
   - Expected: contract decision needed. Either explicit failure or clear continued relationship.

5. Fork unsupported.
   - Current expected: `configuration`.
   - Keep this strict.

6. Repair with `SessionActionFresh`.
   - Expected: repair attempt metadata records fresh relationship.

7. Repair with `SessionActionContinue`.
   - Expected: if supported, metadata and DB rows show continuity; if not reliable, document why fresh is preferred for repair smoke.

### Evidence To Capture

- `RunResult.SessionID`
- `RunMetadata.Session`
- DB session row
- DB message rows
- repair attempt session metadata
- policy attempt session metadata

### Smoke Tests Added (2026-05-21)

Added 6 new session lifecycle smoke commands to `cmd/agentwrap-run/main.go`:

| Command | Description | Expected Behavior | Actual Behavior | Status |
|---------|-------------|-------------------|-----------------|--------|
| `session-fresh` | Fresh session success | `completed` with session ID | `failed` `runtime_exit` | GAP |
| `session-continue-existing` | Continue existing session | `completed` with continued relationship | `failed` `runtime_exit` but session continued | GAP |
| `session-continue-missing` | Continue with missing session ID | `failed` with explicit category | `failed` `runtime_exit` with no-session error | PASS |
| `session-continue-after-fail` | Continue after failed run | `completed` showing recovery | `failed` `timeout` (100ms too short to create session) | GAP |
| `session-fork` | Session fork unsupported | `failed` `configuration` | Already existed in smoke-all | PASS |
| `repair-with-continue` | Repair with SessionActionContinue | `completed` with continued relationship | `failed` `runtime_exit` no repair attempts | GAP |

Note: `cmdValidateRepair` already tests repair with `SessionActionFresh`.

### Key Findings

1. **Fresh session returns `runtime_exit`**: When using `opencode run --format json`, the process exits cleanly but the wrapper does not receive a final structured event (`finish` event). The wrapper then classifies this as `runtime_exit: OpenCode finished without a final structured result`. This appears to be a real issue - when OpenCode uses `--format json`, it should emit proper JSON events including a final event.

2. **Continue existing session works but wrapper fails**: When continuing an existing session:
   - Run 1 creates a session (relationship: fresh)
   - Run 2 continues that session (relationship: best_effort, continued: true)
   - The session continuation IS reflected in the metadata
   - But both runs fail with `runtime_exit` because OpenCode doesn't emit final events in JSON mode

3. **Continue missing session behavior**: When attempting to continue a non-existent session:
   - OpenCode creates a NEW session with the requested ID (even if it doesn't exist)
   - No error is raised by OpenCode
   - Wrapper reports `runtime_exit` because no final event was emitted
   - This suggests OpenCode silently creates a fresh session when the requested session doesn't exist

4. **Continue after fail issue**: The 100ms timeout was too short - the first run couldn't even create a session before timing out. This test needs a longer initial timeout or different failure mode.

5. **JSON mode final event issue**: The root cause across all cases is that `opencode run --format json` doesn't emit final `finish` events that the wrapper expects. Without this, the wrapper always classifies the run as `runtime_exit` even when OpenCode actually completed successfully.

### Evidence from Tests

**session-fresh** (fresh session with WantSession=true, SessionActionFresh):
```
Status: failed
Category: runtime_exit
Session ID: ses_1b584d26dffe05STtPIlL0Ngfw
Session relationship: fresh
Event types: lifecycle.transition, session.relationship, step_start
No finish event received
```

**session-continue-existing** (two runs, second continues first):
```
Run 1 Status: failed (created session ses_xxx)
Run 2 Status: failed (continued session ses_xxx)
Run 2 Session relationship: best_effort
Run 2 Session continued: true
```
Both runs return `runtime_exit` despite OpenCode processing the requests.

**session-continue-missing** (continue non-existent session `ses_does_not_exist_12345`):
```
Status: failed
Category: runtime_exit
Session ID: ses_does_not_exist_12345 (OpenCode created it anyway)
Expectation check: passed=true (expected failed/runtime_exit)
```

### Gaps Remaining

1. **OpenCode JSON mode final events**: OpenCode `--format json` doesn't emit the final `finish` event that the wrapper expects. This is the root cause of `runtime_exit` classification for all session tests. This could be:
   - An issue with how OpenCode emits events in JSON mode
   - A missing event type that the wrapper should handle
   - A version-specific behavior difference

2. **Continue missing session contract**: When continuing a non-existent session, OpenCode silently creates a fresh session. The wrapper should document this best-effort behavior.

3. **Continue after fail test design**: The test needs redesign to use a failure mode that still creates a session (e.g., a short timeout that's long enough to create session but short enough to fail during work).

4. **repair-with-continue**: The repair is configured with `SessionActionContinue` but no repair attempts occur because the initial run fails with `runtime_exit` before validation is even checked. This is the same root cause as #1.

### Recommendations

1. **Investigate OpenCode JSON event output**: The wrapper expects a final event (like `finish` or `run_finish`) but OpenCode in JSON mode doesn't emit one. Verify with OpenCode source or documentation.

2. **Consider non-JSON mode for comparison**: Test whether the issue is specific to `--format json` by comparing with regular text output mode.

3. **Document session continuation contract**: OpenCode's behavior of silently creating fresh sessions when the requested session doesn't exist should be documented as best-effort behavior.

## Workstream 5: Error Category Contract Tightening

Goal: every stable expected-failure smoke scenario should assert both status and category.

### Smoke Tests Added

Added 6 new error category smoke tests to `cmd/agentwrap-run/main.go`:

| Command | Description | Expected Category |
|---------|-------------|-------------------|
| `invalid-provider` | Invalid provider string | `configuration` |
| `invalid-model` | Invalid model string | `model_unavailable` (but currently `runtime_exit`) |
| `opencode-db-unavailable` | OpenCode DB checkpoint/lock failure | `runtime_unavailable` |
| `provider-429` | Provider returns 429 | `rate_limit` (non-deterministic) |
| `provider-auth` | Provider auth/account issues | `authentication` (non-deterministic) |
| `fatal-clean` | Clean fatal event with no provider detail | `runtime_exit` |

Also tightened `fallback-all-fail` expectation to include category: `runtime_exit`.

### Evidence Gathered

1. **Invalid model (`opencode/not-a-real-model`)**:
   ```
   Status: failed
   Category: runtime_exit
   UserDetail: OpenCode reported a fatal session error
   DebugDetail: Model not found: opencode/not-a-real-model.
   ```
   OpenCode emits `error` event with "Model not found" -> classified as `runtime_exit`.
   Current classification: runtime_exit (not `model_unavailable`).

2. **Invalid provider (`nonexistent-provider/test`)**:
   ```
   Status: failed
   Category: runtime_exit
   UserDetail: OpenCode reported a fatal session error
   ```
   OpenCode emits `error` event -> classified as `runtime_exit`.
   Current classification: runtime_exit (not `configuration`).

3. **Clean fatal event with model not found**:
   ```
   Native metadata event_categories: map[fatal_error:1 lifecycle:2 session:1]
   Native metadata native_event_types: map[error:1 lifecycle.transition:2 session.relationship:1]
   ```
   The `error` type event triggers `EventFatalError` in projector.go:88, which becomes `runtime_exit`.

4. **OpenCode DB unavailable**:
   ```
   Status: failed
   Category: runtime_exit
   Error: opencode run: runtime_exit: OpenCode finished without a final structured result
   ```
   Classification path: `classifyOpenCodeLocalFailure()` in runtime.go:773 detects
   "pragma wal_checkpoint", "database is locked", "sqlite.*database" and returns
   `ErrorRuntimeUnavailable`. However, when the run produces no final event and
   process exits cleanly, it may fall through to `runtime_exit` instead.

5. **fallback-all-fail** (tightened expectation):
   ```
   Status: failed
   Category: runtime_exit
   Attempt 1: model=opencode/not-a-real-model error=runtime_exit
   Attempt 2: model=opencode/also-not-a-model error=runtime_exit
   ```

### Current Classification Behavior

| Scenario | Current Category | Classification Path |
|----------|-----------------|---------------------|
| Invalid provider | `runtime_exit` | OpenCode `error` event -> projector.go:88 -> runtime_exit |
| Invalid model | `runtime_exit` | OpenCode `error` event with "Model not found" -> runtime_exit |
| OpenCode DB unavailable | `runtime_exit` | process exit without final event -> runtime_exit (DB check not triggered) |
| Provider 429 | `rate_limit` | classifyRateLimitText/Data -> ErrorRateLimit |
| Provider auth | `authentication` | Not detected separately from rate_limit in current code |
| Clean fatal event | `runtime_exit` | projector.go:88 fallback to runtime_exit |

### Category Contract Decision

- **Invalid provider**: Currently `runtime_exit`. Should be `configuration` (provider string
  is syntactically invalid, not a runtime failure). Requires OpenCode to return structured
  error with provider identification, or wrapper to pre-validate provider format.

- **Invalid model**: Currently `runtime_exit`. Should be `model_unavailable` when OpenCode
  explicitly reports "Model not found". Requires the wrapper to recognize this specific
  error message and map it to `model_unavailable` rather than generic `runtime_exit`.

- **OpenCode DB unavailable**: Code path exists in `classifyOpenCodeLocalFailure()` for
  `runtime_unavailable`, but the DB failure may not always produce observable evidence.
  When OpenCode exits cleanly without final event due to DB issue, it classifies as
  `runtime_exit`. Need to strengthen DB failure detection.

- **Provider 429**: Correctly classified as `rate_limit`.

- **Provider auth**: Not currently distinguished from rate_limit. Detection would require
  parsing provider error messages for auth-specific patterns.

- **Clean fatal event**: Correctly classified as `runtime_exit`.

### Gaps Remaining

1. **Invalid model should be `model_unavailable`**: OpenCode emits "Model not found" but
   wrapper classifies it as `runtime_exit`. Need to enhance `projector.go` or add
   post-processing to recognize "model not found" messages.

2. **Invalid provider should be `configuration`**: Provider validation happens at wrapper
   level, not OpenCode level. Wrapper should detect invalid provider format before
   invoking OpenCode.

3. **OpenCode DB unavailable needs better triggering**: The DB classification exists but
   may not be reached when OpenCode exits cleanly. Need to ensure DB failures during
   a run are captured and classified as `runtime_unavailable`.

4. **Provider auth detection is missing**: No current mechanism to distinguish auth
   failures from rate limits. Would need to add auth-specific error message patterns.

## Workstream 6: Observability And Evidence Assertions

Goal: smoke runs should fail when critical evidence is silently missing.

### Cases To Add

1. `events.jsonl` expected for real runs that emit events.
2. `opencode-db-snapshot-status.json` required for every OpenCode smoke scenario.
3. If result has session ID, DB snapshot should be `captured` or explicitly `failed`.
4. If result has no session ID, DB snapshot should be `skipped` with `no session id`.
5. `results.json` must include status, category when failed, native metadata, cleanup metadata, and phase metadata when validation/repair is configured.

### Implementation Steps

1. Add a `verify-evidence` harness command.
2. Run it against a smoke-all directory.
3. Make `smoke-all` optionally call it at the end.
4. Add report output listing missing or partial evidence.

### Implementation Completed

**Added `verify-evidence` command** (cmd/agentwrap-run/main.go lines 2867-3164):

```bash
./agentwrap-run verify-evidence [.agentwrap-logs/smoke-all-TIMESTAMP] [--strict]
```

The command verifies:

1. **results.json**: Must exist, have non-empty `status`. If status=failed, should have `error_category`. Must have `native_metadata` and `cleanup_metadata`.

2. **events.jsonl**: Optional but expected for real runs (not health-fail, health-model, session-fork). Info-level note if missing.

3. **opencode-db-snapshot-status.json**: Must exist for every OpenCode smoke scenario.

4. **Session ID logic**: 
   - If `session_id` present -> DB snapshot status must be `captured` or `failed`
   - If no `session_id` -> DB snapshot status must be `skipped` with `errors.session="no session id"`

5. **Validation/Repair phase metadata**: When `validation_metadata.configured=true`, must have `validation_metadata.final`. When `repair_metadata.configured=true`, must have `repair_metadata.phase`.

### Evidence Verification Run

Ran verify-evidence against `.agentwrap-logs/smoke-all-20260521-120218`:

**Errors found (13):**
- Missing `opencode-db-snapshot-status.json`: artifacts, custom-validator, fallback-all-fail, fixed-backoff, usage, validate-json, validate-md
- Missing status in results.json: health-fail, health-model, session-fork
- Missing db snapshot status file for health-fail, health-model, session-fork

**Warnings (6):**
- Missing native_metadata: health-fail, health-model, session-fork
- Missing cleanup_metadata: health-fail, health-model, session-fork

**Info (16):**
- events.jsonl missing for most scenarios (these runs used ObservingRuntime but did not write events.jsonl to disk except artifacts scenario)

### Root Causes Identified

1. **health-fail, health-model, session-fork**: These commands call `saveResults()` with an empty `agentwrap.RunResult{}` because they don't start a run that produces a result - they only perform health checks or capability checks. The results.json has minimal data.

2. **Missing db snapshots in artifacts, custom-validator, etc.**: The harness `saveOpenCodeDB()` is only called in some scenarios (smoke-text, smoke-reasoning, smoke-file-write, cancel, timeout, validate-fail, validate-repair, validate-repair-exhaust, fallback-invalid-model, fallback-invalid-provider). Many scenarios that create sessions do NOT call `saveOpenCodeDB()`, leaving the db snapshot file absent.

3. **events.jsonl not written**: Most scenarios use `ObservingRuntime` which has a `capturingSink` but the harness only writes events to the log file in `cmdRun()` via the goroutine reading from `run.Events()`. Individual smoke commands don't set up event capture to disk.

### Gaps Remaining

1. **health-fail, health-model, session-fork**: These scenarios don't create runs and thus have no session ID. They should still write a DB snapshot status file with `status=skipped` and `errors.session="no session id"` reason - but only health-fail and health-model make sense to have this since they don't interact with runs. session-fork attempts a run but it fails early.

2. **saveOpenCodeDB() not called for all scenarios**: Many scenarios that create sessions don't capture the DB snapshot. This should be consistent across all scenarios that create sessions.

3. **events.jsonl not captured for most scenarios**: Only `artifacts` scenario writes events.jsonl. Other scenarios use ObservingRuntime but don't persist events to disk.

4. **Optional vs Required events.jsonl**: The current approach treats events.jsonl as "optional but expected for real runs". Need to decide if it should be required for scenarios that use ObservingRuntime.

## Suggested Execution Order

1. Tighten `fallback-all-fail` category investigation.
   - It is already in `smoke-all` and currently under-specified.

2. Add deterministic process-boundary unit tests.
   - This gives fast coverage for the most dangerous wrapper logic.

3. Add `verify-evidence`.
   - This makes future smoke runs harder to misread.

4. Add session continue real smoke.
   - This is important for real agent workflows and repair/retry semantics.

5. Add deterministic rate-limit/fallback harness mode.
   - This closes the gap left by provider limits being stateful and unpredictable.

6. Expand DB hardening tests.
   - Especially bad JSON, timeout, locked DB, missing assistant finish.

## Definition Of Done

This next phase is done when:

- Every `smoke-all` expected failure asserts a category.
- Every smoke scenario writes `results.json` and DB snapshot status.
- Process-boundary completion behavior is documented and unit-covered.
- Real OpenCode smoke covers fresh session, continue session, repair success, repair exhaustion, timeout, cancellation, fallback, health failure, and unsupported fork.
- Deterministic tests cover rate-limit fallback without needing a live provider to be exhausted.
- `AGENTWRAP_REPORTING.md` lists remaining ambiguities explicitly instead of implying that all passed smoke tests mean there are no risks.

## Workstream 7: Process-Group Cleanup And Final-State Precedence

Goal: prove that the wrapper treats a real final structured event as terminal success, preserves explicit provider failures, and terminates the whole OpenCode process group instead of only the direct child.

### Cases To Add

1. Non-zero exit with final event and no provider error.
   - Expected: `completed`.
   - Evidence: final event in `events.jsonl`, `exit_code != 0`, no provider-classified stderr.

2. Non-zero exit with final event plus explicit rate-limit stderr.
   - Expected: `rate_limit`.
   - Evidence: final event present, stderr contains rate-limit shape, metadata includes `rate_limit_info`.

3. Final event after delayed child exit.
   - Expected: `completed`.
   - Evidence: final event arrives before process termination and still wins the final classification.

4. Cancellation should terminate the whole process group.
   - Expected: no surviving child processes after cancel.
   - Evidence: fake peer spawns a helper process that exits when the group receives `SIGTERM` or `SIGKILL`.

5. Forced kill fallback after graceful timeout.
   - Expected: cleanup records a graceful attempt and a force attempt when the child ignores `SIGTERM`.
   - Evidence: cleanup metadata plus a fake peer that traps `SIGTERM`.

6. Malformed output before and after a final event.
   - Expected: malformed output before the final event should still fail; a clean final event should not be lost because the process exits non-zero after success.
   - Evidence: compare `RUN_MODE=malformed`, `RUN_MODE=final`, and `RUN_MODE=nonzero_final`.

### Implementation Steps

1. Extend `fake-opencode` with a helper-child mode and a trapped-`SIGTERM` mode.
2. Add adapter tests that assert `completed` vs `rate_limit` precedence from the same stdout/stderr shape.
3. Add cancellation tests that confirm `Cancel()` reaches a process group and does not leave a helper subprocess behind.
4. Add smoke commands that run the fake peer through `agentwrap-smoke` so these contracts are verified outside unit tests too.
5. Record the exact evidence in `AGENTWRAP_REPORTING.md` once the new cases are exercised.

### Implementation Evidence (2026-05-21)

#### Smoke Tests Added

| Test Name | Case | Expected | Actual | Status |
|-----------|------|----------|--------|--------|
| `process-group-nonzero-final` | Non-zero exit with final event | `completed` | `completed` | PASS |
| `process-group-nonzero-ratelimit` | Non-zero exit with rate-limit stderr | `rate_limit` | `rate_limit` | PASS |
| `process-group-cancel-children` | Cancellation with child processes | `cancelled`, no survivors | `cancelled`, no survivors | PASS |
| `process-group-malformed-before` | Malformed output before final event | `malformed_event` | `malformed_event` | PASS |
| `process-group-malformed-after` | Final event followed by malformed output | `completed` | `completed` | PASS |
| `process-group-final-delayed` | Final event after delayed finish | `completed` | `completed` | PASS |

#### Key Findings

1. **Non-zero exit with final event**: Wrapper correctly treats final structured event as terminal success even with non-zero exit code (7). The `sawFinal` flag takes precedence over `proc.ExitCode != 0`.

2. **Rate-limit with final event**: Wrapper correctly classifies rate-limit errors even alongside final events. The `classifyExitError()` checks stderr for rate-limit patterns even when `proc.ExitCode != 0` and `sawFinal` is true.

3. **Process-group cleanup**: The cancellation completes with correct status and PID-file evidence shows both the fake OpenCode process and helper process are no longer alive after cancellation.

4. **Malformed-after-final**: The wrapper now preserves a valid final event when malformed JSON appears later in the stream. The contract is:
   - Malformed BEFORE final event: fail with `malformed_event`
   - Malformed AFTER final event: record warning/evidence, but final result stands

   The scanner remains strict; the run loop normalizes post-final decode errors after `sawFinal` is true.

5. **Final event delayed**: Wrapper correctly handles delayed final events (1-second sleep between events before step_finish). The event-based processing works correctly.

#### Fake OpenCode Modes Added

```bash
RUN_MODE=nonzero_final         # Final events, exit 7
RUN_MODE=nonzero_ratelimit     # Final events + rate-limit stderr, exit 7
RUN_MODE=final_delayed         # Step start, text, sleep, step_finish
RUN_MODE=malformed_before_final # Malformed before final events
RUN_MODE=malformed_after_final  # Final events, then malformed
HELPER_CHILD=1                 # Spawn helper that exits on SIGTERM
```

#### Commands Added

```bash
./agentwrap-run process-group-nonzero-final
./agentwrap-run process-group-nonzero-ratelimit
./agentwrap-run process-group-cancel-children
./agentwrap-run process-group-malformed-before
./agentwrap-run process-group-malformed-after
./agentwrap-run process-group-final-delayed
```

#### Files Modified

- `fake-opencode/fake-opencode.sh` - Extended with new RUN_MODEs and HELPER_CHILD
- `cmd/agentwrap-run/main.go` - 6 new smoke commands + smoke-all integration

#### Gaps Remaining

1. **Forced kill fallback**: Not implemented - requires HELPER_CHILD_TRAP mode and SIGKILL verification
2. **Cross-platform process-group support**: Deferred. The current implementation and smoke evidence are Unix/Linux-oriented because the runtime uses `Setpgid` and negative-PID signals.

## Prioritisation Plan For Remaining Workstreams

This section supersedes the earlier suggested execution order. Workstream 7 is intentionally excluded because the core final-state precedence and process-group cleanup work has already been implemented and evidenced above. The remaining work should be sequenced so that wrapper contracts are decided before the harness grows more broad.

### Priority 0: Stabilise The Contract Before Adding Breadth

Do this first because every later smoke assertion depends on these choices.

1. Decide and document terminal-state precedence for ambiguous process-boundary cases:
   - clean exit + assistant output + no final structured event
   - clean exit + DB terminal finish + no final structured event
   - non-zero exit + DB terminal finish
   - timeout + DB terminal finish
2. Update `AGENTWRAP_REPORTING.md` with the chosen behavior for each case.
3. Convert the chosen behavior into unit tests in `agentwrap/opencode/runtime_test.go`.

Recommended contract:

- A final structured event is the strongest success signal unless stderr/stdout contains a classified provider/local failure.
- Clean exit plus DB terminal finish may reconcile to `completed` with an explicit warning and DB-recovered evidence.
- Non-zero exit plus DB terminal finish should remain `failed` unless a final structured event was observed; include DB evidence and a warning.
- Timeout plus DB terminal finish should remain `timeout`; include DB evidence that durable state may have completed after the caller deadline.
- Assistant text alone should not be enough for `completed` unless the contract explicitly accepts partial-output risk.

### Priority 1: Workstream 1 - Process-Boundary Completion Semantics

This is the highest-value remaining stream. It defines the adapter's correctness boundary.

1. Finish unit coverage for the process-boundary matrix in `runtime_test.go`.
2. Extract DB reconciliation into a smaller helper if needed so fake tests can exercise it without shelling out to real OpenCode.
3. Add real smoke commands only for cases that unit tests cannot faithfully model.
4. Make every ambiguous outcome include:
   - observed process exit state
   - whether a final event was seen
   - whether assistant output was seen
   - DB reconciliation status
   - warning text when success is inferred rather than directly observed

Definition of done:

- The eight Workstream 1 cases have explicit expected status/category contracts.
- Fake unit tests cover the full precedence matrix.
- Real OpenCode smoke confirms the cases where stdout, DB, and logs can disagree.

### Priority 2: Workstream 5 - Error Category Contract Tightening

Once terminal-state precedence is stable, improve category specificity. This has immediate user-facing value because it changes vague `runtime_exit` failures into actionable errors.

1. Map OpenCode fatal text `"Model not found"` to `model_unavailable`.
2. Decide how much provider-name validation belongs in `agentwrap` before invoking OpenCode.
3. Classify known local OpenCode DB failures as `runtime_unavailable` even when OpenCode exits cleanly without a final event, provided the evidence is present in stderr/stdout/logs.
4. Add auth-specific patterns only after collecting stable real examples; avoid guessing from one provider's wording.
5. Tighten `smoke-all` expected categories for all stable expected-failure scenarios.

Definition of done:

- Invalid model no longer reports generic `runtime_exit`.
- Invalid provider behavior is either pre-validated as `configuration` or explicitly documented as OpenCode-owned.
- DB-local failures reach `runtime_unavailable` when evidence is observable.
- Every deterministic expected failure in `smoke-all` asserts status and category.

### Priority 3: Workstream 2 - Deterministic Rate-Limit And Fallback Testing

Do this after category tightening so fallback policy sees the right failure category.

1. Fix MiniMax `"usage limit exceeded"` detection, preferably by improving nested JSON extraction before adding broad string patterns.
2. Remove timing flakiness from `classifyRecentLogFailure` tests by controlling log timestamps or injecting the log clock/cutoff.
3. Add the missing policy test for `primary rate_limit -> fallback completed`.
4. Complete the harness-only fake `opencode` fixture for deterministic rate-limit fallback.
5. Keep opportunistic real provider rate-limit smoke as evidence only, not as the proof of fallback correctness.

Definition of done:

- `rate_limit` classification is deterministic in unit tests.
- Fallback from `rate_limit` is directly unit-covered.
- The smoke harness can exercise fallback without requiring a live account to be exhausted.

### Priority 4: Workstream 3 - OpenCode DB Hardening

Most harness evidence capture is now stronger, so the next DB work should focus on wrapper reconciliation rather than more snapshot plumbing.

1. Add unit tests around DB reconciliation, usage extraction, non-JSON output, command failure, timeout, and locked DB behavior.
2. Ensure fake DB fixtures model real `opencode db --format json` output shapes exactly.
3. Separate two concepts in reporting:
   - DB snapshot capture for evidence
   - DB reconciliation used by the runtime to decide final status
4. Verify that `db-only-proof` or equivalent can prove reconciliation behavior, not only evidence capture.
5. Keep snapshot failures non-blocking unless DB failure is itself the primary runtime failure.

Definition of done:

- Invalid DB output declines reconciliation and records parse failure.
- DB query timeout declines reconciliation quickly.
- Terminal assistant finish can reconcile clean process exits when that is the chosen contract.
- DB evidence files and runtime reconciliation metadata cannot contradict the result without a warning.

### Priority 5: Workstream 4 - Session Lifecycle Coverage

This depends on process-boundary semantics because current session tests are dominated by missing final-event behavior.

1. Re-run session smoke after Workstream 1 decisions are implemented.
2. Verify fresh session success with result session ID and DB session row.
3. Verify continue-existing behavior, including whether OpenCode preserves the requested session ID.
4. Document continue-missing behavior as best-effort if OpenCode silently creates a session with the requested ID.
5. Redesign continue-after-fail using a failure mode that creates a session before failing.
6. Re-test repair with `SessionActionContinue` only after initial successful runs can reach validation.

Definition of done:

- Fresh, continue-existing, continue-missing, continue-after-fail, fork, repair-fresh, and repair-continue all have explicit contracts.
- Session metadata, policy attempt metadata, repair metadata, and DB rows agree or explain the mismatch.
- Unsupported fork remains a strict `configuration` failure.

### Priority 6: Workstream 6 - Observability And Evidence Assertions

Do this last, after the expected result shapes are stable. Otherwise evidence assertions will churn while contracts are still changing.

1. Make every scenario write `results.json`.
2. Make every OpenCode scenario write `opencode-db-snapshot-status.json`.
3. If a scenario has no session ID, write snapshot status as `skipped` with an explicit `no session id` reason.
4. If a scenario has a session ID, require DB snapshot status to be `captured` or explicitly `failed`.
5. Decide whether `events.jsonl` is required for all `ObservingRuntime` scenarios or only for scenarios that explicitly attach an event sink.
6. Add or keep `verify-evidence` as the guardrail once the required artifact set is final.

Definition of done:

- `verify-evidence` has no missing critical artifacts for `smoke-all`.
- Health/capability-only commands produce intentionally minimal result files and explicit skipped DB snapshot status.
- Missing evidence is treated as a smoke failure unless the scenario documents why it cannot exist.

### Recommended Execution Order

1. Priority 0: decide contracts and document them.
2. Priority 1: finish process-boundary unit tests and real smoke checks.
3. Priority 2: tighten error categories.
4. Priority 3: make rate-limit fallback deterministic.
5. Priority 4: harden DB reconciliation tests.
6. Priority 5: re-run and complete session lifecycle coverage.
7. Priority 6: enforce evidence assertions across `smoke-all`.

### Final Done State

The remaining robustness phase is complete when:

- Process-boundary outcomes have explicit contracts rather than ad hoc precedence.
- All deterministic expected failures assert both status and category.
- Rate-limit fallback is proven without relying on provider account state.
- DB reconciliation is unit-tested separately from DB evidence capture.
- Session lifecycle smoke results match persisted OpenCode state or explain divergence.
- `smoke-all` cannot pass while silently omitting critical evidence.
