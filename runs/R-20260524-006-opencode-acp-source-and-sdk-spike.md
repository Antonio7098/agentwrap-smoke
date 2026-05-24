# R-20260524-006: OpenCode ACP Source And SDK Spike

Date: 2026-05-24
Type: research | manual
Related issues: `I-0015`, `I-0013`, `I-0014`
Related decisions: `D-0009`

## Commands

```bash
go list -m -versions github.com/sst/opencode-sdk-go
opencode acp --help
cd /home/antonioborgerees/coding/ultraplan/studies/opencode-wrap-study/sources/opencode \
  && grep -R "acp\|session.status\|MessageAbortedError\|ProviderAuthError" -n packages/opencode packages/sdk packages/docs
```

Read source files:

```text
packages/opencode/src/acp/README.md
packages/opencode/src/cli/cmd/acp.ts
packages/opencode/src/cli/cmd/run/stream.transport.ts
packages/opencode/src/server/routes/instance/httpapi/groups/session.ts
packages/opencode/src/server/routes/instance/httpapi/groups/event.ts
packages/opencode/src/server/routes/instance/httpapi/handlers/event.ts
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Adapter repo: `/home/antonioborgerees/coding/agentwrap`
- OpenCode source: `/home/antonioborgerees/coding/ultraplan/studies/opencode-wrap-study/sources/opencode`
- SDK checked: `github.com/sst/opencode-sdk-go` latest observed `v0.19.2`

## Result

- The Go SDK exists and publishes versions through `v0.19.2`.
- `opencode acp --help` exists locally.
- Local source shows `opencode acp` is JSON-RPC/NDJSON ACP over stdio, backed by an internally started HTTP server.
- Local ACP README says streaming responses and tool-call reporting are not yet implemented.
- The richer HTTP API has `/event` SSE, session status, prompt, async prompt, and abort endpoints.
- OpenCode's own CLI run transport already uses session-scoped `session.status` idle events plus live status polling to determine prompt-turn completion.

## Evidence

Important source facts:

- `packages/opencode/src/cli/cmd/acp.ts`:
  - starts `Server.listen(opts)`
  - creates `createOpencodeClient({ baseUrl, headers })`
  - bridges ACP via `AgentSideConnection` and `ndJsonStream(input, output)`

- `packages/opencode/src/acp/README.md`:
  - says ACP implementation follows Agent Client Protocol v1
  - documents `session/new`, `session/load`, `session/prompt`
  - current limitations include no streaming responses and no tool-call reporting

- `packages/opencode/src/server/routes/instance/httpapi/groups/session.ts`:
  - defines `/session`, `/session/status`, `/session/:sessionID/message`, `/session/:sessionID/prompt_async`, `/session/:sessionID/abort`

- `packages/opencode/src/server/routes/instance/httpapi/groups/event.ts` + `handlers/event.ts`:
  - define `/event` SSE endpoint
  - streams bus events as JSON SSE messages

- `packages/opencode/src/cli/cmd/run/stream.transport.ts`:
  - comments: prefer `session.status` idle events and poll session status because transports can miss status events
  - session turn completion is session-id scoped

## Notes

This changes the ACP plan: ACP itself may not yet be the best first target because local docs list streaming/tool reporting gaps. The likely useful future transport is OpenCode HTTP/SSE via the generated Go SDK or direct HTTP.

No production code was changed for this spike.
