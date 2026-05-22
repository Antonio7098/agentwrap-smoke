# I-0002: Provider Rate-Limit Evidence Is Non-Deterministic

Status: monitoring
Severity: medium
Area: rate-limit
Discovered: 2026-05-21
Related decisions: none
Related runs: `.agentwrap-logs/ratelimit-20260521-120501`

## Observation

The MiniMax targeted run did not rate-limit during the documented pass, so fallback was not exercised by that provider state.

This does not invalidate unit coverage for rate-limit text classification, but it means a real provider run can be inconclusive for fallback behavior.

## Expected Contract

Real provider rate-limit tests must distinguish:

- `verified`: provider actually returned a rate-limit shape and fallback behaved correctly.
- `inconclusive`: provider completed successfully, so fallback was not exercised.
- `failed`: provider returned a rate-limit shape and wrapper misclassified or mishandled it.

## Evidence

Source report:

- `AGENTWRAP_REPORTING.md`

Run evidence:

- `.agentwrap-logs/ratelimit-20260521-120501/ratelimit.log`
- `.agentwrap-logs/ratelimit-20260521-120501/results.json`

Observed result:

- Final status: `completed`
- Attempt 1 model: `minimax-coding-plan/MiniMax-M2.7`
- Attempt 1 status: `completed`
- Fallback: not exercised

## Implementation

No code fix required for this observation by itself.

Preferred improvement: add a deterministic fake OpenCode or harness fixture that emits provider 429 / MiniMax usage-limit shapes without relying on live provider account state.

## Verification

Monitoring only. Existing unit tests cover known classifier shapes; real provider evidence remains stateful.

## Next Action

Record the next real provider 429 as a run, or add the deterministic fixture described in `AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md`.

