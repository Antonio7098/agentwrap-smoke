# Current State

Last updated: 2026-05-22

## Latest Verified Baseline

- **Date**: 2026-05-21
- **Smoke run**: `R-20260521-001` (`.agentwrap-logs/smoke-all-20260521-120218`)
- **Result**: 20/20 scenarios passed
- **Adapter state**: sawOutput completion, cancellation precedence, DB reconciliation, rate-limit log scanning applied
- **Note**: `validate-repair-exhaust` timed out before exhaustion (unresolved); health-fail/session-fork are harness artifacts, not semantic failures

## Open Issues

| ID | Status | Severity | Area | Summary | Next Action |
|----|--------|----------|------|---------|-------------|
| `I-0001` | open | high | process-boundary | Non-zero exit may override final structured event | Decide D-0001, verify with unit test |
| `I-0004` | open | medium | timeout | Timeout can race durable DB terminal finish | Decide D-0002, verify adapter metadata |
| `I-0005` | open | medium | process-boundary | Clean exit + output but no final event or DB proof | Decide D-0003, align adapter behavior |
| `I-0006` | open | critical | process-boundary | OpenCode JSON mode lacks final structured event (root cause of all session gaps) | Decide D-0004, ensure smoke tests use output/DB fallback |
| `I-0007` | open | medium | rate-limit | Timeout log classification modtime cutoff too tight | Fix cutoff window or add content-based matching |
| `I-0008` | open | medium | rate-limit | MiniMax nested JSON "usage limit exceeded" not parsed | Add responseBody extraction or direct string patterns |
| `I-0009` | open | high | smoke-harness | Invalid model/provider categories not specific (runtime_exit vs model_unavailable/configuration) | Implement mapping for "Model not found", add provider pre-validation |
| `I-0010` | open | high | db-reconciliation | DB reconciliation not unit-testable with fake runner | Extract reconciler helper, add direct unit tests |
| `I-0011` | open | medium | fallback | Deterministic rate-limit fallback fixture missing | Add RUN_MODE=rate_limit to fake-opencode, add PolicyRunner test |
| `I-0013` | open | medium | cancellation | Cancellation may not preserve OpenCode's final cancellation outcome | Audit `Cancel()` against OpenCode cancellation lifecycle |
| `I-0014` | open | high | end-of-run | `step_finish` may be too coarse as final signal | Audit event payloads for idle, finish reasons, trailing events |
| `I-0002` | monitoring | medium | rate-limit | Real provider rate limits non-deterministic | Record next real 429, or add fixture (I-0011) |
| `I-0003` | open | medium | reporting | Historical reporting concentrated in large top-level files | Backfill complete, archive source docs |
| `I-0012` | open | low | reporting | Archive AGENTWRAP_*.md source documents | Move to docs/archive/ |

## Recently Verified

| ID | Finding | Fix | Verified By |
|----|---------|-----|-------------|
| `I-0001` (partial) | Non-zero exit bug | sawFinal takes precedence over proc.ExitCode | `R-20260521-004` process-group-nonzero-final: completed despite exit code 7 |
| `I-0005` (partial) | Output without final event | sawOutput → completed with warning on clean exit | `R-20260521-001` smoke-text: completed with warning |
| `I-0006` (partial) | Missing final event | DB reconciliation fallback | `R-20260521-001` usage: completed with DB-projected usage |
| — | DB snapshot validation | JSON validation in saveOpenCodeDB | `R-20260521-003` db-non-json: status=failed (was captured) |
| — | DB query timeout | 5s per-file context deadline | `R-20260521-003` db-timeout mechanism in place |
| — | Session ID alignment | Consistent ses_fake123 across fixtures | `R-20260521-003` all DB scenarios coherent |

## Active Contracts To Decide

| ID | Question | Options | Needed Evidence |
|----|----------|---------|-----------------|
| `D-0001` | Final event vs non-zero exit | completed vs failed | Unit test: process exits 0 vs non-zero with identical event stream |
| `D-0002` | Timeout + DB terminal finish | timeout with evidence vs completed | Focused timeout test with DB session that has terminal finish |
| `D-0003` | Output alone enough for completion | completed-with-warning vs runtime_exit | Real smoke where output exists but DB reconciliation is unavailable |
| `D-0004` | Which fallback signals sufficient | Order: final event > output > DB > none | Decision tree for I-0006 resolution |
| `D-0005` | Model not found → model_unavailable | Map text vs keep runtime_exit | OpenCode error message analysis |
| `D-0006` | Provider pre-validation | configuration vs runtime_exit | Provider format validation coverage |
| `D-0008` | Extract DB reconciler | Interface param vs keep subprocess call | Feasibility of extracting query interface |

## Next Recommended Work

1. **Highest impact**: Decide D-0004 (completion fallback signals) since it governs all session and process-boundary smoke.
2. **Unit gap**: Fix I-0008 (MiniMax nested JSON) — simplest measurable fix with existing test.
3. **Unit gap**: Fix I-0007 (log classification timing) — extends existing test stability.
4. **Lifecycle audit**: Use `/home/antonioborgerees/coding/ultraplan/targets/agentwrap/reports/opencode-cancellation-failures-end-of-run.md` to drive I-0013 and I-0014.
5. **Verification**: Run session smoke commands (`session-fresh`, `session-continue-existing`) after D-0004 and adapter alignment.
