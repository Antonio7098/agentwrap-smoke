# agentwrap Real OpenCode Robustness Test Plan

## Purpose

This plan defines the next set of real OpenCode integration tests needed to make the `agentwrap` OpenCode adapter robust.

The goal is not only to check that happy-path runs complete. The goal is to prove that the adapter handles the messy runtime boundary correctly:

- stdout may omit terminal events
- OpenCode may persist final state only in SQLite
- provider errors may arrive as human-readable text
- cancellation and timeout can race subprocess exit
- validation/repair can involve multiple runs and sessions
- OpenCode's SQLite DB can fail under concurrent access

This plan assumes the wrapper cannot change OpenCode source. The adapter must work with OpenCode as a black-box CLI plus its local persisted runtime state.

---

## Current Baseline

### Verified Passing

The following commands were run successfully with real OpenCode after the latest adapter fixes:

```bash
GOCACHE=/tmp/agentwrap-go-build go test ./...
GOCACHE=/tmp/agentwrap-smoke-build go test ./...

GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run

./agentwrap-run usage --model opencode/deepseek-v4-flash-free
./agentwrap-run artifacts --model opencode/deepseek-v4-flash-free
./agentwrap-run fixed-backoff --model opencode/deepseek-v4-flash-free
./agentwrap-run validate-json
./agentwrap-run validate-md
./agentwrap-run custom-validator
./agentwrap-run cancel
./agentwrap-run timeout
./agentwrap-run health-fail
./agentwrap-run session-fork
```

### Known Caveats

- `rate-limit` is not deterministic while OpenAI usage is available.
- Parallel real OpenCode runs can fail with SQLite checkpoint errors.
- Real OpenCode runs should be executed sequentially unless the test is explicitly about concurrency/DB contention.
- Some real scenarios use the configured default model from `config.json`; every scenario should eventually support an explicit `--model`.

---

## Test Strategy

Use three layers:

1. **Unit tests in `agentwrap`**
   - Fast, deterministic, fake subprocess/fixture-driven.
   - Prove classification logic, event projection, cancellation races, and policy decisions.

2. **Harness tests in `agentwrap-smoke/cmd/agentwrap-run`**
   - Real OpenCode subprocess.
   - Sequential, explicit scenarios.
   - Save logs, results, event streams, and DB snapshots.

3. **Manual/CI smoke script**
   - One command that runs the full real scenario matrix.
   - Produces a summary JSON and exits nonzero if any expected result fails.

---

## Harness Improvements Required First

### H1: Add a `smoke-all` command

Add:

```bash
./agentwrap-run smoke-all --model opencode/deepseek-v4-flash-free
```

It should run all stable real scenarios sequentially:

- usage
- artifacts
- fixed-backoff
- validate-json
- validate-md
- custom-validator
- cancel
- timeout
- health-fail
- session-fork

Output:

- one parent log directory
- one child directory per scenario
- `summary.json`
- final table on stdout

The command should exit nonzero if any scenario result does not match its expectation.

### H2: Add expectation flags

Each scenario should support:

```bash
--expect-status completed|failed|cancelled
--expect-category rate_limit|timeout|cancellation|runtime_exit|validation|configuration
```

Examples:

```bash
./agentwrap-run timeout --expect-status failed --expect-category timeout
./agentwrap-run cancel --expect-status cancelled --expect-category cancellation
```

This prevents "command exited 0 but scenario failed internally" from being missed.

### H3: Add `--model` to every real scenario

Currently some commands use `cfgJSON.SprintExecutionModel`. Every real scenario should accept:

```bash
--model opencode/deepseek-v4-flash-free
```

Apply this to:

- validate-json
- validate-md
- custom-validator
- cancel
- timeout
- health-fail where relevant
- session-continue

### H4: Save complete result metadata

`saveResults()` currently writes a reduced summary. Extend it to include:

- `result.RunID`
- `result.SessionID`
- `result.Status`
- `result.Err`
- `result.Warnings`
- `result.Artifacts`
- `result.Usage`
- `result.Metadata.Errors`
- `result.Metadata.NativeMetadata`
- `result.Metadata.Policy`
- `result.Metadata.Attempts`
- `result.Metadata.Validation`
- `result.Metadata.Cleanup`

