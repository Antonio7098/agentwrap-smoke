# Current State

Last updated: 2026-05-24

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
| `I-0011` | open | medium | fallback | Deterministic rate-limit fallback fixture missing | Add RUN_MODE=rate_limit to fake-opencode, add PolicyRunner test |
| `I-0002` | monitoring | medium | rate-limit | Real provider rate limits non-deterministic | Record next real 429, or add fixture (I-0011) |
| `I-0003` | open | medium | reporting | Historical reporting concentrated in large top-level files | Backfill complete, archive source docs |
| `I-0012` | open | low | reporting | Archive AGENTWRAP_*.md source documents | Move to docs/archive/ |
| `I-0015` | open | medium | transport | Evaluate OpenCode ACP / HTTP-SSE transport | Spike SDK spike run (R-20260524-006) |
| `I-0016` | open | critical | rate-limit | Rate-limit classifier ignores nested `error.type: rate_limit_error` | Spike fix in `rate_limit.go`, add unit test |

## Recently Verified

| ID | Finding | Fix | Verified By |
|----|---------|-----|-------------|
| `I-0001` (partial) | Non-zero exit bug | sawFinal takes precedence over proc.ExitCode | `R-20260521-004` process-group-nonzero-final: completed despite exit code 7 |
| `I-0005` (partial) | Output without final event | sawOutput → completed with warning on clean exit | `R-20260521-001` smoke-text: completed with warning |
| `I-0006` (partial) | Missing final event | DB reconciliation fallback | `R-20260521-001` usage: completed with DB-projected usage |
| — | DB snapshot validation | JSON validation in saveOpenCodeDB | `R-20260521-003` db-non-json: status=failed (was captured) |
| — | DB query timeout | 5s per-file context deadline | `R-20260521-003` db-timeout mechanism in place |
| — | Session ID alignment | Consistent ses_fake123 across fixtures | `R-20260521-003` all DB scenarios coherent |
| `I-0014` (partial) | `step_finish` was too coarse | terminal finish-reason gating, idle terminal evidence metadata, trailing usage test | `R-20260524-007` adapter unit suite |
| `I-0013` (partial) | Cleanup errors could replace primary outcome | cancellation/completion preserved; cleanup errors stored as metadata/warnings | `R-20260524-008` adapter unit suite |
| `I-0013` smoke | Cancellation final outcome | real OpenCode cancel returned `cancelled`/`cancellation`; process-group fixture killed child process | `R-20260524-009` targeted lifecycle smokes |
| `I-0014` real shapes | Native lifecycle payloads captured | CLI JSON omits final; DB has assistant `finish: stop`; HTTP/SSE has `session.status` idle and nested `part.reason: stop` | `R-20260524-010` raw OpenCode capture |

## Active Contracts To Decide

| ID | Question | Options | Needed Evidence |
|----|----------|---------|-----------------|
| `D-0001` | Final event vs non-zero exit | completed vs failed | Unit test: process exits 0 vs non-zero with identical event stream |
| `D-0002` | Timeout + DB terminal finish | timeout with evidence vs completed | Focused timeout test with DB session that has terminal finish |
| `D-0003` | Output alone enough for completion | completed-with-warning vs runtime_exit | Real smoke where output exists but DB reconciliation is unavailable |
| `D-0004` | Which fallback signals sufficient | accepted: final event > output > DB > none; timeout remains timeout | Adapter/test alignment |
| `D-0005` | Model not found → model_unavailable | Map text vs keep runtime_exit | OpenCode error message analysis |
| `D-0006` | Provider pre-validation | configuration vs runtime_exit | Provider format validation coverage |
| `D-0008` | Extract DB reconciler | Interface param vs keep subprocess call | Feasibility of extracting query interface |

## Next Recommended Work

1. **Transport spike**: Continue D-0009/R-20260524-006 HTTP/SSE or SDK spike for live `session.status` polling and abort semantics.
2. **Reporting cleanup**: Continue I-0003/I-0012 archive work.
