# R-20260524-007: I-0014 JSON Final Signal Hardening

Date: 2026-05-24
Type: unit
Related issues: `I-0014`
Related decisions: `D-0001`, `D-0003`, `D-0004`, `D-0009`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./opencode
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
```

## Result

- OpenCode adapter tests: pass
- Full adapter repo tests: pass
- Smoke repo compile/tests: pass

## Implementation Summary

- `step_finish` now sets final only when its finish reason is terminal.
- Non-terminal finish reasons such as `tool_calls`, `max_tokens`, `length`, and `content_filter` project as progress and do not set `sawFinal`.
- `session.status` with idle status is captured as native terminal evidence and can complete a clean run without using the assistant-output warning fallback.
- Finish reason and native terminal evidence are preserved in `RunMetadata.NativeMetadata`.
- Assistant-output fallback and DB reconciliation fallback behavior remain unchanged.

## Coverage Added

- Projection coverage for terminal and non-terminal `step_finish` finish reasons.
- Runtime coverage that `tool_calls` `step_finish` does not complete by itself.
- Runtime coverage that `session.status` idle completes without output fallback warning.
- Runtime coverage that usage after a terminal `step_finish` is still captured.

## Notes

This is a JSON-mode adaptation of the OpenCode source finding from `stream.transport.ts`: OpenCode's own run transport prefers session-scoped `session.status` idle evidence and re-checks live status to avoid stale idle. The subprocess/JSON adapter cannot poll live status through the current transport, so this change records idle evidence and keeps fallback ordering conservative.
