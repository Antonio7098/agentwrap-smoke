# R-20260524-002: Error Category Mappings And Module Path

Date: 2026-05-24
Type: unit | smoke
Related issues: `I-0009`
Related decisions: `D-0005`, `D-0006`, `D-0007`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run fallback-invalid-model
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run fallback-invalid-provider
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: smoke defaults
- Commit: not recorded

## Result

- Adapter tests: pass
- Smoke repo tests: pass after canonical module/import path update to `github.com/Antonio7098/agentwrap`
- `fallback-invalid-model`: completed after first attempt classified `model_unavailable`
- `fallback-invalid-provider`: completed after first attempt classified `model_unavailable`; this smoke scenario uses combined model string `not-a-provider/model`, not separate invalid `Provider`

## Evidence

`fallback-invalid-model` excerpt:

```text
Attempt 1 target=0 model=opencode/not-a-real-model status=failed error=model_unavailable
Attempt 2 target=1 model=opencode/deepseek-v4-flash-free status=completed error=
```

`fallback-invalid-provider` excerpt:

```text
Attempt 1 target=0 model=not-a-provider/model status=failed error=model_unavailable
Attempt 2 target=1 model=opencode/deepseek-v4-flash-free status=completed error=
```

## Notes

Implemented partial `I-0009` coverage:

- fatal OpenCode `Model not found` / `unknown model` -> `model_unavailable`
- fatal auth text -> `authentication`
- invalid wrapper provider/model syntax -> `configuration` before process start

Remaining `I-0009` work: smoke coverage for separate invalid `Provider`, DB-unavailable clean-exit classification, and explicit `fallback-all-fail` category contract.
