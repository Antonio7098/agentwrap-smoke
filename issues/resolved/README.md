# Resolved Issues

Issues that have been resolved, verified, or downgraded to monitoring.

| ID       | Status     | Severity | Area             | Summary                                                                                   |
| -------- | ---------- | -------- | ---------------- | ----------------------------------------------------------------------------------------- |
| `I-0002` | monitoring | medium   | rate-limit       | Real provider rate limits are non-deterministic, making fallback evidence inconclusive.   |
| `I-0006` | resolved   | critical | process-boundary | OpenCode JSON mode stdout lacks final structured event — root cause of most session gaps. |
| `I-0007` | resolved   | medium   | rate-limit       | Timeout log classification modtime cutoff too tight for tests and quick runs.             |
| `I-0008` | resolved   | medium   | rate-limit       | MiniMax nested JSON "usage limit exceeded" not parsed by classifyRateLimitText.           |
| `I-0009` | resolved   | high     | smoke-harness    | Error category contract gaps: invalid model/provider not mapped to specific categories.    |
| `I-0010` | resolved   | high     | db-reconciliation | DB reconciliation not unit-testable with fake runner.                                      |
| `I-0013` | resolved   | medium   | cancellation      | Cancellation primary outcome is preserved; cleanup errors remain metadata/warnings.        |

Total: 7 resolved issues
