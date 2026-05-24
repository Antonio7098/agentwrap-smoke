# R-20260524-003: DB Reconciliation Unit Testability

Date: 2026-05-24
Type: unit
Related issues: `I-0010`, `I-0005`
Related decisions: `D-0003`, `D-0004`, `D-0008`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: n/a
- Commit: not recorded

## Result

- Adapter tests: pass
- Smoke repo tests: pass

## Evidence

Added unit coverage in `/home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go` for:

- clean assistant output without final event completes with warning
- DB terminal assistant finish completes via injected fake DB query
- direct DB response helper handles message/part terminal finish shapes
- non-JSON / no-finish DB responses do not falsely complete
- locked DB / `wal_checkpoint` error classifies as `runtime_unavailable`

## Notes

This pass makes DB reconciliation unit-testable with fake runner state through:

- `reconcileDBResponse(body, err)` helper
- private `withDBQuery` test hook
- run-level `reconcileFinalState()` using the injected query hook

Remaining work: wire the default production OpenCode DB query path so real adapter runs use the tested helper without an injected test hook.