This makes failures diagnosable without manually querying the DB after the fact.

### H5: Save OpenCode DB snapshot for each run

After each real run, if `result.SessionID` is set, write:

```text
opencode-session.json
opencode-messages.json
opencode-parts.json
```

Using:

```sql
select * from session where id = ?
select * from message where session_id = ? order by time_created
select * from part where session_id = ? order by time_created
```

This should run after `Wait()` returns and should not affect scenario pass/fail unless explicitly requested.

### H6: Capture event stream by default

For every scenario, drain `run.Events()` concurrently and write:

```text
events.jsonl
```

Each line should include:

- sequence
- kind
- type
- session id
- payload keys
- raw native type if present

This will make stdout-vs-DB divergence visible.

---

## Test Matrix

## A. Successful Completion

### A1: Short Text Completion

Command:

```bash
./agentwrap-run smoke-text --model opencode/deepseek-v4-flash-free
```

Prompt:

```text
Reply with exactly: OK
```

Expected:

- status: `completed`
- session id present
- usage present
- no SDK error
- DB assistant message has `finish`

Purpose:

- Prove shortest successful run works.
- Prove DB reconciliation fills gaps if stdout lacks final events.

### A2: Reasoning/Text Completion

Prompt:

```text
Think briefly, then answer with exactly: OK
```

Expected:

- status: `completed`
- usage present
- warning allowed if `step_finish` missing

Purpose:

- Prove `reasoning` and `text` event handling does not change completion semantics.

### A3: File Write Completion

Prompt:

```text
Create reports/smoke.txt containing exactly: OK
```

Expected:

- status: `completed`
- file exists
- content equals `OK` after trimming trailing whitespace
- usage present

Purpose:

- Prove tool-using runs complete and write artifacts to the requested workdir.

### A4: No Stdout Final, DB Completed

Use a real run known to omit `step_finish` on stdout.

Expected:

- status: `completed`
- warning indicates missing final structured result
- DB snapshot shows assistant `finish`
- usage projected from DB

Purpose:

- Prove the key adapter fix remains working.

---

## B. Usage Projection

### B1: Usage From Stdout

If OpenCode emits usage-like events in stdout for a model, verify:

- usage projected directly from stdout
- DB reconciliation does not overwrite with worse data

### B2: Usage From DB

Known current behavior:

```bash
./agentwrap-run usage --model opencode/deepseek-v4-flash-free
```

Expected:

- status: `completed`
- usage projected from OpenCode `session.tokens_*`
- total equals input + output + reasoning when reasoning exists in DB

Purpose:

- Prove DB usage reconciliation works.

---

## C. Validation and Repair

### C1: JSON Validation Passes First Try

Existing:

```bash
./agentwrap-run validate-json --model opencode/deepseek-v4-flash-free
```

Expected:

- status: `completed`
- validation result passed
- `reports/test.json` exists
- required fields exist

### C2: Markdown Template Validation Passes

Existing:

```bash
./agentwrap-run validate-md --model opencode/deepseek-v4-flash-free
```

Expected:

- status: `completed`
- required headings present
- no unresolved placeholders

### C3: Custom Validator Passes

Existing:

```bash
./agentwrap-run custom-validator --model opencode/deepseek-v4-flash-free
```

Expected:

- status: `completed`
- `reports/custom.txt` content passes `strings.TrimSpace(content) == "VALID"`

### C4: Required Validation Fails Deterministically

Add:

```bash
./agentwrap-run validate-fail --model opencode/deepseek-v4-flash-free
```

Prompt:

```text
Do not create any files. Reply only: done
```

Expectation:

- required file `reports/missing.txt`

Expected:

- status: `failed`
- error category: `validation`
- validation metadata explains missing file

Purpose:

- Prove validation failures are explicit and not confused with runtime failures.

### C5: Repair Succeeds

Add:

```bash
./agentwrap-run validate-repair --model opencode/deepseek-v4-flash-free
```

Initial prompt intentionally underspecifies output.

