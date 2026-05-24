# I-0015: Evaluate OpenCode ACP / HTTP-SSE Transport

Status: open
Severity: medium
Area: transport
Discovered: 2026-05-24
Related decisions: `D-0009`
Related runs: `R-20260524-006`

## Observation

The current OpenCode adapter is subprocess/stdout based. It has been hardened with final-event precedence, output fallback, DB reconciliation, recent-log classification, and error text mapping. However, this still leaves lifecycle complexity at the process boundary.

OpenCode source has richer programmatic surfaces:

- `opencode acp` command
- internal HTTP API server
- `/event` SSE stream
- `/session/status`
- `/session/:sessionID/abort`
- generated Go SDK `github.com/sst/opencode-sdk-go`

## Evidence

Local source path inspected:

- `/home/antonioborgerees/coding/ultraplan/studies/opencode-wrap-study/sources/opencode`

Key findings:

- `packages/opencode/src/cli/cmd/acp.ts` starts an HTTP server and then bridges ACP over JSON-RPC/NDJSON stdio.
- `packages/opencode/src/acp/README.md` says ACP currently lacks streaming responses and tool call reporting.
- `packages/opencode/src/server/routes/instance/httpapi/groups/session.ts` defines REST session endpoints including create, prompt, async prompt, status, and abort.
- `packages/opencode/src/server/routes/instance/httpapi/groups/event.ts` and `handlers/event.ts` define `/event` SSE.
- `packages/opencode/src/cli/cmd/run/stream.transport.ts` shows OpenCode's own CLI run transport prefers `session.status` idle events and also polls live session status to avoid stale/missed idle events.
- `go list -m -versions github.com/sst/opencode-sdk-go` confirms `v0.19.2` exists.
- `go doc github.com/sst/opencode-sdk-go` exposes a generated `Client`, typed message/error types, and event response types.

## Expected Contract

The current CLI subprocess runtime remains the default until a spike proves an alternative transport fits `agentwrap`'s one-shot `StartRun` contract.

A future ACP/HTTP transport must prove:

1. one-shot prompt execution can be represented as `StartRun` / `Events` / `Wait`
2. final completion can be detected by session-scoped `session.status` idle plus message state
3. cancellation can use `/session/:id/abort` and preserve a final cancellation outcome
4. typed provider/model/auth errors can map to `SDKError.Category`
5. event streaming is rich enough for message, tool, permission, usage, and artifact projection

## Implementation

No adapter implementation yet. This is a spike/tracking issue.

## Verification

Initial source/API discovery recorded in `R-20260524-006`.

## Next Action

Build a tiny standalone spike, outside the default adapter path, that either:

- uses the Go SDK against a locally started OpenCode HTTP/SSE server, or
- uses direct HTTP/SSE if the SDK lacks stream ergonomics.

The spike should create a session, send a prompt, listen to `/event`, wait for session idle, abort a long prompt, and record the exact event/error shapes.
