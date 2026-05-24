# agentwrap-smoke

`agentwrap-smoke` is a local Go smoke harness for stress-testing `agentwrap` against a real `opencode` binary.

The point of this repo is not just to prove that a happy-path run works. Wrapping an external agent system is hard because the wrapper does not control the child process, provider behavior, local OpenCode state, stdout timing, stderr content, or DB/log writes. The useful work is in forcing bad states, checking what the wrapper reports, comparing that to OpenCode's durable state, and tightening the adapter until failures are explicit, classified correctly, handled gracefully, and successful runs are not misreported as broken.

The standard is: no important information should be silently lost. A wrapper-visible failure should say which phase failed, what category it belongs to, what retry/fallback/repair path was attempted, and where the raw evidence lives.

## What This Repo Is For

- Run real `agentwrap` + `opencode` scenarios.
- Reproduce timeout, cancellation, validation, fallback, and rate-limit behavior.
- Capture artifacts in `.agentwrap-logs/`.
- Cross-check wrapper results against OpenCode's DB and logs when stdout is incomplete.
- Keep exercising the wrapper with real process runs until ambiguous states become explicit states.
- Turn each discovered edge case into adapter code, harness evidence, and regression coverage.

This checkout currently contains the smoke harness code and prior logs. Some commands in `cmd/study` and parts of `cmd/agentwrap-run` expect a fuller study/config tree to exist beside it. The self-contained smoke commands are still useful without that larger tree.

## Important Source Paths

- Smoke harness repo:
  `/home/antonioborgerees/coding/agentwrap-smoke`
- `agentwrap` source under test:
  `/home/antonioborgerees/coding/agentwrap`
- OpenCode study source for reading only:
  `/home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode`
- OpenCode local DB:
  `~/.local/share/opencode/opencode.db`
- OpenCode local logs:
  `~/.local/share/opencode/log`

## How To Really Dig In

The reliable process is:

1. Start from a real failing or suspicious run, not an assumption.
2. Save the wrapper-visible evidence first:
   `results.json`, harness log, emitted status, emitted category, usage, warnings.
3. Read the adapter code path that produced that result in `agentwrap/opencode`.
4. Reproduce with the narrowest real command that still shows the problem.
5. If stdout is incomplete or ambiguous, inspect OpenCode's durable state:
   session row, message row, part row, and recent OpenCode logs.
6. Fix the adapter only after the wrapper-visible result and the OpenCode durable state disagree in a concrete way.
7. Add a regression test in `agentwrap/opencode/runtime_test.go` for the exact failure shape.
8. Re-run both unit tests and the real smoke command.
9. Update `AGENTWRAP_REPORTING.md` with the exact command, log directory, observed category/status, DB/log evidence, and any remaining ambiguity.

That order matters. If you skip the DB/log check, you can end up fixing the wrong layer or masking a real adapter bug with a looser heuristic.

## Robustness Goals

Every real smoke scenario should make these properties easy to verify:

- Fail fast when the wrapper can classify a terminal condition before waiting for a generic timeout.
- Preserve raw evidence: `results.json`, scenario log, `events.jsonl` when emitted, OpenCode DB snapshots when a session exists, and DB snapshot status when it does not.
- Classify failures using stable categories instead of generic process-exit strings whenever possible.
- Record the phase that failed: initial run, validation, repair attempt, repair exhaustion, fallback, timeout, cancellation, health check, or local OpenCode state.
- Make retries and fallback visible in metadata, including which target ran, why it was attempted, and what it returned.
- Treat missing evidence as evidence too. For example, no session ID, no event log, or a failed DB snapshot query should be recorded explicitly.
- Keep expected-failure smoke tests strict. A scenario should not pass merely because it failed somehow; the expected category should match when one is specified.

## Bug-Hunting Loop

Use the harness as a repeated investigation loop, not as a one-off test suite:

1. Run a narrow real scenario.
2. Read its `results.json` before reading code.
3. Check whether status, category, phase, retry metadata, repair metadata, and native metadata agree with the scenario.
4. Query OpenCode DB only when process output is incomplete or contradictory.
5. Read recent OpenCode logs for provider errors, DB errors, and timeout-adjacent failures.
6. Decide whether the bug is in `agentwrap`, the smoke harness, or OpenCode itself.
7. If the bug is in OpenCode, do not patch OpenCode. Add wrapper classification, fallback, retry, or clearer reporting around it.
8. Add the smallest unit regression in `agentwrap`, then rerun the real smoke scenario.
9. Record the result and remaining uncertainty in `AGENTWRAP_REPORTING.md`.

The aim is not to make all external failures disappear. The aim is to make each failure explicit, fast where possible, recoverable where appropriate, and diagnosable from saved evidence.

## When To Query The OpenCode DB

Query the DB when the process boundary is not a trustworthy final-state contract.

Good times to query:

- Clean process exit, but no final structured event was emitted.
- Output was produced, but the wrapper returned `runtime_exit`.
- Usage/token counts are missing even though the run appears to have completed.
- A provider error is suspected, but stdout/stderr at the wrapper boundary is incomplete.
- You need to confirm whether a session actually persisted an assistant message with `finish`.

Do not lead with the DB for every issue. First inspect the wrapper-visible result. The DB is the tie-breaker when OpenCode's stdout stream is partial, delayed, or missing terminal structure.

## What To Query

Start with the session:

```bash
opencode db --format json "select id, model, tokens_input, tokens_output, tokens_reasoning from session order by time_updated desc limit 5"
```

Then inspect messages for a session:

```bash
opencode db --format json "select role, data from message where session_id='ses_...' order by time_created desc"
```

Then inspect parts when you need to know whether OpenCode persisted a terminal step or tool activity:

```bash
opencode db --format json "select data from part where session_id='ses_...' order by time_created desc"
```

Use OpenCode logs when the provider error never crossed the process boundary cleanly:

```bash
rg -n "rate_limit|429|usage limit exceeded|too many requests" ~/.local/share/opencode/log
```

## How To Interpret DB State

- `tokens_input/output/reasoning > 0` with a clean process exit usually means the run really happened, even if stdout missed a terminal event.
- Assistant message with `finish:"stop"` is a strong completion signal.
- User-only session with zero tokens usually means the provider failed before a real completion.
- Provider 429 in OpenCode logs with a wrapper timeout means timeout was probably the symptom, not the root cause.

This is exactly the class of bug that required hardening in `agentwrap/opencode/runtime.go`.

## Main Files To Read

- Harness entrypoint:
  [cmd/agentwrap-run/main.go](/home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go)
- Higher-level study CLI:
  [cmd/study/main.go](/home/antonioborgerees/coding/agentwrap-smoke/cmd/study/main.go)
- Adapter runtime:
  [runtime.go](/home/antonioborgerees/coding/agentwrap/opencode/runtime.go)
- Native event projection:
  [projector.go](/home/antonioborgerees/coding/agentwrap/opencode/projector.go)
- Rate-limit classification:
  [rate_limit.go](/home/antonioborgerees/coding/agentwrap/opencode/rate_limit.go)
- Adapter regression tests:
  [runtime_test.go](/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go)


## Practical Commands

Build the harness:

```bash
GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
```

Run repo tests:

```bash
GOCACHE=/tmp/agentwrap-smoke-build go test ./...
```

Run agentwrap tests in the adapter repo:

```bash
cd /home/antonioborgerees/coding/agentwrap
GOCACHE=/tmp/agentwrap-go-build go test ./...
```

Run a real smoke scenario:

```bash
./agentwrap-run usage --model opencode/deepseek-v4-flash-free
```

Run a real fallback scenario:

```bash
./agentwrap-run rate-limit --primary-model minimax-coding-plan/MiniMax-M2.7 --fallback-model opencode/deepseek-v4-flash-free
```

## Compatibility Note

The project name is now `agentwrap-smoke`, but the code still accepts `ULTRAPLAN_ROOT` as an environment override in a few places for compatibility with older local scripts.