Validation requires a file.

Repair config:

- `MaxAttempts: 2`
- `SessionAction: continue`

Expected:

- initial attempt fails validation
- repair attempt succeeds
- final status: `completed`
- metadata includes repair attempt summary

Purpose:

- Prove validation repair works across real OpenCode sessions.

### C6: Repair Exhaustion

Add:

```bash
./agentwrap-run validate-repair-exhaust --model opencode/deepseek-v4-flash-free
```

Use an impossible validator.

Expected:

- status: `failed`
- error category: `validation`
- repair attempts count equals configured max

---

## D. Cancellation

### D1: Cancel Before Output

Prompt:

```text
Wait 30 seconds before answering.
```

Cancel after:

- 500ms

Expected:

- status: `cancelled`
- error category: `cancellation`
- cleanup attempted
- no `runtime_exit`

### D2: Cancel During Active Output

Prompt:

```text
Count from 1 to 100, one number per line.
```

Cancel after:

- first text event or 3 seconds

Expected:

- status: `cancelled`
- error category: `cancellation`

### D3: Cancel During Tool Use

Prompt:

```text
Run a shell command that sleeps for 30 seconds.
```

Expected:

- status: `cancelled`
- cleanup metadata indicates process cleanup
- no false `completed` from DB state

Purpose:

- Prove cancellation wins over subprocess exit and DB reconciliation.

---

## E. Timeout

### E1: Timeout Before Session Creation

Use very short timeout:

```bash
./agentwrap-run timeout --timeout-ms 1
```

Expected:

- status: `failed`
- error category: `timeout`
- session may be absent

### E2: Timeout After Session Creation

Use timeout long enough for session creation but too short for completion:

```bash
./agentwrap-run timeout --timeout-ms 500
```

Expected:

- status: `failed`
- error category: `timeout`
- no DB reconciliation to completed

### E3: Timeout During Tool Use

Prompt asks OpenCode to run a slow shell command.

Expected:

- status: `failed`
- error category: `timeout`
- cleanup attempted

Purpose:

- Prove timeout has priority over later DB state.

---

## F. Policy and Fallback

### F1: Invalid Primary Model Falls Back

Add deterministic fallback scenario:

```bash
./agentwrap-run fallback-invalid-model \
  --primary-model opencode/not-a-real-model \
  --fallback-model opencode/deepseek-v4-flash-free
```

Expected:

- first attempt fails with model/provider/runtime error
- fallback attempt succeeds
- final status: `completed`
- attempts metadata includes both attempts
- policy metadata records fallback decision

Purpose:

- Prove fallback orchestration without relying on real rate limits.

### F2: Invalid Primary Provider Falls Back

Primary:

```text
not-a-provider/model
```

Expected:

- fallback succeeds
- final status completed

### F3: Fallback Also Fails

Primary invalid.

Fallback invalid.

Expected:

- final status failed
- attempts metadata has both failures
- final error category clear

### F4: Real Rate Limit

Use only when a provider is actually rate limited or when a controllable provider can emit 429.

Expected:

- category: `rate_limit`
- retry/fallback behavior follows policy

Note:

- This should not be the primary deterministic fallback test.

---

## G. Rate Limit and Provider Error Classification

### G1: Known Max-Retry String

Already covered in unit test.

Real test requires a provider to emit:

```text
maximum retry attempts reached for rate limit
```

Expected:

- error category: `rate_limit`

### G2: OpenCode JSON Error Payload

If stderr contains JSON with:

```json
{"statusCode":429}
```

Expected:

- error category: `rate_limit`
- retry-after parsed if present

### G3: Quota Exhaustion Is Not Retryable Rate Limit

If provider emits `insufficient_quota` or equivalent:

Expected:

- not classified as retryable rate limit
- likely provider/configuration/account failure

Purpose:

- Avoid retrying or falling back incorrectly on permanent quota/account failures.

---

## H. Permission Policy

### H1: Allow Read/Edit, Ask Shell

Prompt:

```text
Read a file and write reports/permission.txt.
```

Policy:

- read: allow
- edit/write: allow
- shell: ask/deny

Expected:

