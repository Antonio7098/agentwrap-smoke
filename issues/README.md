# Issues

Issues are the unified tracking unit for this repo. Use them for findings, bugs, ambiguity, fixes, regression coverage, monitoring items, and unresolved behavior contracts.

Every issue should answer:
- What was observed?
- What evidence proves it?
- What behavior contract applies?
- What changed, if anything?
- What run verified the current status?

## Index

| ID | Status | Severity | Area | Summary |
|----|--------|----------|------|---------|
| `I-0001` | open | high | process-boundary | Non-zero process exit may override final structured event completion. |
| `I-0002` | monitoring | medium | rate-limit | Real provider rate limits are non-deterministic, making fallback evidence inconclusive. |
| `I-0003` | open | medium | reporting | Historical investigation state is spread across large top-level reports. |
| `I-0004` | open | medium | timeout | Timeout can race durable DB terminal completion. |
| `I-0005` | open | medium | process-boundary | Assistant output without final event or DB proof needs an explicit contract. |
| `I-0006` | open | critical | process-boundary | OpenCode JSON mode stdout lacks final structured event — root cause of most session gaps. |
| `I-0007` | open | medium | rate-limit | Timeout log classification modtime cutoff too tight for tests and quick runs. |
| `I-0008` | open | medium | rate-limit | MiniMax nested JSON "usage limit exceeded" not parsed by classifyRateLimitText. |
| `I-0009` | open | high | smoke-harness | Error category contract gaps: invalid model/provider not mapped to specific categories. |
| `I-0010` | open | high | db-reconciliation | DB reconciliation not unit-testable with fake runner — tightly coupled to subprocess. |
| `I-0011` | open | medium | fallback | No deterministic rate-limit fallback fixture — relies on live provider state. |
| `I-0012` | open | low | reporting | Archive historical AGENTWRAP_*.md source documents. |
| `I-0013` | open | medium | cancellation | Cancellation should wait for or preserve OpenCode's final cancellation outcome where possible. |
| `I-0014` | open | high | end-of-run | End-of-run detection should account for idle events, finish reasons, and trailing events. |

Total: 14 issues
