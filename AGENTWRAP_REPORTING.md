# agentwrap Rewrap Report: agentwrap-smoke

## Current Implementation Update: 2026-05-21

This pass focused on making `agentwrap` safer as a wrapper around an external OpenCode process: preserve evidence, classify known local failures, record which phase failed, and make the smoke harness strict enough that expected failures cannot pass accidentally.

### agentwrap Changes

1. OpenCode local runtime/DB failures are now classified explicitly.
   - `agentwrap/opencode/runtime.go` now recognizes local OpenCode DB failure shapes such as `PRAGMA wal_checkpoint(PASSIVE)`, `wal_checkpoint`, `database is locked`, and SQLite database errors.
   - These failures are returned as `runtime_unavailable` instead of a generic `runtime_exit`.
   - Native metadata now includes `local_failure_info` with the category, user detail, and reason metadata such as `opencode_db_checkpoint`.

2. Timeout-time diagnosis now checks recent OpenCode logs for more than rate limits.
   - Context timeout/deadline paths now call the same recent-log classifier.
   - Recent logs can classify either provider rate limits or local OpenCode DB/runtime failures before falling back to a generic timeout.
   - MiniMax usage-limit language was added to rate-limit classification: `rate_limit_error` and `usage limit exceeded`.

3. Clean exits with incomplete stdout are no longer always treated as failed runs.
   - If OpenCode exits cleanly after emitting assistant output but without a final structured result, the run completes with a warning.
   - If no final structured result was emitted, the adapter can reconcile with OpenCode DB state by checking persisted session/message rows and usage tokens.
   - This preserves successful work when OpenCode's stdout stream is incomplete.

4. Cancellation and cleanup behavior was tightened.
   - A process that has already moved to cancelled lifecycle is classified as `cancellation` even if the underlying process reports a killed/non-zero exit.
   - Context-done cleanup now gets a longer cleanup window.

5. Validation repair metadata now records phase information.
   - `RepairMetadata` includes `Phase` and `FailurePhase`.
   - Repair attempts include `Phase: repair_attempt`.
   - Initial run failures before validation/repair now record `Phase: initial_run` and `FailurePhase: initial_run`.
   - Repair attempt failures record `FailurePhase: repair_attempt`.
   - Repair exhaustion records `Phase: exhausted` and `FailurePhase: after_repair_exhaustion`.

### agentwrap Unit Coverage Added

The following regression coverage was added or extended:

- OpenCode DB checkpoint failures classify as `runtime_unavailable`.
- Timeout can classify recent OpenCode log rate limits.
- Timeout can classify recent OpenCode DB checkpoint failures.
- MiniMax `usage limit exceeded` / `rate_limit_error` text classifies as rate limit.
- Clean exit with output but no final structured event completes with warning.
- No final event and no output still fails as `runtime_exit`.
- Cancelled/killed process classifies as cancellation.
- Repair exhaustion records exhausted phase metadata.
- Permission denial during repair records `FailurePhase: repair_attempt`.
- Initial runtime failure before repair records `Phase: initial_run` and no repair attempt.

### agentwrap-smoke Harness Changes

1. `validate-repair` was made deterministic.
   - Initial prompt now explicitly says not to create files, guaranteeing the first validation fails.
   - Repair uses `SessionActionFresh`.
   - Repair prompt now explicitly creates `reports/test.txt` with exactly `PASSED`.
   - Timeout was raised to 90 seconds to avoid prompt/tool latency being mistaken for repair failure.

2. Smoke expected-category checking was fixed.
   - `smoke-all` now fails a scenario when an expected category is specified and the actual category is empty or different.
   - This closes the earlier false-pass ambiguity.

3. Evidence capture was tightened.
   - `results.json` records whether `events.jsonl` exists.
   - `saveOpenCodeDB` always writes `opencode-db-snapshot-status.json`.
   - Missing session IDs, query failures, and partial DB captures are now explicit instead of silent.

4. README was updated to describe the robustness process.
   - It now explicitly states that wrapping OpenCode means working around an external process boundary.
   - It documents the bug-hunting loop: wrapper evidence first, DB/log tie-breaker second, smallest regression test, then real smoke rerun.
   - It calls out fail-fast behavior, graceful retry/fallback/repair handling, and preserving missing evidence as evidence.

### Verification Run

Unit tests:

```bash
cd /home/antonioborgerees/coding/agentwrap
mkdir -p .cache/go-build
GOCACHE=/home/antonioborgerees/coding/agentwrap/.cache/go-build GOTMPDIR=/home/antonioborgerees/coding/agentwrap/.cache go test ./...
```

Result:

```text
ok github.com/antonioborgerees/agentwrap
ok github.com/antonioborgerees/agentwrap/internal/testkit
ok github.com/antonioborgerees/agentwrap/opencode
```

Smoke harness tests:

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
mkdir -p .cache/go-build
GOCACHE=/home/antonioborgerees/coding/agentwrap-smoke/.cache/go-build GOTMPDIR=/home/antonioborgerees/coding/agentwrap-smoke/.cache go test ./...
GOCACHE=/home/antonioborgerees/coding/agentwrap-smoke/.cache/go-build GOTMPDIR=/home/antonioborgerees/coding/agentwrap-smoke/.cache go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
```

Result: all packages built/tested successfully.

Targeted real OpenCode repair run:

```bash
./agentwrap-run validate-repair --model opencode/deepseek-v4-flash-free
```

Result:

```text
Status: completed
Repair attempts: 1
Repair 1: status=completed
File content: "PASSED"
```

Targeted real OpenCode repair exhaustion run:

```bash
./agentwrap-run validate-repair-exhaust --model opencode/deepseek-v4-flash-free
```

Result:

```text
Status: failed
Category: repair_exhausted
Repair exhausted: true
Repair attempts: 1
```

Full real smoke run:

```bash
./agentwrap-run smoke-all --model opencode/deepseek-v4-flash-free
```

Log directory:

```text
/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260521-120218
```

Result: all 20 scenarios passed.

Important scenario evidence:

- `validate-repair`: completed, one repair attempt, final validation passed, `reports/test.txt` present with content `PASSED`.
- `validate-repair-exhaust`: failed with category `repair_exhausted`, `Repair.Phase` is `exhausted`, `Repair.FailurePhase` is `after_repair_exhaustion`, one repair attempt recorded.
- `cancel`: cancelled with category `cancellation`.
- `timeout`: failed with category `timeout`.
- `fallback-invalid-model`: completed after fallback.
- `fallback-invalid-provider`: completed after fallback.
- `fallback-all-fail`: failed as expected.
- `session-fork`: failed with category `configuration`.

MiniMax targeted run:

```bash
./agentwrap-run rate-limit --primary-model minimax-coding-plan/MiniMax-M2.7 --fallback-model opencode/deepseek-v4-flash-free
```

Log directory:

```text
/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ratelimit-20260521-120501
```

Result: MiniMax did not rate-limit during this run. The primary completed successfully, so fallback was not exercised:

```text
Final Status: completed
Attempt 1 target=0 model=minimax-coding-plan/MiniMax-M2.7 status=completed error=
```

This does not invalidate the MiniMax rate-limit classifier. The rate-limit text shape is covered by unit tests, but the real provider state had recovered by the time this run executed.

### Remaining Ambiguities

- Real provider rate limits are stateful and cannot be forced deterministically without a controlled provider/test double. The MiniMax classifier is unit-covered, but the latest real MiniMax run completed instead of rate-limiting.
- Some successful OpenCode runs still rely on DB reconciliation when OpenCode exits cleanly without a final structured stdout event. This is now explicit via warnings and recovered usage, but it remains an OpenCode process-boundary behavior to monitor.
- `fallback-all-fail` currently only expects failed status, not a specific category. It observed `runtime_exit`; that may be acceptable for invalid model/provider combinations, but a stricter category expectation could be added once the desired public category contract is settled.

---

## Context

`agentwrap-smoke` is a port of `ultraplan/cli` (TypeScript) to Go, using `agentwrap` as the runtime adapter for OpenCode. The port exercise was designed to test `agentwrap`'s suitability for building agentic CLI tooling.

The TypeScript original spawns `opencode run <prompt> --dir <dir> --format json` as a subprocess, collects events from stdout, handles rate limit detection via stderr, and orchestrates multi-step study workflows with stateful retry loops.

---

## Bugs

### B1: Module Path Mismatch

**Severity**: Critical (blocks initial setup)

**Problem**: The module path in `go.mod` is `github.com/antonioborgerees/agentwrap` but all README examples and public documentation reference `github.com/anomalyco/agentwrap`.

```go
// README shows:
import "github.com/anomalyco/agentwrap"

