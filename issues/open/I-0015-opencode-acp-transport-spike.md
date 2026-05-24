# I-0015: Evaluate OpenCode ACP / HTTP-SSE Transport

Status: in_progress
Severity: critical
Area: transport
Discovered: 2026-05-24
Related decisions: `D-0009`
Related runs: `R-20260524-006`, `R-20260524-018`

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

Local source path inspected (R-20260524-006):

- `/home/antonioborgerees/coding/ultraplan/studies/opencode-wrap-study/sources/opencode`

Key findings:

- `packages/opencode/src/cli/cmd/acp.ts` starts an HTTP server and then bridges ACP over JSON-RPC/NDJSON stdio.
- `packages/opencode/src/acp/README.md` says ACP currently lacks streaming responses and tool call reporting.
- `packages/opencode/src/server/routes/instance/httpapi/groups/session.ts` defines REST session endpoints including create, prompt, async prompt, status, and abort.
- `packages/opencode/src/server/routes/instance/httpapi/groups/event.ts` and `handlers/event.ts` define `/event` SSE.
- `packages/opencode/src/cli/cmd/run/stream.transport.ts` shows OpenCode's own CLI run transport prefers `session.status` idle events and also polls live session status to avoid stale/missed idle events.

## SDK Status (R-20260524-018)

✅ **Go SDK available**: `github.com/sst/opencode-sdk-go@v0.19.2` via `go get`
✅ **Auto-generated from OpenAPI**: Stainless-generated, fully typed
✅ **Server works**: `opencode serve --port 0` starts on auto-assigned port (e.g. 4096)
✅ **Session management works**: Create, prompt, abort, get all functional
✅ **SSE streaming works**: `client.Event.ListStreaming()` returns typed stream
✅ **No auth required**: Local server unsecured (warning shown)

SDK key types:

- `*Client` with `Session`, `Event`, `Agent`, `File`, `Project` services
- `EventService.ListStreaming()` → `*ssestream.Stream[EventListResponse]`
- `EventListResponse` with `AsUnion()` for 19 typed event variants
- `SessionService.New()`, `Prompt()`, `Abort()`, `Get()`

## Event Shapes (R-20260524-018)

### `session.idle` — Terminal Completion Signal

```json
{
  "properties": { "sessionID": "ses_..." },
  "type": "session.idle"
}
```

**`session.idle` is the definitive completion event.** Multiple fires per session (idle enter/exit). Fires also after abort.

### `message.updated` — Full Message Snapshot

```json
{
  "properties": {
    "info": {
      "id": "msg_...",
      "role": "assistant",
      "sessionID": "ses_...",
      "modelID": "deepseek-v4-flash-free",
      "mode": "build",
      "tokens": {
        "cache": { "read": 8960, "write": 0 },
        "input": 19569,
        "output": 4730,
        "reasoning": 58
      },
      "cost": 0,
      "error": null
    }
  },
  "type": "message.updated"
}
```

### `message.part.updated` — Part Snapshot with Delta

```json
{
  "properties": {
    "part": {
      "id": "prt_...",
      "messageID": "msg_...",
      "type": "text",
      "text": "Say hello in exactly 5 words.",
      "tool": "",
      "reason": "",
      "cost": 0,
      "tokens": null
    },
    "delta": "optional incremental delta"
  },
  "type": "message.part.updated"
}
```

Part types observed: `text`, `step-start`, `reasoning`, `tool`, `step-finish`

### `message.part.delta` — Streaming Delta (Note: Minimal SSE Data)

```json
{
  "properties": { "version": "" },
  "type": "message.part.delta"
}
```

⚠️ **Warning**: `message.part.delta` events in the SSE stream have `properties.version=""` — actual delta text is not in the SSE event body. The server logs show `service=bus type=message.part.delta publishing` with full delta data, but the SSE stream only receives `version` property. The actual incremental text comes via `message.part.updated` with `properties.delta` field.

### `session.error` — Error Events

Typed union with variants:

- `ProviderAuthError` — provider auth failure
- `MessageAbortedError` — message was aborted
- `APIError` — provider API error with `statusCode`, `isRetryable`, `message`
- `UnknownError` — unknown error
- `MessageOutputLengthError` — output length exceeded

