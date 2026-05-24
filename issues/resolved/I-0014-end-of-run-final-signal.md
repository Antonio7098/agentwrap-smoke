# I-0014: End-Of-Run Detection Uses Too Coarse A Final Signal

Status: fixed pending close
Severity: high
Area: end-of-run
Discovered: 2026-05-22
Related decisions: `D-0001`, `D-0003`, `D-0004`
Related runs: `R-20260524-007`, `R-20260524-009`, `R-20260524-010`

## Observation

The OpenCode lifecycle report shows that OpenCode's run completion is more nuanced than a single `step_finish` event:

- prompt loop completion depends on assistant finish, pending tool calls, tool results, and assistant ordering
- session idle is the authoritative bus-level completion event
- `step_finish` can have finish reasons that may not mean final completion
- trailing usage, artifacts, warnings, or errors may arrive around the final boundary

The adapter currently treats projected final events, especially `step_finish`, as the main end-of-run signal.

## Expected Contract

End-of-run detection should distinguish:

- normal final completion
- finish that still has pending tool calls
- permission/error stop
- cancellation/abort
- idle session signal, if OpenCode emits it
- trailing structured evidence after final

The adapter should not mark a run completed solely because a `step_finish` was seen if the native payload says more work or an error state remains.

## Evidence

Source report:

- `/home/antonioborgerees/coding/ultraplan/targets/agentwrap/reports/opencode-cancellation-failures-end-of-run.md`

Relevant sections:

- `3.1 OpenCode's Completion Detection`
- `3.2 agentwrap's Current End-of-Run Detection`
- `3.3 Gaps and Recommendations`

## Root Cause

The adapter compresses OpenCode's richer lifecycle into `sawFinal`. That is pragmatic, but it risks false completion or dropped metadata when native events contain finish reasons, idle events, or trailing data.

## Implementation

In progress / initial JSON-mode hardening implemented in `/home/antonioborgerees/coding/agentwrap` on 2026-05-24:

- `opencode/projector.go`: `step_finish` inspects `finish_reason` / `finishReason` / `stopReason` / `stop_reason` / `reason` / `finish`.
- `opencode/projector.go`: terminal finish reasons (`stop`, `end_turn`, `stop_sequence`, plus legacy empty reason) set final; non-terminal reasons (`tool_calls`, `max_tokens`, `length`, `content_filter`) project as progress and do not set final.
- `opencode/projector.go` + `runtime.go`: `session.status` idle is recorded as native terminal evidence and can complete a clean JSON-mode run without using the assistant-output fallback warning.
- `opencode/runtime.go`: preserves `finish_reason` and `native_terminal_evidence` in `RunMetadata.NativeMetadata`.
- Assistant-output fallback and DB reconciliation fallback remain unchanged.
- Added unit coverage for non-terminal finish-like events, idle completion evidence, and trailing usage after `step_finish`.

Real OpenCode shape capture in `R-20260524-010`:

- direct `opencode run --format json` stdout emitted `step_start` and/or `text`, but no `step_finish` and no `session.status`, confirming output/DB fallback remains required for subprocess JSON mode.
- direct-run DB rows recorded terminal assistant `finish: "stop"` and part `type: "step-finish", reason: "stop"` with token/cost evidence.
- HTTP/SSE emitted `session.status` with `properties.status.type: "idle"`.
- HTTP/SSE emitted step finish as `message.part.updated` with nested `properties.part.type: "step-finish"` and `properties.part.reason: "stop"`.
- `opencode/projector.go` now extracts finish reasons from nested `part.reason` / `part.finish*` fields as well as top-level fields.

Deferred / remaining:

- current subprocess JSON transport cannot poll live session status the way OpenCode's HTTP/SDK `stream.transport.ts` does, so live status polling belongs to the transport spike/follow-up (`I-0015`/D-0009), not this JSON-mode contract fix.

## Verification

Verified 2026-05-24 in `R-20260524-007`:

- `/home/antonioborgerees/coding/agentwrap`: `go test ./opencode`
- `/home/antonioborgerees/coding/agentwrap`: `go test ./...`
- `/home/antonioborgerees/coding/agentwrap-smoke`: `go test ./...`

Smoke evidence in `R-20260524-009`:

- `go run ./cmd/agentwrap-run process-group-final-delayed`: completed; native terminal evidence recorded as `step_finish`.
- `go run ./cmd/agentwrap-run process-group-malformed-after`: completed; malformed output after final preserved as warning and `NativeMetadata.post_final_decode_warning`.

Real OpenCode shape evidence in `R-20260524-010`:

- Direct CLI JSON stdout: no `step_finish` or `session.status` across three real completed runs.
- OpenCode DB: terminal `finish: "stop"` on assistant message and `type: "step-finish", reason: "stop"` on part rows.
- HTTP/SSE: `session.status` idle shape is `{"type":"session.status","properties":{"sessionID":"...","status":{"type":"idle"}}}`.
- HTTP/SSE: step-finish shape is `message.part.updated` with nested `properties.part.reason: "stop"`.

## Remaining Risk

OpenCode event schema may vary by version. Tests should use captured native event fixtures where possible.

## Next Action

Review `R-20260524-007`, `R-20260524-009`, and `R-20260524-010`; move to resolved if accepted. Track live HTTP/SSE status polling and abort semantics under the transport spike (`I-0015`/D-0009), not as a blocker for this JSON-mode final-signal fix.