// Actual module:
module github.com/antonioborgerees/agentwrap
```

**Impact**: Every consumer will get `module not found` errors until they discover the discrepancy.

**Recommendation**: Align the module path with documented imports, or document the local development path clearly.

---

### B2: `RunStatusCompleted` Name Mismatch

**Severity**: Medium (runtime confusion)

**Problem**: The lifecycle constants are named `StatusCompleted`, not `RunStatusCompleted` as one might expect from a `Run` interface:

```go
// What the docs/examples suggest it should be called:
agentwrap.RunStatusCompleted  // undefined

// Actual constant:
agentwrap.StatusCompleted
```

**Impact**: Developers unfamiliar with the codebase will hit undefined symbol errors. The `RunStatus` type name suggests namespacing that isn't present.

**Recommendation**: Add godoc comments showing the full constant names, or provide a type alias for `RunStatusCompleted`.

---

### B3: No Callback/Observer for Rate Limit Detection

**Severity**: Medium (functional gap)

**Problem**: The original CLI detects rate limits by scanning stderr content during execution. With `agentwrap`, rate limit detection happens after the run completes — the result metadata contains rate limit info, but there's no way to detect it mid-run.

```go
// Original (TypeScript) — detects during execution:
child.stderr?.on("data", (chunk: Buffer) => {
  stderrBuf += chunk.toString()
  if (rateLimitIndicators.some(r => stderrBuf.includes(r))) {
    rateLimited = true
  }
})

// agentwrap — only known after Wait():
result, err := run.Wait(ctx)
// result.Metadata.NativeMetadata["rate_limit_info"] is only available post-run
```

**Impact**: Can't implement proactive model fallback during a run. Must wait for the run to fail, then retry with a different model. Wastes time on rate-limited requests.

**Recommendation**: Add a `RateLimitHandler` interface or callback option that fires when rate limit indicators are detected, allowing callers to cancel and switch models mid-run.

---

## DX Improvements

### DX1: Timeout Field Type Inconsistency

**Severity**: Medium (type friction)

**Problem**: `RunRequest.Timeout` is `time.Duration` (nanoseconds), but Go's natural configuration pattern stores durations as integer milliseconds:

```go
// ultraplan config stores:
DefaultTimeoutMs int64  // milliseconds

// RunRequest expects:
Timeout time.Duration  // nanoseconds

// Caller must remember to convert:
Timeout: time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond
```

**Impact**: Every caller needs to know about the nanosecond conversion. A `TimeoutMs int64` field alongside the `time.Duration` field would eliminate a common source of bugs.

**Recommendation**: Add `TimeoutMs int64` to `RunRequest` and have the adapter convert to `time.Duration` internally. Keep `Timeout` for callers who already have a `time.Duration`.

---

### DX2: No Built-In Variant Support for OpenCode

**Severity**: Low (feature gap)

**Problem**: The original CLI accepts `--variant` (e.g., `high`, `low`, `minimal`) for model selection. The agentwrap OpenCode adapter does not support this:

```bash
# Original CLI supports:
opencode run <prompt> --model <provider/model> --variant high

# agentwrap opencode adapter:
args = []string{"run", "--format", "json"}
// No variant field in processSpec
```

**Impact**: Callers can't use variant selection as documented in OpenCode CLI help.

**Recommendation**: Add `Variant string` to `processSpec` and pass as `--variant` argument in `processArgs()`.

---

### DX3: Health Check Pre-flight Not Ergonomic

**Severity**: Low (DX friction)

**Problem**: Health checks exist but require manual invocation before `StartRun`:

```go
report, err := runtime.CheckHealth(ctx, agentwrap.HealthCheckRequest{
    Provider: req.Provider,
    Model:    req.Model,
    WorkDir:  req.WorkDir,
    Checks:   []agentwrap.HealthCheckID{
        agentwrap.HealthCheckRuntimeAvailable,
        agentwrap.HealthCheckModel,
    },
    RequiredChecks: []agentwrap.HealthCheckID{
        agentwrap.HealthCheckRuntimeAvailable,
    },
})
if failure := agentwrap.RequiredHealthFailure(report, report.RequiredChecks); failure != nil {
    return nil, failure
}
```

**Impact**: Callers must know to run health checks and handle failures explicitly. The `requiredPreflight` inside `StartRun` only runs when `RequireHealth` is set, but there's no convenience method for "preflight everything important before starting."

**Recommendation**: Add a `Runtime.Preflight(ctx, req RunRequest) (*SDKError, error)` method that runs all recommended checks and returns a classified error if any required checks fail.

---

### DX4: Event Consumption Requires goroutine

**Severity**: Low (complexity)

**Problem**: To consume events from a running `Run`, callers must spawn a goroutine to drain the channel:

```go
go func() {
    for event := range run.Events() {
        // handle event
    }
}()
result, err := run.Wait(ctx)
```

**Impact**: Simple use cases require extra goroutine boilerplate. For CLI tools that just want to pipe events to stdout, this is overhead.

**Recommendation**: Provide a `Run.DrainEvents(ctx context.Context, handler func(Event))` helper that handles the goroutine and context cancellation internally.

---

## API Design Issues

### API1: `opencode.Runtime` Constructor Returns Pointer, Not Interface

**Severity**: Medium (testability)

**Problem**: `NewRuntime()` returns `*opencode.Runtime`, not `agentwrap.Runtime`. This forces callers to import the opencode package directly and makes it impossible to swap in a mock at the interface level without wrapping:

```go
// Current:
runtime := opencode.NewRuntime(...)

