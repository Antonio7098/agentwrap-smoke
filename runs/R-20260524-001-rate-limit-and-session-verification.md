# R-20260524-001: Rate-Limit Unit Fixes And Session Verification

Date: 2026-05-24
Type: unit | smoke
Related issues: `I-0007`, `I-0008`, `I-0013`, `I-0014`
Related decisions: `D-0004`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./opencode
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && ./agentwrap-run session-fresh
cd /home/antonioborgerees/coding/agentwrap-smoke && ./agentwrap-run session-continue-existing
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: smoke defaults
- Commit: not recorded

## Result

- Adapter unit tests: pass
- Adapter full tests: pass
- `session-fresh`: completed
- `session-continue-existing`: completed

## Evidence

- Unit output: `ok github.com/Antonio7098/agentwrap/opencode`
- Full adapter output: all packages passed
- Session fresh: completed with fresh session relationship
- Session continue: completed with best_effort continue relationship

## Notes

Implemented fixes:

- `I-0008`: recursive nested JSON/string scanning for provider `responseBody` plus MiniMax `rate_limit_error` / `usage limit exceeded` patterns.
- `I-0007`: timeout path scans recent OpenCode logs with a 5-minute cutoff and model/provider content matching, then promotes matching provider logs to `rate_limit` with `rate_limit_info`.

Lifecycle audit note for `I-0013`/`I-0014`: sampled current smoke artifacts show DB `finish` evidence in `opencode-parts.json`, but no captured `step_finish` or `session.status` in `events.jsonl`; further work should use native OpenCode fixture captures or the lifecycle report.
