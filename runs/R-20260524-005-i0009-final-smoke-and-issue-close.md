# R-20260524-005: I-0009 Final Smoke And Issue Close Prep

Date: 2026-05-24
Type: smoke | unit
Related issues: `I-0009`, `I-0007`, `I-0008`, `I-0010`
Related decisions: `D-0005`, `D-0006`, `D-0007`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run invalid-separate-provider
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run fallback-all-fail
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: smoke defaults
- Commit: not recorded

## Result

- Smoke repo tests: pass
- `invalid-separate-provider`: failed/configuration before process start
- `fallback-all-fail`: failed/model_unavailable with both attempts classified model_unavailable

## Evidence

`invalid-separate-provider`:

```text
Provider: bad/provider Model: test-model
StartRun error: opencode model: configuration: provider must not contain '/'
Category: configuration
UserDetail: provider must not contain '/'
```

`fallback-all-fail`:

```text
Both primary and fallback use invalid models
Status: failed
Category: model_unavailable
Attempt 1 target=0 model=opencode/not-a-real-model status=failed error=model_unavailable
Attempt 2 target=1 model=opencode/also-not-a-model status=failed error=model_unavailable
```

## Notes

Added smoke harness coverage for the separate invalid Provider pre-validation path and made `fallback-all-fail` assert an explicit `model_unavailable` category contract.

Moved `I-0007`, `I-0008`, and `I-0010` from open to resolved after reviewing their implementation and run records.
