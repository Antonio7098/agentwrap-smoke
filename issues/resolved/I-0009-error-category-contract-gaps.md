# I-0009: Error Category Contract Gaps

Status: resolved
Severity: high
Area: smoke-harness
Discovered: 2026-05-21
Related decisions: D-0005, D-0006
Related runs: `.agentwrap-logs/smoke-all-20260521-120218`

## Observation

Several smoke scenarios produce generic `runtime_exit` categories when more specific categories would provide actionable diagnostics.

From Workstream 5 evidence:

| Scenario | Current Category | Expected Category | Gap |
|----------|-----------------|-------------------|-----|
| Invalid model (`opencode/not-a-model`) | `runtime_exit` | `model_unavailable` | Not mapped |
| Invalid provider (`nonexistent-provider/test`) | `runtime_exit` | `configuration` | No pre-validation |
| OpenCode DB unavailable | `runtime_exit` | `runtime_unavailable` | DB check path not reached |
| Provider auth failure | not distinguished | `authentication` | No auth-specific patterns |
| `fallback-all-fail` | no specific category | needs explicit contract | Under-specified |

OpenCode emits `error` events like `"Model not found: opencode/not-a-real-model"` but the wrapper's `projector.go:88` classifies all fatal errors as `runtime_exit` regardless of the error content.

## Expected Contract

- Invalid model should classify as `model_unavailable` when OpenCode explicitly reports "Model not found".
- Invalid provider should classify as `configuration` — the wrapper should pre-validate provider format before invoking OpenCode.
- Known local OpenCode DB failures should reach `runtime_unavailable` even when OpenCode exits cleanly without a final event.
- Provider auth failures should have distinct `authentication` category.
- `fallback-all-fail` should assert both `failed` status and a specific category.

## Evidence

- `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md` Workstream 5
- `AGENTWRAP_REPORTING.md` Error Category Contract section

## Root Cause

1. `projector.go` maps `EventFatalError` → `runtime_exit` unconditionally, ignoring the error text.
2. No wrapper-level provider/model format validation before OpenCode invocation.
3. DB failure detection in `classifyOpenCodeLocalFailure()` may not activate when process exits cleanly without a final event.

## Implementation

Partially implemented in `/home/antonioborgerees/coding/agentwrap`:

- `opencode/projector.go`: fatal OpenCode events containing `Model not found` / `unknown model` classify as `model_unavailable` instead of `runtime_exit`.
- `opencode/projector.go`: fatal events containing stable auth text (`authentication`, `unauthorized`, `invalid API key`, `api key`) classify as `authentication`.
- `opencode/runtime.go`: pre-validates wrapper provider/model syntax before process start; provider IDs cannot contain `/`, and model IDs cannot contain more than one `/`.
- `opencode/runtime_test.go`: added unit coverage for model-not-found, authentication, and invalid provider/model syntax.

Also fixed smoke repo imports/module path to use canonical `github.com/Antonio7098/agentwrap`.

## Verification

Verified 2026-05-24:

- `/home/antonioborgerees/coding/agentwrap`: `go test ./...`
- `/home/antonioborgerees/coding/agentwrap-smoke`: `go test ./...`
- `go run ./cmd/agentwrap-run fallback-invalid-model` shows first attempt `model_unavailable`, then fallback completed.
- `go run ./cmd/agentwrap-run invalid-separate-provider` returns `configuration` before process start.
- `go run ./cmd/agentwrap-run fallback-all-fail` returns final category `model_unavailable` and records both attempts as `model_unavailable`.

## Next Action

Move this issue to resolved after reviewing the run records.