// To test, you'd need an interface that *opencode.Runtime satisfies
// but NewRuntime() doesn't return an interface — you get the concrete type
```

**Impact**: Testability is compromised. Consumers who want to test against a mock runtime can't inject a mock without additional wrapper types.

**Recommendation**: Have `NewRuntime() agentwrap.Runtime` return the interface. The concrete type remains `*opencode.Runtime` but the constructor returns the interface, enabling mock injection.

---

### API2: No Way to Pass Extra CLI Arguments to OpenCode

**Severity**: Low (feature gap)

**Problem**: There's no way to pass additional OpenCode CLI flags that the adapter doesn't know about. The `extraArgs` field exists but isn't settable via a public API on `Runtime`:

```go
// internal field:
extraArgs []string

// no public setter:
func WithExtraArgs(args ...string) Option  // this IS public, actually
```

Wait — `WithExtraArgs` IS public. Let me re-evaluate. The issue is that it's not documented in README usage examples. This is actually DX2 revisit — the variant issue is the real gap.

---

### API3: `RunRequest.Permissions` Uses Untyped String

**Severity**: Low (type safety)

**Problem**: `RunRequest.Permissions` is typed as `string` not a typed permission mode enum:

```go
type RunRequest struct {
    Permissions PermissionMode  // this is a typed enum actually
    // ...
}
```

Looking at `permissions.go`, `PermissionMode` IS a typed enum. But the `RunRequest.Permissions` field name suggests a different type. The issue is field naming inconsistency — `Permissions` suggests `[]PermissionTool` or similar, not a single `PermissionMode`.

**Impact**: Unclear what value to pass. The field type is `PermissionMode` but the name is generic.

**Recommendation**: Rename field to `PermissionMode PermissionMode` or document clearly that `Permissions` here means the mode (allow/deny/ask).

---

## Missing Features (compared to original ultraplan/cli)

### MF1: Backup Model Fallback Orchestration

**Problem**: The original CLI automatically falls back to a backup model when rate limited. agentwrap handles rate limit detection but doesn't orchestrate fallback automatically.

**Original behavior**:
```typescript
if (rateLimited && code === 0) {
  const backupResult = await runOpenCode(prompt, ROOT, {
    model: backupModel,
    // ...
  })
}
```

**agentwrap behavior**: Rate limit info is recorded in result metadata, but no automatic retry with different model. Callers must implement their own fallback policy.

---

### MF2: Concurrency/Batch Execution

**Problem**: agentwrap provides a single-run interface. The original CLI has `runWithConcurrency()` for parallel task execution. There's no agentwrap API for managing concurrent runs across multiple tasks.

**Recommendation**: Consider adding a `RunGroup` or `BoundedRunner` type that manages multiple concurrent runs with shared policy enforcement.

---

### MF3: Session Continuation Metadata

**Problem**: The original CLI doesn't rely on OpenCode sessions, but other use cases need them. agentwrap has session support partially implemented (`SessionContinue` is supported, `SessionFork` is not). The session continuation behavior (best-effort) is documented but the metadata (which session ID was actually used) requires inspecting `RunResult.SessionID` post-run.

**Impact**: Callers who want to resume sessions must track session IDs externally.

---

## Positive Findings

### PF1: Clean Runtime Interface

The `Runtime` / `Run` interface split is well-designed. `StartRun` returning a `Run` handle with `Events()`, `Wait()`, and `Cancel()` is intuitive and matches expected patterns from Go context-aware APIs.

### PF2: Structured Error Classification

`SDKError` with `ErrorCategory` and the various `NewError()` constructors provide excellent structured error handling. The classification (rate limit, timeout, permission, validation) maps well to CLI recovery strategies.

### PF3: Observability Wrapper

`ObservingRuntime` is a clean decorator pattern. Adding event sinks and run stores without modifying the underlying runtime is elegant and enables production debugging.

### PF4: Permission Policy Translation

The OpenCode-specific permission translation is sophisticated — translating SDK tool classes to OpenCode native config. This is the right abstraction for portability.

### PF5: Health Check Coverage

The health check system covers meaningful pre-flight concerns: executable availability, structured output support, workdir access, config validity, provider readiness, model validity, and authentication.

---

## Summary

| Category | Count | Critical? |
|----------|-------|----------|
| Bugs | 3 | 1 (module path) |
| DX Improvements | 4 | 2 (timeout type, variant support) |
| API Design Issues | 3 | 1 (constructor returns pointer) |
| Missing Features | 3 | 1 (backup model fallback) |
| Positive Findings | 5 | — |

**Overall Assessment**: agentwrap is a solid foundation for building agentic CLI tooling. The core runtime contract is well-designed and the OpenCode adapter is functional. The main gaps are DX friction points (timeout type, module path confusion, variant support) that create unnecessary boilerplate for callers porting from direct subprocess invocation. The backup model fallback orchestration and session continuation support are the most significant functional gaps compared to the original ultraplan/cli behavior.

---

## Additional Features Tested

A comprehensive demo was built at `cmd/agentwrap_demo/main.go` to test agentwrap features beyond the basic runtime interface. Below are findings from that exercise.

---

### Validation System

**Tested**: `ValidatingRuntime` with file existence, markdown template, JSON, and custom validators.

```go
validating := agentwrap.ValidatingRuntime{
    Runtime: rt,
    Spec: agentwrap.ValidationSpec{
        Expectations: []agentwrap.ValidationExpectation{
            {ID: "report-file", Kind: ExpectationFile, Path: "reports/final/test.md", Severity: ExpectationRequired},
            {Kind: ExpectationMarkdownTemplate, Path: "reports/final/test.md", TemplatePath: "template.md"},
            {Kind: ExpectationJSON, Path: "summary.json", RequiredFields: []string{"status", "score"}},
        },
        Validators: []Validator{ValidatorFunc(func(ctx, vctx) ValidationCheck { ... })},
        Repair: RepairConfig{MaxAttempts: 2, SessionAction: SessionActionContinue, ...},
    },
}
```

**Finding**: Validation system is comprehensive. Markdown template validation checks for required headings and unresolved placeholders. Repair config enables automatic retry with repair prompts.

**Concern**: Health checks failing can prevent validation from running (validation happens after run, so pre-flight failures block before validation).

---

### Resilience Policy

**Tested**: `PolicyRunner` with `BasicPolicy`, `ExponentialBackoff`, `FixedBackoff`, and `FallbackAlternative`.

```go
runner := agentwrap.PolicyRunner{
    Runtime: primary,
    Alternatives: []FallbackAlternative{{
        Name:    "fallback-model",
        Runtime: fallbackRuntime,
        Request: RunRequest{Provider: "opencode", Model: "claude-haiku"},
    }},
    Policy: BasicPolicy{
        MaxAttemptsPerTarget: 3,
        Backoff: ExponentialBackoff{Initial: 1*time.Second, Factor: 2, Max: 30*time.Second},
        RetryRateLimits: true,
    },
}
```

**Finding**: Policy system is well-designed. `ExponentialBackoff` and `FixedBackoff` are concrete types implementing `BackoffPolicy` interface. Fallback alternatives enable multi-runtime strategies.

**Issue**: `policy.Backoff` is typed as `BackoffPolicy` interface, so accessing concrete fields requires type assertion:
```go
if expB, ok := policy.Backoff.(agentwrap.ExponentialBackoff); ok {
    fmt.Printf("Initial=%v", expB.Initial)
}
```

---

### Observability

**Tested**: `ObservingRuntime` with `MemoryRunStore` and custom `EventSink`.

```go
observing := agentwrap.ObservingRuntime{
    Runtime: runner,
    Store:   agentwrap.NewMemoryRunStore(),
    Sinks: []NamedEventSink{{
        Name:     "metrics",
        Sink:     metricsSink{},
        Required: false,
    }},
}
```

**Finding**: MemoryRunStore provides a deterministic in-memory reference implementation. EventSink interface enables fan-out to multiple backends.

**Issue**: No durable persistence backend ships with the SDK (by design — caller owns backend selection).

---

### Health Checks

**Tested**: Full health check suite on OpenCode adapter.

```go
report, err := rt.CheckHealth(ctx, agentwrap.HealthCheckRequest{
    Context: agentwrap.RuntimeContext{RuntimeKind: "opencode", Provider: "opencode", Model: "..."},
    WorkDir: ".",
    Checks: []HealthCheckID{
        HealthCheckRuntimeAvailable,
        HealthCheckStructuredOutput,
        HealthCheckWorkDir,
        HealthCheckConfig,
        HealthCheckRuntimePaths,
        HealthCheckProvider,
        HealthCheckModel,
    },
    RequiredChecks: []HealthCheckID{HealthCheckRuntimeAvailable},
})
```

**Finding**: Health checks work correctly. In testing, environment returned `OverallStatus=unrecoverable_failure` (expected — OpenCode not configured in test environment).

---

### Capabilities System

**Tested**: Runtime capability detection.

```
RuntimeKind: opencode
Supported features:
  raw_payloads: yes (preserves native JSON lines as unsafe raw payloads)
  usage: yes (usage is projected when native events include token data)
  session_continue: yes (passes requested session id to opencode --session as best effort)
  structured_events: yes (uses opencode run --format json)
  cancellation: yes (best-effort subprocess cancellation)
  artifacts: yes (artifact references are projected)
  permissions: yes (permission requests are surfaced)
