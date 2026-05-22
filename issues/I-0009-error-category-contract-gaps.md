# I-0009: Error Category Contract Gaps

Status: open
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

No implementation recorded in this repo yet.

## Verification

Not verified.

## Next Action

1. Add pattern matching in `projector.go` or `runtime.go` to map "model not found" to `model_unavailable`.
2. Add provider format validation in wrapper before OpenCode invocation.
3. Ensure DB failure detection is reached even on clean exits without final events.
4. Add `authentication` pattern matching based on real provider error examples.
