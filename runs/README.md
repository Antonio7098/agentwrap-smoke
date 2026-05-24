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
| `R-20260524-012` | 2026-05-24 | smoke | Real rate-limit from gpt-5.5: fallback triggered but classification was `runtime_exit` not `rate_limit` | I-0016, I-0002, I-0011 |

All historical raw artifacts live in `.agentwrap-logs/`. Run records reference specific directories by their timestamp.

Raw run directories referenced:

- `.agentwrap-logs/smoke-all-20260521-120218` — authoritative 20/20 baseline
- `.agentwrap-logs/smoke-all-20260520-212857` — pre-fix 17/20 baseline
- `.agentwrap-logs/ratelimit-20260521-120501` — MiniMax targeted (inconclusive)
- `.agentwrap-logs/ws3-db/` — DB hardening first pass
- `.agentwrap-logs/ws3-db-v2/` — DB hardening second pass with fixes