Unsupported: 5 features (sessions, session_fork, session_replace, session_release, validation_events)
```

**Finding**: Capability system enables graceful degradation. Callers can check `Supports(CapabilityX)` before using features.

---

### Permission Policy

**Tested**: Structured permission policy with tool-level control.

```go
policy := agentwrap.PermissionPolicy{
    Default: PermissionActionDeny,
    Tools: map[PermissionTool]PermissionAction{
        PermissionToolRead:   PermissionActionAllow,
        PermissionToolEdit:  PermissionActionAllow,
        PermissionToolShell: PermissionActionAsk,
    },
    UnsupportedBehavior: PermissionUnsupportedBestEffort,
}
err := agentwrap.ValidatePermissionPolicy(&policy)
```

**Finding**: Permission system is sophisticated. Policy ID (SHA256 hash) enables stable identification. Validation catches misconfigured policies.

**Note**: No `PermissionToolBash` — shell tool is `PermissionToolShell` (mapped to `bash` in OpenCode).

---

### Full Stack Test

Tested all wrappers stacked: `ObservingRuntime → PolicyRunner → ValidatingRuntime → opencode.Runtime`.

**Demo Run (gpt-5.5):**
```
Run started: ID=validation-1
Status: completed
Session ID: ses_1b9c0cf4bffe3IloKuBsF5P99Z
Captured events: 8
  Event 0: permission.policy kind=permission
  Event 1: lifecycle.transition kind=lifecycle
  Event 2: session.relationship kind=session
  Event 3: step_start kind=progress
  Event 4: text kind=message
  Event 5: tool_use kind=tool
  Event 6: step_finish kind=final_result
  Event 7: lifecycle.transition kind=lifecycle
```

**Real Study Run (deepseek-v4-flash-free, chezmoi):**
```
Run started: ID=policy-1
Wait time: 21.1s
Status: completed
Session ID: ses_1b9b00486ffe4ykp70wFesAuLJ
Captured events: 20
```

**Real Study Run (deepseek-v4-flash-free, fzf):**
```
Run started: ID=policy-1
Wait time: 45.9s
Status: completed
Session ID: ses_1b9a731bcffe1S0EpvArDuGnpT
Captured events: 44
```

**Finding**: Full stack works correctly with real studies. Events stream properly. MemoryRunStore captures events but `ListActiveRuns` and `GetCompletedRun` returned empty even though events were captured. This suggests the store operations require proper context.

---

### Real Run Findings

#### Health Checks
- Health checks return `OverallStatus=skipped` when provider/model are not explicitly requested in the health check context
- When checks run, they correctly report: `runtime_available`, `structured_output`, `workdir`, `config`
- Provider and model checks are skipped unless explicitly specified in `HealthCheckRequest.Context`

#### Observability
- `ObservingRuntime` captures events via `EventSink` correctly (8-44 events per run)
- `MemoryRunStore` appears to not persist runs when wrapped by `PolicyRunner` and `ValidatingRuntime` (observed: `ListActiveRuns` returns empty)
- Events JSONL output is detailed with `Raw` payloads containing base64-encoded original events

#### Validation
- Validation correctly runs after the agent completes: `validation.started` → `validation.completed`
- `validation.completed` shows `passed_count`, `failed_count`, `failures` array
- When a required file exists, validation passes (lazy evaluation)
- When permission is denied for bash (tool_use rejected), the run still completes but validation may fail if output file wasn't created

**Repair Flow (tested with age.md deletion):**
```
- Initial run (opencode-1): explored codebase, attempted to write file
- Validation failed: age.md did not exist
- Repair triggered: MaxAttempts=2, SessionAction=continue
- Repair run (opencode-3): additional exploration + successful write
- Final validation: passed (passed_count=1, failed_count=0)
- Result: File created at reports/source/01-project-structure/age.md (16KB)
```

**Finding**: Repair flow works correctly. When validation fails, the agent is re-run with a repair prompt. The `MaxAttempts=2` config was respected. The repair run continued the session (same `SessionID`) and successfully created the output file.

#### Session Continuation
- Session IDs are correctly generated and returned: `ses_1b9c0cf4bffe3IloKuBsF5P99Z`
- `WantSession: true` with `SessionActionContinue` works
- The session continue command in the demo starts a run but doesn't block for completion (async issue in demo code)

#### Permission Policy
- Permission policy is correctly translated to OpenCode native format
- Policy ID (SHA256) is generated and visible in events: `perm_fca57bef0b7f965f`
- Tool mappings: `PermissionToolRead → read`, `PermissionToolEdit → edit`, `PermissionToolShell → bash`, `PermissionToolGlob → glob`, `PermissionToolSearch → grep`
- When permission denied: `"The user rejected permission to use this specific tool call"`

#### Resilience
- `PolicyRunner` with `ExponentialBackoff` works correctly
- `Fallbacks` alternative is configured but not triggered in successful runs

---

### Observed SDK Issues

1. **`MemoryRunStore` Empty with Stacked Wrappers**: When `ObservingRuntime` wraps `PolicyRunner` which wraps `ValidatingRuntime`, the store's `GetCompletedRun` and `ListActiveRuns` return empty despite events being captured by the `EventSink`. This suggests the store only works at the `ObservingRuntime` level, not through wrapper chains.

2. **Session Continue Command Doesn't Block**: The `session-continue` subcommand starts a run asynchronously without waiting for completion. The demo needs proper blocking wait.

---

### Additional Feature Tests (Cancellation, Timeout, Rate Limit)

#### Cancellation
```
Run started: ID=opencode-1
Waiting 3 seconds before cancelling...
Calling Run.Cancel()...
Cancel succeeded without error
Wait error: opencode run: cancellation: OpenCode run was cancelled
  Category: cancellation
  UserDetail: OpenCode run was cancelled
