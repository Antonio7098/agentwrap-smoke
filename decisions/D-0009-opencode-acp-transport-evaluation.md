# D-0009: Evaluate OpenCode ACP Transport

Status: proposed
Date: 2026-05-24
Related issues: `I-0013`, `I-0014`
Related runs: `R-20260524-006`

## Problem

The current OpenCode adapter uses `opencode run --format json` as a one-shot subprocess. Recent fixes improved stdout final-signal fallback, DB reconciliation, cancellation classification, and provider error categorization, but several risks remain process-boundary specific:

- final completion can require heuristics across stdout, process exit, DB state, and logs
- cancellation is SIGTERM/SIGKILL based instead of an OpenCode-native abort operation
- typed provider/model/auth errors are often reconstructed from text

OpenCode also exposes ACP/server APIs that may provide a better structured transport.

## Evidence

Local source inspection under `/home/antonioborgerees/coding/ultraplan/studies/opencode-wrap-study/sources/opencode` found two distinct API surfaces:

1. **ACP command** (`packages/opencode/src/cli/cmd/acp.ts`, `packages/opencode/src/acp/README.md`)
   - `opencode acp` starts an HTTP server internally, then exposes Agent Client Protocol over JSON-RPC/NDJSON stdio using `@agentclientprotocol/sdk`.
   - The local ACP README explicitly says streaming responses and tool-call reporting are not yet implemented in ACP.

2. **OpenCode HTTP/SSE API** (`packages/opencode/src/server/routes/instance/httpapi/...`)
   - REST endpoints include `/session`, `/session/:sessionID/message`, `/session/:sessionID/abort`, `/session/status`, and `/event` SSE.
   - `/event` streams bus events as SSE, including `session.status`, `session.error`, message and permission events.
   - The CLI run transport already prefers `session.status` idle events and also polls `session.status` to guard against missed idle events.

The Go SDK exists: `github.com/sst/opencode-sdk-go` latest observed version `v0.19.2`. `go doc` shows a generated `Client`, `EventListResponse`, `AssistantMessage`, `MessageAbortedError`, and `ProviderAuthError` types.

## Options Considered

1. **Keep CLI subprocess only**
   - Lowest risk.
   - Continues to require stdout/DB/log reconciliation and process-boundary cancellation.

2. **Add optional HTTP/SSE transport**
   - Spawn `opencode acp --port 0` or `opencode serve`-equivalent server mode, then talk to the HTTP API via SDK or direct HTTP.
   - Use `/event` and `/session/status` for final-signal semantics.
   - Use `/session/:id/abort` for cancellation.
   - More stateful but closer to OpenCode internals.

3. **Use ACP JSON-RPC stdio directly**
   - Protocol-compliant for editor-style clients.
   - Local source notes current ACP limitations: no streaming responses and limited tool-call reporting, so it may not yet satisfy agentwrap event streaming requirements.

## Current Leaning

Do not replace the current CLI adapter immediately.

Create a separate spike for an optional OpenCode HTTP/SSE or ACP-backed runtime. Prefer investigating the HTTP/SSE API first because local OpenCode's own CLI run transport already uses `session.status` idle events and `/event` SSE for robust run completion.

## Consequences

- `I-0014` should use the local `stream.transport.ts` logic as evidence for the correct final-signal contract: prompt turn completion is session-id scoped, waits for idle, and re-checks live status to avoid stale idle events.
- `I-0013` should consider `/session/:id/abort` as the clean cancellation primitive for a future transport.
- ACP itself is not a guaranteed drop-in replacement yet because the local ACP README says streaming and tool call reporting are still incomplete.
