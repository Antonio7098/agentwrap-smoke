# Open Issues

Issues that are actively being tracked and need resolution.

| ID       | Severity | Area              | Summary                                                                                        |
| -------- | -------- | ----------------- | ---------------------------------------------------------------------------------------------- |
| `I-0001` | high     | process-boundary  | Non-zero process exit may override final structured event completion.                          |
| `I-0003` | medium   | reporting         | Historical investigation state is spread across large top-level reports.                       |
| `I-0004` | medium   | timeout           | Timeout can race durable DB terminal completion.                                               |
| `I-0005` | medium   | process-boundary  | Assistant output without final event or DB proof needs an explicit contract.                   |
| `I-0011` | medium   | fallback          | No deterministic rate-limit fallback fixture — relies on live provider state.                  |
| `I-0012` | low      | reporting         | Archive historical AGENTWRAP\_\*.md source documents.                                          |
| `I-0015` | medium   | transport         | Evaluate OpenCode ACP / HTTP-SSE transport as alternative to subprocess/stdout mode.           |
| `I-0016` | critical | rate-limit       | Rate-limit classifier ignores nested `error.type: rate_limit_error`.                          |

Total: 8 open issues