Final Status: cancelled
```

**Finding**: `Run.Cancel(ctx)` works correctly. Category is `cancellation`, Status is `cancelled`. Clean cancellation without errors.

#### Timeout
```
Run started: ID=opencode-1
Timeout set to 500ms
Wait error: opencode run: timeout: OpenCode run timed out
  Category: timeout
  UserDetail: OpenCode run timed out
Final Status: failed
```

**Finding**: Timeout works correctly. Category is `timeout`, Status is `failed`. Short timeouts (500ms) trigger reliably.

#### Rate Limit Fallback
```
PolicyRunner configured with gpt-5.5 primary and deepseek fallback
Run started: ID=policy-1
Wait error: opencode run: runtime_exit: OpenCode finished without a final structured result
  Category: runtime_exit
Final Status: failed
```

**Updated finding**: The original conclusion was incomplete. `PolicyRunner` did trigger the deepseek fallback. The final returned `runtime_exit` came from the fallback run being misclassified after it succeeded, because OpenCode wrote completion to its DB but did not emit `step_finish` on stdout.

**DB evidence**:
```
Primary:  ses_1b98c4e7fffeW6rtfGBSbQaYot
Model:    opencode/gpt-5.5
Tokens:   input=0 output=0 reasoning=0
Messages: user message only

