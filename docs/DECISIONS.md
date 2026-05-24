# Decisions

This file records behavior contracts that future adapter, harness, and report changes must preserve.

## D-0001: Final Event Versus Non-Zero Exit

Status: proposed

Question: if OpenCode emits a final structured completion event but the process exits non-zero, should `agentwrap` report `completed` or `failed`?

Current leaning: report `completed` when the final event is strong enough to prove terminal success, and include a warning plus native process-exit metadata. Do not silently discard the non-zero exit.

Related issues:
- `I-0001`

## D-0002: Timeout With Durable DB Completion

Status: proposed

Question: if the caller deadline is exceeded but OpenCode durable state later shows terminal assistant finish, should the wrapper report timeout or completion?

Current leaning: keep status `failed` with category `timeout`, but include DB terminal-finish evidence in native metadata. A caller timeout is a real contract failure even if the child eventually completed.

Related issues:
- `I-0004`

## D-0003: Assistant Output Without Final Event Or DB Proof

Status: proposed

Question: if OpenCode exits cleanly with assistant text but no final structured event and no DB terminal proof, is that enough to mark completed?

Current leaning: completed-with-warning is acceptable only if the adapter can prove useful assistant output was emitted and there is no contradictory process or provider evidence. Otherwise prefer `runtime_exit`.

Related issues:
- `I-0005`

## D-0004: Fallback Signals Sufficient For Completion

Status: accepted

Question: which fallback signals are sufficient to treat an OpenCode run as completed when stdout lacks a final structured event?

Accepted fallback order (strongest to weakest):

1. **Final structured event**: strongest. Report completed unless stronger contradictory evidence exists, such as provider/rate-limit classification.
2. **Assistant output on clean exit**: report completed with warning when there is useful assistant output and no contradictory process/provider evidence.
3. **DB terminal finish evidence**: report completed with warning when durable OpenCode state proves a terminal assistant finish and there is no caller timeout or stronger process/provider failure.
4. **No final event, no output, no DB proof**: report failed/runtime_exit.

Timeout remains governed by D-0002: durable DB evidence may be recorded, but does not silently convert a caller deadline into success.

Current adapter behavior: final event and clean-output fallback are implemented; DB fallback behavior remains covered by I-0010/D-0008 testability work.

Related issues:
- `I-0006`

## D-0005: Invalid Model Classification

Status: proposed

Question: should "Model not found" from OpenCode classify as `model_unavailable` or remain `runtime_exit`?

Current leaning: classify as `model_unavailable` when OpenCode explicitly reports "Model not found". This is more actionable than generic `runtime_exit`.

Related issues:
- `I-0009`

## D-0006: Invalid Provider Pre-Validation

Status: proposed

Question: should `agentwrap` pre-validate provider format before invoking OpenCode?

Current leaning: yes, for known-invalid provider formats. Return `configuration` error before process start. OpenCode should still handle unknown providers at runtime, but the wrapper should catch syntactically invalid provider strings early.

Related issues:
- `I-0009`

## D-0007: Provider Auth vs Rate-Limit Classification

Status: proposed

Question: should provider authentication failures be distinguished from rate limits?

Current leaning: yes, but only after collecting stable real error examples. Do not guess from a single provider's error format. Auth failures are typically permanent and should not trigger retry/fallback the same way rate limits do.

Related issues:
- `I-0009`

## D-0008: DB Reconciliation Unit Testability

Status: proposed

Question: should `reconcileFinalState()` be extracted into a smaller testable helper that accepts DB query responses as input?

Current leaning: yes. The current coupling to shelling out to `opencode db` makes unit testing impossible with the fake runner. A DB query interface or function parameter would enable direct unit tests for non-JSON, timeout, locked-DB, and terminal-finish responses.

Related issues:
- `I-0010`
