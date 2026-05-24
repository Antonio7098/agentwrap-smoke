# Runs

Run records capture verification attempts. They should be short, factual, and linked from issues.

Use one run record per meaningful command group. A full `smoke-all` run is one record. A targeted unit test plus one targeted smoke command can also be one record if they verify the same issue.

## Index

| ID | Date | Type | Summary | Issues |
|----|------|------|---------|--------|
| `R-20260520-001` | 2026-05-20 | smoke | Original smoke-all baseline: 17 pass, 3 ambiguous (pre-fixes) | I-0001, I-0005, I-0006, I-0009 |
| `R-20260521-001` | 2026-05-21 | smoke | Full smoke-all baseline: 20/20 passed after adapter fixes | I-0001, I-0005, I-0006, I-0009 |
| `R-20260521-002` | 2026-05-21 | smoke | MiniMax targeted rate-limit run: inconclusive (account reset) | I-0002, I-0008, I-0011 |
| `R-20260521-003` | 2026-05-21 | fixture | DB hardening second pass (ws3-db-v2): JSON validation, timeout, ID alignment | I-0010 |
| `R-20260521-004` | 2026-05-21 | fixture | Process-group smoke: non-zero final, rate-limit precedence, cancel-children | I-0001, I-0005 |
| `R-20260521-005` | 2026-05-21 | fixture | Verify-evidence against smoke-all: 13 errors, 6 warnings, 16 info | I-0003 |
| `R-20260524-001` | 2026-05-24 | unit,smoke | Rate-limit unit fixes and session verification | I-0007, I-0008, I-0013, I-0014 |
| `R-20260524-002` | 2026-05-24 | unit,smoke | Error category mappings and module path fixes | I-0009 |
| `R-20260524-003` | 2026-05-24 | unit | DB reconciliation unit testability investigation | I-0010 |
| `R-20260524-004` | 2026-05-24 | unit | Production DB query wiring | I-0010 |
| `R-20260524-005` | 2026-05-24 | smoke,unit | I-0009 final smoke and issue close prep | I-0009, I-0007, I-0008, I-0010 |
| `R-20260524-006` | 2026-05-24 | research,manual | OpenCode ACP source and SDK spike | I-0015 |
| `R-20260524-007` | 2026-05-24 | unit | I-0014 JSON final signal hardening | I-0014 |
| `R-20260524-008` | 2026-05-24 | unit | I-0013 cancellation cleanup preservation | I-0013 |
| `R-20260524-009` | 2026-05-24 | smoke | Real lifecycle smoke evidence collection | I-0013 |
| `R-20260524-010` | 2026-05-24 | smoke | Real OpenCode shape capture | I-0014 |
| `R-20260524-011` | 2026-05-24 | smoke | Smoke evidence collection and test validation | I-0009, I-0010 |
| `R-20260524-012` | 2026-05-24 | smoke | Final configuration and test validation | I-0016, I-0002, I-0011 |
| `R-20260524-013` | 2026-05-24 | smoke | I-0001 non-zero exit final event spike | I-0001 |
| `R-20260524-014` | 2026-05-24 | smoke | I-0004 timeout DB terminal finish spike | I-0004 |
| `R-20260524-015` | 2026-05-24 | smoke | I-0005 output without final event spike | I-0005 |
| `R-20260524-016` | 2026-05-24 | smoke | I-0011 rate-limit fixture spike | I-0011 |
| `R-20260524-017` | 2026-05-24 | smoke | I-0012 archive historical reports | I-0012 |
| `R-20260524-020` | 2026-05-24 | assessment | I-0003 reporting sprawl assessment | I-0003 |

All historical raw artifacts live in `.agentwrap-logs/`. Run records reference specific directories by their timestamp.

Raw run directories referenced:

- `.agentwrap-logs/smoke-all-20260521-120218` — authoritative 20/20 baseline
- `.agentwrap-logs/smoke-all-20260520-212857` — pre-fix 17/20 baseline
- `.agentwrap-logs/ratelimit-20260521-120501` — MiniMax targeted (inconclusive)
- `.agentwrap-logs/ws3-db/` — DB hardening first pass
- `.agentwrap-logs/ws3-db-v2/` — DB hardening second pass with fixes