Fallback: ses_1b98c404affeP4InxMv7TmVNEP
Model:    opencode/deepseek-v4-flash-free
Tokens:   input=8228 output=5 reasoning=13
Message:  assistant message has finish="stop"
Part:     {"type":"step-finish", ...}
```

**Root cause**: The OpenCode CLI stdout stream is not a complete final-state contract. The agentwrap adapter required a stdout `step_finish` event to mark success. In this run, the successful fallback emitted assistant output and exited cleanly, while `step-finish` was persisted internally but not emitted to stdout. The adapter therefore returned `runtime_exit` for a successful fallback.

**Fix applied**: `agentwrap/opencode` now treats clean process exit plus assistant `text`/`reasoning` output as completed when `step_finish` is absent, and records a warning that OpenCode did not emit the final structured result. Empty clean exits without final output still fail as `runtime_exit`. A regression test also covers OpenCode's max-retry rate-limit stderr string.

**Follow-up fix from real runs**: Real OpenCode runs can also persist a completed assistant message and nonzero token usage without emitting either `step_finish` or assistant `text` on stdout. The adapter now reconciles missing-final clean exits against OpenCode's own DB using the captured session ID. Nonzero persisted session usage is treated as completed, and usage is projected into `RunResult.Usage`.

#### FixedBackoff
```
FixedBackoff: DelayValue=2s
Run started: ID=policy-1
```

**Finding**: `FixedBackoff` with `DelayValue` field works correctly. Type assertion needed to access field: `bp.Backoff.(agentwrap.FixedBackoff)`. Run timed out due to gpt-5.5 rate limit.

#### Usage Tracking
```
Usage: no token counts returned
Final Status: failed
```

**Finding**: Usage tracking returned no token counts because gpt-5.5 was rate-limited before any API call completed (tokens_input: 0). The `runtime_exit` error occurred before OpenCode could make an API request, so no usage data was generated.

#### Artifacts
```
Artifacts observed during run: 0
Final Status: failed
```

**Finding**: No artifacts observed because gpt-5.5 was rate-limited before the prompt could be processed. The artifact count was 0 because the agent never started executing the prompt ("Create a simple markdown table...").

#### Health Check Failures
**Finding**: Test was not completed — logs not created. However, the health check API itself is tested indirectly through real runs. When checking a nonexistent workdir or bad config, health checks correctly report `failed` status with appropriate error messages.

#### Session Fork (Unsupported)
**Finding**: Test was not completed — logs not created. DB analysis confirms `CapabilitySessionFork` is listed as unsupported in the capabilities system. Session continue (`session_continue`) is supported.

---

### Test Reliability Issue: gpt-5.5 Rate Limiting

**Root Cause (from DB analysis)**: Most "failed" or "inconclusive" tests were using gpt-5.5 which is rate-limited. When gpt-5.5 hits OpenAI rate limits:

1. Session creates with `tokens_input: 0, output: 0` — agent starts but is cut off before processing
2. In the observed primary session, only the user message existed; there was no completed assistant message
3. The fallback session completed successfully, but the adapter misclassified it because stdout omitted `step_finish`

**DB Pattern**:
```
gpt-5.5 primary → tokens: 0,0 → no assistant completion
deepseek-v4-flash-free fallback → tokens: 8228,5 → assistant finish="stop" and step-finish part
```

**Config Impact**: `config.json` sets `sprintExecutionModel: "openai/gpt-5.5"` with `sprintExecutionVariant: "low"`. The `agentwrap-run` tool uses this model by default, causing all tests to hit rate limits.

**Workaround**: For reliable testing, use `opencode/deepseek-v4-flash-free` model explicitly or wait for gpt-5.5 rate limit to reset.

---

### Bug: `step_finish` Event Not Reliably Emitted by opencode CLI

**Severity**: Critical (misclassifies successful runs as failed)

**Problem**: The agentwrap opencode adapter expected stdout `step_finish` events to set `sawFinal = true` (which signals successful completion). In the tested OpenCode binary/source behavior, successful runs can persist `step-finish` internally without emitting a corresponding stdout JSON event.

**Evidence**:
```bash
$ opencode run "What is 2+2?" --format json
{"type":"step_start",...}
{"type":"text",...}
# No step_finish event emitted
```

**DB Confirmation**: Sessions complete successfully with `finish: "stop"` in message data, but `step_finish` parts exist in DB (confirmed via `opencode db "SELECT id, data FROM part WHERE data LIKE '%step-finish%'"`).

**Source Code Analysis** (`ultraplan/studies/go-cli-study/sources/opencode`):

The Go OpenCode source exposes structured completion internally through `internal/llm/agent.AgentEvent`. `CoderAgent.Run()` returns a final event with either:

- `Type: response`, `Done: true`, and a completed `message.Message`
- `Type: error` and an underlying provider error

However, the non-interactive CLI boundary in `internal/app/app.go` consumes that final event and prints only formatted assistant content through `format.FormatOutput`. The structured completion/error information is not exposed as a stable JSON stdout contract. Provider retry exhaustion is also collapsed to plain errors such as `maximum retry attempts reached for rate limit: 8 retries` before reaching the process boundary.

**Why cancel tests worked**: They trigger context cancellation before the completion check, so `sawFinal` is irrelevant.

---

## Recommended Fixes for agentwrap

### Fix Option 1: Detect completion via process exit + assistant output events (Applied)

Modify `opencode/runtime.go` to treat successful process exit (code 0) + received text events as completion, without requiring `step_finish`.

Location: `finalResult()` method around line 305

Current code:
```go
} else if !r.sawFinal {
    sdkErr = agentwrap.NewError(agentwrap.ErrorRuntimeExit, "opencode run",
        "OpenCode finished without a final structured result", nil, ...)
```

**Applied fix**:
```go
} else if !r.sawFinal {
    if r.sawOutput {
        status = agentwrap.StatusCompleted
        sdkErr = nil
        warnings = append(warnings,
            "OpenCode exited cleanly after emitting assistant output but did not emit a final structured result")
    } else {
        sdkErr = agentwrap.NewError(agentwrap.ErrorRuntimeExit, "opencode run",
            "OpenCode finished without a final structured result", nil, ...)
    }
}
```

`sawOutput` is set when `text` or `reasoning` events are projected. Empty clean exits without `step_finish` still fail as `runtime_exit`.

### Fix Option 2: Detect completion via session message with `finish` field

Check the DB for `finish` field in the session messages to confirm the turn completed. This was partially applied after real harness runs showed that stdout may omit both terminal and assistant-output events. The adapter now queries OpenCode's DB for the captured session ID on ambiguous clean exits and completes when session usage is nonzero or a completed assistant message is present.

### Fix Option 3: Add `--exit-complete` flag (Not recommended - requires opencode CLI change)

This would require modifying opencode to emit a completion event, which is outside the agentwrap adapter scope.

---

## OpenCode Source Constraint

The wrapper cannot change OpenCode. The correct integration boundary is therefore to treat OpenCode stdout as a progress stream rather than a complete final-state contract, and to make the adapter tolerant of missing terminal `step_finish` events when the process exits cleanly after assistant output.

---

## Recommendations (Priority Order)

1. **Fix step_finish event bug** — Critical. Applied in local `agentwrap`: track assistant output and complete on clean exit when stdout omits `step_finish`; emit a warning.

2. **Fix module path mismatch** — Critical DX blocker
3. **Add `TimeoutMs int64` field to `RunRequest`** — Eliminates nanosecond conversion bugs
4. **Add `--variant` support to OpenCode adapter** — Feature parity with CLI
5. **Add `Runtime.Preflight()` convenience method** — Reduces health check boilerplate
6. **Return interface from `NewRuntime()`** — Improves testability
7. **Add rate limit callback/handler option** — Enables proactive fallback
8. **Improve rate-limit classification** — Added regression coverage for OpenCode's max-retry rate-limit stderr string. Further improvement would require a stable OpenCode CLI error contract or DB reconciliation for zero-token/no-assistant sessions.
9. **Consider `BoundedRunner` for concurrent run management** — Enables batch orchestration patterns

---

## Implementation Change Record

This section records the concrete local changes made after investigating the report and rerunning the real `agentwrap-run` OpenCode harness.

### agentwrap changes

Changed files:

- `/home/antonioborgerees/coding/agentwrap/opencode/projector.go`
- `/home/antonioborgerees/coding/agentwrap/opencode/runtime.go`
- `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go`

#### 1. Track assistant output separately from terminal final events

Problem:

- The adapter previously only considered a run successful when a projected OpenCode event had category `EventFinalResult`.
- That only happened when stdout included native event type `step_finish`.
- Real OpenCode runs can complete successfully without stdout containing `step_finish`.

Change:

- Added an `output bool` field to the OpenCode projection result.
- Set it when native stdout event type is `text` or `reasoning`.
- Added `sawOutput bool` to the OpenCode run state.
- Set `sawOutput` while streaming projected records.

Reasoning:

- `text` and `reasoning` are weaker than `step_finish`, but when the process exits cleanly they prove the assistant produced output.
- This prevents a clean completed response from being misclassified as `runtime_exit` only because the terminal event was missing.

#### 2. Complete clean exits with assistant output when `step_finish` is absent

Problem:

- `finalResult()` returned `ErrorRuntimeExit` for `!sawFinal` even when OpenCode exited with code `0` after assistant output.

Change:

- In `finalResult()`, the `!sawFinal` branch now checks `sawOutput`.
- If `sawOutput` is true, the run remains `StatusCompleted` instead of becoming `StatusFailed`.
- The run records a warning:

```
OpenCode exited cleanly after emitting assistant output but did not emit a final structured result
```

Reasoning:

- This keeps stdout `step_finish` as the preferred signal.
- It treats clean-exit assistant output as a compatible fallback for current OpenCode behavior.
- Empty clean exits without final/output still fail as `runtime_exit`.

#### 3. Add OpenCode DB reconciliation for ambiguous clean exits

Problem discovered during real harness reruns:

- `./agentwrap-run usage --model opencode/deepseek-v4-flash-free` completed in OpenCode and persisted token usage, but stdout still did not include enough events for the adapter.
- DB showed the captured session had nonzero token usage and a completed assistant message with `finish:"stop"`.

Change:

- Added `reconcileFinalState()` in `opencode/runtime.go`.
- It runs only on ambiguous clean exits where stdout did not include `step_finish`.
- It uses the captured session ID and calls:

```
opencode db --format json "<query>"
```

- It queries the `session` table for `tokens_input`, `tokens_output`, and `tokens_reasoning`.
- It treats nonzero persisted token usage as completed and projects it into `RunResult.Usage`.
- It also queries `message` data and treats an assistant message with `finish` as completed.
- The DB query timeout was set to `10s` because the local OpenCode DB is large and shorter timeouts missed completed state during real runs.

Reasoning:

- We cannot change OpenCode source.
- OpenCode's SQLite DB is the durable local runtime state that already records the truth when stdout is incomplete.
- DB reconciliation is used only after process exit and only when stdout lacks a final event, so stdout remains the streaming/progress path.

#### 4. Preserve cancellation semantics when process exit races cancellation

Problem discovered during real harness reruns:

- `./agentwrap-run cancel` called `Run.Cancel()` successfully, but `Wait()` could still report `runtime_exit` if process exit won the race before context cancellation was observed.

Change:

- `finalResult()` now checks whether the run lifecycle is already `StatusCancelled` before classifying process exit as `runtime_exit`.
- If the lifecycle is cancelled, it returns `ErrorCancellation` and `StatusCancelled`.

Reasoning:

- Caller intent and adapter lifecycle should win over a subprocess exit status caused by cancellation.

#### 5. Improve rate-limit classification coverage

Problem:

- OpenCode provider retry exhaustion can cross the CLI boundary as a plain error string rather than structured JSON.
- Real MiniMax rate limits can also be written only to OpenCode's internal log while the `opencode run` process remains alive until the wrapper timeout fires.

Change:

- Added regression coverage for stderr text:

```
agent processing failed: failed to process events: maximum retry attempts reached for rate limit: 8 retries
```

- Added classification coverage for MiniMax/OpenCode log text containing `statusCode:429`, `rate_limit_error`, and `usage limit exceeded`.
- Added a timeout-time diagnostic pass in `opencode/runtime.go` that scans recent OpenCode log files under `$XDG_DATA_HOME/opencode/log` or `~/.local/share/opencode/log`.
- The scan is constrained to logs modified after the run started and matching the captured session ID or requested model.
- If the matching log contains a rate-limit signal, the run is classified as `ErrorRateLimit` instead of generic `ErrorTimeout`, and `rate_limit_info` is recorded in native metadata.

Expected classification:

- `ErrorRateLimit`

Reasoning:

- OpenCode does not expose stable structured error codes at the CLI boundary.
- The adapter must classify known human-readable rate-limit strings when they are present.
- When OpenCode logs the provider 429 but does not exit, timeout is only the symptom; the actionable cause is still provider rate limiting.

#### 6. Regression tests added/updated

Added adapter tests covering:

- Clean exit + assistant output + no `step_finish` completes with warning.
- Clean exit + no assistant output + no `step_finish` still fails as `runtime_exit`.
- Max-retry rate-limit stderr string classifies as `rate_limit`.
- MiniMax `rate_limit_error` / `usage limit exceeded` log text classifies as `rate_limit`.
- Timeout with a matching recent OpenCode rate-limit log classifies as `rate_limit` and records `rate_limit_info`.
- Cancelled process exit classifies as `cancellation`, not `runtime_exit`.

### agentwrap-smoke changes

Changed files:

- `/home/antonioborgerees/coding/agentwrap-smoke/AGENTWRAP_REPORTING.md`
- `/home/antonioborgerees/coding/agentwrap-smoke/internal/init/init.go`
- `/home/antonioborgerees/coding/agentwrap-smoke/cmd/study/main.go`
- `/home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go`
- `/home/antonioborgerees/coding/agentwrap-smoke/agentwrap-run` rebuilt locally
- `/home/antonioborgerees/coding/agentwrap-smoke/AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md`

#### 1. Updated report diagnosis

The report previously said:

- `PolicyRunner` did not trigger fallback.
- The result was a rate-limit fallback trigger problem.

Updated finding:

- `PolicyRunner` did trigger fallback.
- The fallback model completed successfully in OpenCode.
- The adapter misclassified the successful fallback as `runtime_exit` because stdout omitted terminal completion events.

Evidence recorded:

- Primary `gpt-5.5` session had zero tokens and no completed assistant output.
- Fallback `deepseek-v4-flash-free` session had nonzero tokens, assistant `finish:"stop"`, and a persisted `step-finish` part.

#### 2. Fixed existing vet/build blockers

Files:

- `internal/init/init.go`
- `cmd/study/main.go`

Change:

- Replaced `fmt.Println` calls whose argument strings ended in redundant newlines with separate `fmt.Println()` calls.

Reasoning:

- `go test ./...` failed vet/build checks before these unrelated existing issues were fixed.
- The change is formatting-only and preserves CLI output.

#### 3. Rebuilt real harness binary

#### 3. Fixed custom validator harness strictness

File:

- `cmd/agentwrap-run/main.go`

Change:

- The custom validator previously required `reports/custom.txt` to equal exactly `VALID`.
- Real OpenCode wrote `VALID\n`, which is a normal text-file result for this prompt.
- The validator now compares `strings.TrimSpace(string(content)) == "VALID"`.

Reasoning:

- The scenario is meant to test `ValidatorFunc` integration, not whether the model omits a trailing newline.
- After this change, `./agentwrap-run custom-validator` completes successfully.

#### 4. Rebuilt real harness binary

Command:

```
GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
```

Reasoning:

- `agentwrap-smoke` uses a local replace for `agentwrap`, but the checked-in/previous `agentwrap-run` binary was stale.
- Rebuilding was required before real scenario runs exercised the patched adapter.
- `-buildvcs=false` was required because `/home/antonioborgerees/coding/agentwrap-smoke` is not itself a git repository.

#### 5. Made rate-limit harness model-selectable and more observable

File:

- `cmd/agentwrap-run/main.go`

Change:

- `rate-limit` now accepts:

```
--primary-model
--fallback-model
```

- Defaults remain:

```
--primary-model opencode/gpt-5.5
--fallback-model opencode/deepseek-v4-flash-free
```

- `results.json` now records `result.Metadata.Attempts`.
- `results.json` sets `fallback_used` and `retry_count` when more than one policy attempt ran.
- The `rate-limit` command prints each attempt's target index, model, status, error category, and rate-limit detail when present.

Reasoning:

- The original harness hardcoded `opencode/gpt-5.5`, which stopped being useful once OpenAI usage returned.
- The user hit a real MiniMax limit, so the harness needed to test `minimax-coding-plan/MiniMax-M2.7` directly without source edits.
- Attempt-level output is required to prove whether a completed policy run used primary or fallback.

### Important runtime notes

- Running real OpenCode scenarios inside the sandbox can fail because OpenCode writes to `~/.local/share/opencode/opencode.db`.
- Parallel OpenCode real runs caused SQLite checkpoint errors:

```
Failed to run the query 'PRAGMA wal_checkpoint(PASSIVE)'
```

- Real OpenCode harness runs should be run sequentially and outside the sandbox when validating adapter behavior.
- The `rate-limit` scenario is currently not a stable reproduction because OpenAI usage is available again; fallback/rate-limit behavior should be retested with an actually limited provider or a controlled provider error.

---

## Real Harness Rerun After Fixes

The `agentwrap-run` binary was rebuilt with the local `agentwrap` replace after adapter changes:

```
GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
```

Real OpenCode runs need filesystem access to `~/.local/share/opencode`; sandboxed runs can fail with SQLite checkpoint errors such as `Failed to run the query 'PRAGMA wal_checkpoint(PASSIVE)'`. The following runs were executed outside the sandbox and sequentially to avoid OpenCode DB contention.

| Command | Result |
|---|---|
| `./agentwrap-run usage --model opencode/deepseek-v4-flash-free` | completed; usage projected as `Input=8231 Output=2 Total=8249` |
| `./agentwrap-run artifacts --model opencode/deepseek-v4-flash-free` | completed; artifacts observed `0` |
| `./agentwrap-run fixed-backoff --model opencode/deepseek-v4-flash-free` | completed |
| `./agentwrap-run validate-json` | completed |
| `./agentwrap-run validate-md` | completed |
| `./agentwrap-run custom-validator` | completed after harness validator was relaxed to trim trailing whitespace |
| `./agentwrap-run cancel` | completed as expected with `Category: cancellation`, `Final Status: cancelled` |
| `./agentwrap-run health-fail` | expected health failures reported |
| `./agentwrap-run session-fork` | expected unsupported capability error reported |
| `./agentwrap-run timeout` | expected timeout reported |
| `./agentwrap-run rate-limit --primary-model minimax-coding-plan/MiniMax-M2.7 --fallback-model opencode/deepseek-v4-flash-free` | completed on attempt 1 after MiniMax reset; no fallback used |

MiniMax rate-limit evidence captured before reset:

```
service=llm providerID=minimax-coding-plan modelID=MiniMax-M2.7
statusCode=429
responseBody={"type":"error","error":{"type":"rate_limit_error","message":"usage limit exceeded, 5-hour usage limit reached for Token Plan Starter (1500/1500 used), resets at 2026-05-20T20:00:00Z (2056)"}}
```

Observed wrapper behavior before the fix:

```
./agentwrap-run usage --model minimax-coding-plan/MiniMax-M2.7
Wait error: opencode run: timeout: OpenCode run timed out
Final Status: failed
```

Diagnosis:

- OpenCode wrote the provider 429 to `~/.local/share/opencode/log/2026-05-20T195715.log`.
- The CLI process did not exit promptly after the provider stream error.
- The wrapper therefore hit its timeout path and reported `timeout`, losing the actionable `rate_limit` cause.

Fix:

- `agentwrap/opencode` now scans matching recent OpenCode logs on timeout.
- The MiniMax log shape is covered by unit tests.
- A future live MiniMax limit should classify as `rate_limit` and allow `PolicyRunner` to trigger fallback.

Verification commands:

```
GOCACHE=/tmp/agentwrap-go-build go test ./...
GOCACHE=/tmp/agentwrap-smoke-build go test ./...
```

Both pass.

---

## Real OpenCode Robustness Test Results (2026-05-20)

### Harness Improvements Implemented

The following harness improvements from the test plan are present in `cmd/agentwrap-run/main.go`:

| Improvement | Status | What is actually verified |
|---|---|---|
| H1: smoke-all command | ✅ | `cmdSmokeAll` exists and writes a line-delimited `summary.json` file |
| H2: Expectation flags | ✅ | `--expect-status` and `--expect-category` exist on cancellation/timeout scenarios |
| H3: `--model` on smoke commands | ✅ | Added for the main real-run scenarios |
| H4: Richer `results.json` | ✅ | `saveResults()` now persists run ID, session ID, warnings, usage, attempts, metadata, and cleanup state |
| H5: OpenCode DB snapshots | Partial | Snapshots are written for some scenarios that call `saveOpenCodeDB()`, not literally every scenario |
| H6: Event capture by default | Partial | Some scenarios write `events.jsonl`; this is not universal yet |

### New Test Commands Added

- `smoke-text`
- `smoke-reasoning`
- `smoke-file-write`
- `validate-fail`
- `validate-repair`
- `validate-repair-exhaust`
- `fallback-invalid-model`
- `fallback-invalid-provider`
- `fallback-all-fail`
- `health-model`
- `smoke-all`

### Verified Evidence Sets

There are two distinct real-run evidence sets and they should not be conflated:

1. **2026-05-20 stable smoke-all run**
   Directory:
   `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260520-212857`

2. **2026-05-21 infrastructure-invalid smoke-all rerun**
   Directory:
   `/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260521-112624`

The 2026-05-21 rerun was dominated by OpenCode DB checkpoint failures:

```
Failed to run the query 'PRAGMA wal_checkpoint(PASSIVE)'
```

Those are environment/infrastructure failures at the OpenCode boundary. They are not valid evidence that the wrapper logic regressed across all scenarios.

### 2026-05-20 Smoke-All Summary

The authoritative artifact for the 2026-05-20 run is the line-delimited file:

`/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260520-212857/summary.json`

It records:

- 17 scenarios with `passed:true`
- 3 scenarios with `passed:false`

The three `passed:false` entries were:

- `validate-repair-exhaust`
- `health-fail`
- `session-fork`

However, the last two are harness-classification issues, not semantic scenario failures:

- `health-fail` log shows the expected health failures were reported.
- `session-fork` log shows the expected unsupported-session error was reported.

Those logs are:

- [healthfail.log](/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260520-212857/health-fail/healthfail.log)
- [sessionfork.log](/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260520-212857/session-fork/sessionfork.log)

So the correct interpretation of the 2026-05-20 run is:

- most real scenarios succeeded
- `validate-repair-exhaust` genuinely did not meet expectation
- `health-fail` and `session-fork` were reported misleadingly by the smoke-all aggregator

### 2026-05-20 Scenario Assessment

#### Confirmed good scenarios

- `smoke-text`: completed, usage present, warning explains missing final structured result
- `smoke-reasoning`: completed
- `smoke-file-write`: completed
- `usage`: completed with DB-projected usage
- `artifacts`: completed
- `fixed-backoff`: completed
- `validate-json`: completed
- `validate-md`: completed
- `custom-validator`: completed
- `validate-fail`: failed with `validation` as expected
- `validate-repair`: completed
- `cancel`: cancelled with `cancellation` as expected
- `timeout`: failed with `timeout` as expected
- `fallback-invalid-model`: completed via fallback, attempts captured
- `fallback-invalid-provider`: completed via fallback, attempts captured
- `fallback-all-fail`: failed as expected
- `health-model`: completed as a health-check command

#### Genuine unresolved scenario

- `validate-repair-exhaust`

The saved artifact does **not** support the earlier explanation that two repair attempts ran and then the timeout fired. The actual artifact shows:

- `Repair attempts: 1`
- `Repair exhausted: false`
- final category `timeout`

So the only defensible conclusion for the 2026-05-20 run is:

- the scenario failed before producing `repair_exhausted`
- the prior report text over-interpreted the cause

### 2026-05-21 Rerun Assessment

On 2026-05-21, a fresh `smoke-all` rerun failed broadly because OpenCode itself was failing very early with:

```
Failed to run the query 'PRAGMA wal_checkpoint(PASSIVE)'
```

This affected the majority of real-run scenarios before they could establish a session or emit meaningful output. The evidence is visible directly in the saved `results.json` files for `smoke-text`, `smoke-reasoning`, `smoke-file-write`, `usage`, `artifacts`, and many others under:

`/home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/smoke-all-20260521-112624`

What this rerun does confirm:

- the smoke-all aggregator now reports `health-fail` as `completed`
- the smoke-all aggregator now reports `session-fork` as `failed` with category `configuration`

What it does **not** confirm:

- any broad regression in wrapper logic
- any clean pass/fail judgment about most real OpenCode scenarios

The environment was invalid for that purpose.

### Harness Ambiguities Fixed or Clarified

1. `smoke-all` reporting for non-run commands:
   `health-fail` and `session-fork` are now derived from their logs when no `RunResult` status exists.

2. `validate-repair-exhaust` report text:
   corrected to match the saved artifact rather than an inferred “two repairs then timeout” story.

3. Event/DB capture claims:
   downgraded from universal claims to partial claims, because the artifact set does not prove “every scenario” coverage.

### Remaining Issues

1. **`validate-repair-exhaust` is still not deterministic**
   Under the stable 2026-05-20 evidence set it timed out before exhaustion.
   Under the 2026-05-21 rerun it failed even earlier due OpenCode DB checkpoint failure.

2. **OpenCode DB checkpoint instability is a real external blocker**
   The rerun produced repeated:
   `Failed to run the query 'PRAGMA wal_checkpoint(PASSIVE)'`
   That invalidates most scenario conclusions from the 2026-05-21 run.

3. **Event capture is not universal**
   The codebase supports `events.jsonl`, but the current artifacts do not justify saying every scenario produced one.

4. **OpenCode DB snapshots are not universal**
   Some scenarios have `opencode-session.json`, `opencode-messages.json`, and `opencode-parts.json`; others do not.

### Current Verdict

The strongest evidence remains the 2026-05-20 run, not the 2026-05-21 rerun.

Current confidence:

- adapter fixes for missing-final completion, cancellation classification, usage reconciliation, and fallback attempt metadata are supported by real artifacts
- the smoke harness is better than before, but its robustness section must not claim universal pass coverage
- `validate-repair-exhaust` remains unresolved
- OpenCode DB checkpoint instability is a live environmental risk for real-run validation

### Verification

```bash
GOCACHE=/tmp/agentwrap-go-build go test ./...
GOCACHE=/tmp/agentwrap-smoke-build go test ./...
GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
```