- completed
- permission metadata includes translated policy

### H2: Deny Write

Prompt asks to create a file.

Policy denies edit/write.

Expected:

- run may complete as a model response
- validation fails because file not created
- permission audit records denial

### H3: Unsupported Path Rule Required

Expected:

- `StartRun` fails before process start
- error category: `configuration`

---

## I. Session Behavior

### I1: Session ID Returned

Every successful run should return:

- `RunResult.SessionID`
- session metadata

### I2: Session Continue

Command:

```bash
./agentwrap-run session-continue <session-id> "Now append one more line"
```

Expected:

- uses requested session id
- relationship marked best-effort continue
- final status completed

### I3: Session Fork Unsupported

Existing:

```bash
./agentwrap-run session-fork
```

Expected:

- start fails
- error category: `configuration`
- capability reports `session_fork` unsupported

---

## J. Observability and Store Behavior

### J1: Event Sink Captures Events

For every real run:

- `events.jsonl` should contain lifecycle events
- native event counts should match result metadata

### J2: MemoryRunStore With PolicyRunner

Reproduce previous observation:

- `ObservingRuntime -> PolicyRunner -> ValidatingRuntime -> opencode.Runtime`

Expected:

- active/completed run records should be present
- if not, identify wrapper layer where store visibility is lost

### J3: Attempt Metadata

Policy/fallback runs should record:

- attempt number
- target index
- model/provider
- status
- error category if failed

---

## K. Concurrency and DB Contention

### K1: Sequential Runs

Run five short completions sequentially.

Expected:

- all completed
- no SQLite checkpoint errors

### K2: Parallel Runs

Run two real OpenCode processes concurrently.

Expected:

- either both complete, or failures are classified as DB/runtime contention with clear diagnostics
- no silent hangs

### K3: Locked DB Recovery

Simulate or provoke DB lock/checkpoint issue.

Expected:

- adapter returns bounded error
- no indefinite wait
- logs include enough debug detail

Purpose:

- Decide whether `agentwrap` should serialize OpenCode runs by default or expose a `BoundedRunner` recommendation.

---

## L. Health Checks

### L1: Runtime Available

Expected:

- passes when `opencode` executable is available

### L2: Bad Workdir

Existing:

```bash
./agentwrap-run health-fail
```

Expected:

- `workdir` unrecoverable failure

### L3: Bad Config

Expected:

- config failure with clear diagnostic

### L4: Provider/Model Checks

Add explicit model/provider health check:

```bash
./agentwrap-run health-model --model opencode/deepseek-v4-flash-free
```

Expected:

- provider/model check passes or reports actionable unavailable status

---

## Robustness Acceptance Criteria

The adapter is robust enough when:

1. All deterministic real scenarios pass sequentially.
2. Every expected failure has the correct category.
3. Successful runs return:
   - status completed
   - session id
   - usage when OpenCode persists usage
4. Cancellation never reports `runtime_exit`.
5. Timeout never reports `completed` from delayed DB reconciliation.
6. Validation failures are distinct from runtime failures.
7. Fallback metadata identifies primary and fallback attempts.
8. OpenCode DB contention is either avoided by the harness or classified clearly.
9. The report includes the command output summary and log paths for the last full smoke run.

---

## Proposed Implementation Order

1. Add `smoke-all`.
2. Add full result metadata and DB snapshots.
3. Add `--model` to all commands.
4. Add expectation flags and scenario-level pass/fail.
5. Add deterministic fallback tests using invalid primary model/provider.
6. Add repair success/exhaustion tests.
7. Add cancellation timing variants.
8. Add timeout timing variants.
9. Add permission policy scenarios.
10. Add sequential and parallel run tests.
11. Update `AGENTWRAP_REPORTING.md` with every real smoke run result.

---

## Immediate Next Commands

After implementing `smoke-all`, the command to validate the stable matrix should be:

```bash
GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
./agentwrap-run smoke-all --model opencode/deepseek-v4-flash-free
GOCACHE=/tmp/agentwrap-go-build go test ./...
GOCACHE=/tmp/agentwrap-smoke-build go test ./...
```

