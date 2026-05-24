# R-20260524-004: Production DB Query Wiring

Date: 2026-05-24
Type: unit | smoke
Related issues: `I-0010`
Related decisions: `D-0008`, `D-0004`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run db-only-proof
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run db-non-json
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run db-timeout
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run db-locked
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- Model: fake OpenCode fixture for DB smokes
- Commit: not recorded

## Result

- Adapter tests: pass
- Smoke repo tests: pass
- `db-only-proof`: completed
- `db-non-json`: completed
- `db-timeout`: completed
- `db-locked`: completed

## Evidence

Production adapter now wires default DB query through `Runtime.queryOpenCodeDB`, calling:

```text
opencode db --format json <query>
```

for session, message, and part queries, then feeding the combined JSON into the unit-tested reconciler.

## Notes

`db-non-json`, `db-timeout`, and `db-locked` complete because the current accepted D-0004 order allows clean assistant-output fallback before DB proof. `db-only-proof` verifies the DB-only path can complete when no stdout final/output proof exists.
