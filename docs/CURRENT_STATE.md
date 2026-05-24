# Current State

Last updated: 2026-05-24

## Latest Verified Baseline

- **Date**: 2026-05-24
- **Smoke run**: `R-20260524-013` (`process-group-nonzero-final`)
- **Result**: I-0001 verified — `sawFinal` correctly wins over non-zero runtime_exit; `D-0001` confirmed

## Open Issues

| ID       | Status     | Severity | Area             | Summary                                                                          | Next Action                                                     |
| -------- | ---------- | -------- | ---------------- | -------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `I-0001` | verified   | high     | process-boundary | Non-zero exit vs final event — already correct, verified                         | Close                                                           |
| `I-0004` | open       | medium   | timeout          | Timeout can race durable DB terminal finish                                      | Decide D-0002, verify adapter metadata                          |
| `I-0005` | open       | medium   | process-boundary | Clean exit + output but no final event or DB proof                               | Decide D-0003, align adapter behavior                           |
| `I-0006` | open       | critical | process-boundary | OpenCode JSON mode lacks final structured event (root cause of all session gaps) | Decide D-0004, ensure smoke tests use output/DB fallback        |
| `I-0007` | resolved   | low      | rate-limit       | Recursive JSON/string rate-limit scanning implemented                           | Close after I-0016 fix                                          |
| `I-0008` | resolved   | low      | rate-limit       | MiniMax rate_limit_error pattern matching implemented                           | Close after I-0016 fix                                          |
| `I-0009` | resolved   | medium   | validation       | Invalid model/provider → explicit categories (model_unavailable, configuration)  | Close                                                           |
| `I-0010` | resolved   | low      | db-reconciliation| DB path timeout, JSON validation, session ID alignment                           | Close                                                           |
| `I-0011` | open       | medium   | fallback         | Deterministic rate-limit fallback fixture missing                                | Add RUN_MODE=rate_limit to fake-opencode, add PolicyRunner test |
| `I-0012` | verified   | low      | reporting        | Archive AGENTWRAP_*.md source documents (docs/archive/ created)                  | Complete                                                        |
| `I-0013` | resolved   | medium   | lifecycle        | Cancellation/completion preserved over cleanup errors                           | Close                                                           |
| `I-0014` | resolved   | medium   | lifecycle        | step_finish terminal gating, idle evidence, native metadata                       | Close                                                           |
| `I-0015` | in_progress | critical | transport        | HTTP/SSE via SDK spike complete — `session.idle` confirmed, abort works, 19 typed event variants | Verify delta delivery, test session.error, SSE filtering check |
| `I-0016` | open       | critical | rate-limit       | Rate-limit classifier ignores nested `error.type: rate_limit_error`              | Spike fix in `rate_limit.go`, add unit test                     |
| `I-0003` | verified   | medium   | reporting        | Tracking infrastructure sufficient; historical docs archived                     | Complete                                                        |

## Recently Verified

| ID                   | Finding                                      | Fix                                                                                                                      | Verified By                                                                 |
| -------------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| `I-0001`             | Non-zero exit bug                            | sawFinal takes precedence over proc.ExitCode                                                                             | `R-20260524-013` process-group-nonzero-final: completed despite exit code 7 |
| `I-0005` (partial)   | Output without final event                   | sawOutput → completed with warning on clean exit                                                                         | `R-20260521-001` smoke-text: completed with warning                         |
| `I-0006` (partial)   | Missing final event                          | DB reconciliation fallback                                                                                               | `R-20260521-001` usage: completed with DB-projected usage                   |
| —                    | DB snapshot validation                       | JSON validation in saveOpenCodeDB                                                                                        | `R-20260521-003` db-non-json: status=failed (was captured)                  |
| —                    | DB query timeout                             | 5s per-file context deadline                                                                                             | `R-20260521-003` db-timeout mechanism in place                              |
| —                    | Session ID alignment                         | Consistent ses_fake123 across fixtures                                                                                   | `R-20260521-003` all DB scenarios coherent                                  |
| `I-0014` (partial)   | `step_finish` was too coarse                 | terminal finish-reason gating, idle terminal evidence metadata, trailing usage test                                      | `R-20260524-007` adapter unit suite                                         |
| `I-0015` (partial)   | HTTP/SSE transport viable                   | `session.idle` is definitive terminal; SDK v0.19.2 works; 19 typed events; abort via API | `R-20260524-018` HTTP/SSE spike: 10,178 events, session.idle confirmed, abort works |
| `I-0013` (partial)   | Cleanup errors could replace primary outcome | cancellation/completion preserved; cleanup errors stored as metadata/warnings                                            | `R-20260524-008` adapter unit suite                                         |
| `I-0013` smoke       | Cancellation final outcome                   | real OpenCode cancel returned `cancelled`/`cancellation`; process-group fixture killed child process                     | `R-20260524-009` targeted lifecycle smokes                                  |
| `I-0014` real shapes | Native lifecycle payloads captured           | CLI JSON omits final; DB has assistant `finish: stop`; HTTP/SSE has `session.status` idle and nested `part.reason: stop` | `R-20260524-010` raw OpenCode capture                                       |

## Active Contracts To Decide

| ID       | Question                            | Options                                                             | Needed Evidence                                                                 |
| -------- | ----------------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `D-0001` | Final event vs non-zero exit        | completed vs failed                                                 | ✅ VERIFIED: completed when final event present, non-zero preserved as metadata |
| `D-0002` | Timeout + DB terminal finish        | timeout with evidence vs completed                                  | Focused timeout test with DB session that has terminal finish                   |
| `D-0003` | Output alone enough for completion  | completed-with-warning vs runtime_exit                              | Real smoke where output exists but DB reconciliation is unavailable             |
| `D-0004` | Which fallback signals sufficient   | accepted: final event > output > DB > none; timeout remains timeout | Adapter/test alignment                                                          |
| `D-0005` | Model not found → model_unavailable | Map text vs keep runtime_exit                                       | OpenCode error message analysis                                                 |
| `D-0006` | Provider pre-validation             | configuration vs runtime_exit                                       | Provider format validation coverage                                             |
| `D-0008` | Extract DB reconciler               | Interface param vs keep subprocess call                             | Feasibility of extracting query interface                                       |

## Next Recommended Work

1. **Transport spike**: Continue D-0009/R-20260524-006 HTTP/SSE or SDK spike for live `session.status` polling and abort semantics.
2. **Rate-limit nested error**: Fix `I-0016` — classify nested `error.type: rate_limit_error` in `rate_limit.go`.
3. **Clean up resolved issues**: Close I-0007, I-0008, I-0009, I-0010, I-0013, I-0014 (verified/resolved per tracking assessment).