```json
{
  "properties": {
    "error": {
      "data": { "message": "...", "providerID": "..." },
      "name": "ProviderAuthError"
    },
    "sessionID": "ses_..."
  },
  "type": "session.error"
}
```

### Other Events

- `server.heartbeat` — ~3s keepalive, `properties={}`
- `server.connected` — server connection, `properties={}`
- `session.created` — session creation with full `Session` object
- `session.updated` — session metadata update
- `session.status` — session status change (properties.version="" in SSE)
- `session.diff` — session diff events
- `session.next.model.switched` / `session.next.agent.switched` — switches

## Expected Contract (Partially Verified)

| Requirement                                | Status      | Notes                                             |
| ------------------------------------------ | ----------- | ------------------------------------------------- |
| One-shot `StartRun` as session/prompt/idle | ✅ Possible | Session.New → Prompt → SSE idle                   |
| Completion via `session.idle`              | ✅ Verified | Fires 5 times across 2 sessions                   |
| Cancellation via `/session/abort`          | ✅ Verified | Abort sent, idle fires after                      |
| Typed error mapping                        | ✅ Possible | 5 typed error variants in SDK                     |
| Rich event streaming                       | ⚠️ Partial  | Text delta in `message.part.updated`, not `delta` |

## Comparison: CLI vs HTTP/SSE

| Aspect             | CLI Subprocess                         | HTTP/SSE via SDK                                               |
| ------------------ | -------------------------------------- | -------------------------------------------------------------- |
| Final signal       | CLI JSON mode final event (unreliable) | `session.idle` SSE event                                       |
| Part streaming     | Stdout text lines                      | `message.part.updated` + `message.part.delta`                  |
| Session management | Process lifecycle                      | Explicit session.create/prompt/abort                           |
| Error types        | Text classification                    | Typed union (`ProviderAuthError`, `MessageAbortedError`, etc.) |
| Token/usage info   | CLI output + DB                        | `message.updated.tokens` field                                 |
| Cancellation       | Process kill                           | `Session.Abort()` call                                         |
| Complexity         | Low                                    | High (server + sessions + SSE + event filtering)               |
| Dependency         | None                                   | `github.com/sst/opencode-sdk-go@v0.19.2`                       |
| Auth required      | No                                     | No (local server unsecured)                                    |
| Event typing       | None                                   | 19 typed union variants                                        |

## Implementation

No adapter implementation yet. This is a spike/tracking issue.

The spike binary is at `cmd/spike-http-sse/main.go` (not in production adapter path).

## Verification

| Run              | What                    | Result                                                      |
| ---------------- | ----------------------- | ----------------------------------------------------------- |
| `R-20260524-006` | Source + SDK discovery  | SDK exists, ACP limited, HTTP API rich                      |
| `R-20260524-018` | HTTP/SSE spike with SDK | 10,178 events captured, session.idle confirmed, abort works |

## Next Action

Remaining verification items:

1. **Verify delta delivery**: Confirm `message.part.updated.properties.delta` contains actual incremental text (vs `version=""`). The SSE stream's `message.part.delta` type had empty properties, but `message.part.updated` had `delta` field. Need to confirm delta extraction path.

2. **Test `session.error`**: Trigger auth failure and MessageAbortedError to see typed error shapes.

3. **SSE filtering**: Check if `/event` endpoint supports `sessionID` query parameter to filter events per-session. The current run received events from multiple sessions in the same stream.

4. **Adapter design**: Evaluate whether managing `opencode serve` lifecycle plus session state is worth the richer typing and completion signal. The CLI subprocess remains simpler for one-shot use cases.

5. **Delta accumulation**: Test if running prompts with tools (read/write) produce `tool` type parts with `tool` name, `CallID`, `State` fields.

## Findings Summary

- ✅ Go SDK v0.19.2 works end-to-end with local server
- ✅ `session.idle` is the reliable completion signal (not `session.status`)
- ✅ Abort via `Session.Abort()` works and produces `session.idle`
- ✅ 19 typed event variants via union types
- ✅ Token/usage info in `message.updated.tokens`
- ⚠️ `message.part.delta` has no SSE delta data (only `version`)
- ⚠️ SSE stream not filtered by session (global event bus)
- ⚠️ Actual text streaming via `message.part.updated` with `delta` field
- ✅ No auth required for local server
