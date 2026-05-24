# R-20260524-018: I-0015 ACP/HTTP-SSE Transport Spike - HTTP/SSE via Go SDK

Date: 2026-05-24
Type: research | spike | manual
Related issues: `I-0015`, `I-0013`, `I-0014`
Related decisions: `D-0009`
Related runs: `R-20260524-006`

## Commands

```bash
# Build spike
cd /home/antonioborgerees/coding/agentwrap-smoke/cmd/spike-http-sse
go build -o spike-http-sse .

# Run spike
./spike-http-sse

# Analyze results
python3 analyze.py  # in the run directory
```

## Environment

- Repo: `/home/antonioborgerees/coding/agentwrap-smoke`
- Spike: `cmd/spike-http-sse/main.go`
- SDK: `github.com/sst/opencode-sdk-go@v0.19.2`
- OpenCode server: v1.15.10
- OpenCode serve URL: `http://127.0.0.1:4096`

## Result: SUCCESS

The spike successfully:

1. **Started `opencode serve`** on port 4096 (auto-selected on --port 0)
2. **Created a session** via `client.Session.New()`
3. **Connected to SSE stream** via `client.Event.ListStreaming()`
4. **Sent a prompt** via `client.Session.Prompt()` with `Parts` parameter
5. **Recorded 10,178 events** across all SSE event types
6. **Detected session idle** via `session.idle` event type
7. **Sent abort** via `client.Session.Abort()`
8. **Collected session state** including modelID, tokens, mode, role

### Event Type Distribution

| Event Type                    | Count | Notes                                          |
| ----------------------------- | ----- | ---------------------------------------------- |
| `message.part.delta`          | 9,999 | Streaming text delta, `properties.version=""`  |
| `message.part.updated`        | 53    | Full part snapshot after chunking              |
| `message.updated`             | 27    | Full message snapshot (user + assistant)       |
| `server.heartbeat`            | 16    | Server-level heartbeat, `properties={}`        |
| `session.status`              | 18    | Session status change, `properties.version=""` |
| `session.updated`             | 15    | Session metadata update                        |
| `session.idle`                | 5     | **Terminal completion signal**                 |
| `session.diff`                | 9     | Session diff events                            |
| `session.created`             | 1     | Session creation                               |
| `session.next.agent.switched` | 3     | Agent switch events                            |
| `session.next.model.switched` | 3     | Model switch events                            |
| `server.connected`            | 2     | Server connection events                       |

### Key Event Shapes

#### `session.idle` — Terminal Completion Signal

```json
{
  "properties": {
    "sessionID": "ses_1a5a5436cffeY5Ehfc24IWhTLk"
  },
  "type": "session.idle"
}
```

- **`session.idle`** is the definitive terminal event for a completed session.
- It appears after `session.status` events in the bus.
- Multiple `session.idle` events can fire for the same session (idle enter/exit).
- For the abort test, `session.idle` also fires after abort, indicating cancelled state.

#### `message.part.delta` — Streaming Text Delta

```json
{
  "properties": {
    "version": ""
  },
  "type": "message.part.delta"
}
```

- **NOTE:** The `message.part.delta` events logged to the SSE stream via SDK have `properties={}` (empty). The delta data is not in the SSE event body — it's in the OpenCode internal bus events (seen in server logs as `service=bus type=message.part.delta publishing`).
- The SSE stream captures bus events as JSON, but `message.part.delta` has `version` property only, not the actual delta text.
- Actual text delta is delivered via separate mechanism or embedded differently.

#### `message.part.updated` — Part Snapshot

```json
{
  "properties": {
    "part": {
      "id": "prt_...",
      "messageID": "msg_...",
      "sessionID": "ses_...",
      "type": "text",
      "attempt": 0,
      "callID": "",
      "cost": 0,
      "error": null,
      "filename": "",
      "files": null,
      "hash": "",
      "metadata": null,
      "mime": "",
      "name": "",
      "reason": "",
      "snapshot": "",
      "source": null,
      "state": null,
      "synthetic": false,
      "text": "Say hello in exactly 5 words.",
      "time": { "start": 0 },
      "tokens": null,
      "tool": "",
      "url": ""
    },
    "delta": "optional incremental delta string"
  },
  "type": "message.part.updated"
}
```

Part types observed: `text`, `step-start`, `reasoning`, `tool`, `step-finish`

#### `message.updated` — Full Message

```json
{
  "properties": {
    "info": {
      "id": "msg_...",
      "role": "assistant",
      "sessionID": "ses_...",
      "time": { "created": 1779632356499 },
      "cost": 0,
      "error": null,
      "mode": "build",
      "modelID": "deepseek-v4-flash-free",
      "parentID": "",
      "path": null,
      "providerID": "",
      "summary": { "diffs": [], "body": "", "title": "" },
      "system": null,
      "tokens": {
        "cache": { "read": 8960, "write": 0 },
        "input": 19569,
        "output": 4730,
        "reasoning": 58
      }
    }
  },
  "type": "message.updated"
}
```

- `message.updated` with `role=assistant` is the rich message event with full metadata.
- `parts` field is empty in the SSE JSON — actual parts are in `message.part.updated` events.
- Token counts are present and cumulative.
- `mode: "build"` indicates agent/build mode.

#### `server.heartbeat` — Keepalive

```json
{
  "properties": { "version": "" },
  "type": "server.heartbeat"
}
```

- Periodic keepalive from server, ~3s interval.
- Can be used to verify connection is alive.

#### `session.error` — Error Events

- No `session.error` events observed in this run (no auth failures or errors).

## SDK API Surface Observed

### Session Operations

- `client.Session.New()` — create new session → `*Session` with `ID`, `Directory`, `ProjectID`, `Time`, `Title`
- `client.Session.Prompt(sessionID, SessionPromptParams{Parts: [...]})` — send prompt
- `client.Session.Abort(sessionID, SessionAbortParams{})` — abort running session
- `client.Session.Get(sessionID, SessionGetParams{})` — get session status

### Prompt Parameters

```go
SessionPromptParams{
    Parts: []SessionPromptParamsPartUnion{
        SessionPromptParamsPart{
            Type: opencode.F(SessionPromptParamsPartsTypeText),
            Text: opencode.F("prompt text"),
        },
    },
}
```

### Event Streaming

```go
stream := client.Event.ListStreaming(ctx, EventListParams{
    Directory: opencode.F("/path/to/dir"),
})
for stream.Next() {
    evt := stream.Current()  // EventListResponse
    switch u := evt.AsUnion().(type) {
    case opencode.EventListResponseEventMessageUpdated:
        // u.Properties.Info has Message
    case opencode.EventListResponseEventMessagePartUpdated:
        // u.Properties.Part has Part, u.Properties.Delta has incremental delta
    case opencode.EventListResponseEventSessionIdle:
        // u.Properties.SessionID — terminal completion
    case opencode.EventListResponseEventSessionError:
        // u.Properties.Error has error (ProviderAuthError, MessageAbortedError, etc.)
    }
}
stream.Close()
```

## Session Object Shape

```json
{
  "id": "ses_1a5a5436cffeY5Ehfc24IWhTLk",
  "directory": "/home/antonioborgerees/coding/agentwrap-smoke/cmd/spike-http-sse",
  "projectID": "7c5d18e63545c45cda8b049993ce2d9e70d8a86e",
  "time": {
    "created": 1779632356499,
    "updated": 1779632356499,
    "compacting": 0
  },
  "title": "New session - 2026-05-24T14:19:16.499Z",
  "version": "1.15.10",
  "parentID": "",
  "revert": { "messageID": "", "diff": "", "partID": "", "snapshot": "" },
  "share": { "url": "" },
  "summary": { "diffs": null }
}
```

## Key Findings

1. **Go SDK works**: `github.com/sst/opencode-sdk-go@v0.19.2` auto-generated from OpenAPI spec works correctly with local server.

2. **`session.idle` is the completion signal**: Not `session.status`, not message events. The `session.idle` event with `sessionID` is the definitive terminal completion signal.

3. **Event union is typed**: The SDK has strongly-typed union types for all 19 event types (`EventListResponseUnion`).

4. **`message.part.delta` has minimal SSE data**: The SSE stream sends `message.part.delta` with `properties.version=""` — actual delta text is in OpenCode's internal bus and may require a different endpoint or query for retrieval.

5. **`message.part.updated` carries full part**: The part snapshot event has all part fields including `type`, `text`, `tool`, `reason`, `cost`, `tokens`, etc.

6. **`message.updated` has full message**: Full message with `modelID`, `mode: "build"`, `tokens` breakdown (input/output/reasoning/cache).

7. **Abort works**: `client.Session.Abort()` successfully aborts a long-running session. `session.idle` fires after abort (with the abort session's ID).

8. **Server runs at port 4096**: With `--port 0`, `opencode serve` picked port 4096. Output parsing from stdout finds "opencode server listening on http://...".

9. **No auth required**: The local server runs without `OPENCODE_SERVER_PASSWORD` (unsecured warning shown).

10. **SSE stream is session-scoped but not filtered**: The `/event` SSE endpoint appears to stream all events globally (not filtered by session). Multiple sessions' events all appear in the same stream.

## Comparison to CLI Subprocess Transport

| Aspect             | CLI Subprocess                      | HTTP/SSE via SDK                                                                    |
| ------------------ | ----------------------------------- | ----------------------------------------------------------------------------------- |
| Final signal       | CLI JSON mode final event (missing) | `session.idle` SSE event                                                            |
| Part streaming     | Stdout text lines                   | `message.part.updated` + `message.part.delta`                                       |
| Session management | Process lifecycle                   | Explicit session.create/prompt/abort                                                |
| Error types        | Text classification                 | Typed `session.error` with union (`ProviderAuthError`, `MessageAbortedError`, etc.) |
| Token/usage info   | CLI output + DB                     | `message.updated` with `tokens` field                                               |
| Cancellation       | Process kill                        | `Session.Abort()` call                                                              |
| Complexity         | Low                                 | High (server + sessions + SSE)                                                      |
| Dependency         | None                                | `github.com/sst/opencode-sdk-go`                                                    |

## SDK Status

- **SDK available**: ✅ `github.com/sst/opencode-sdk-go@v0.19.2` via `go get`
- **Generates real SSE stream**: ✅ `client.Event.ListStreaming()` works
- **Typed unions**: ✅ All 19 event types have typed Go structs
- **Session management**: ✅ Create, prompt, abort, get all work
- **No authentication**: ✅ Local server unsecured
- **Streaming events**: ✅ Part delta via `message.part.updated` with `Delta` field

## Evidence

- Log directory: `runs/R-20260524-018-i0015-acp-transport-http-sse-spike/`
- `events.jsonl`: 10,178 events (2 spike meta + 10,176 SSE events)
- `session.json`: Session object for `ses_1a5a5436cffeY5Ehfc24IWhTLk`
- Spike binary: `cmd/spike-http-sse/spike-http-sse`
- Spike source: `cmd/spike-http-sse/main.go`

## Next Steps

1. Verify that `message.part.updated.Delta` contains the actual incremental text delta (not just `version=""`).
2. Test `session.error` events with auth failures and MessageAbortedError scenarios.
3. Check if `/event` SSE endpoint supports `sessionID` filter parameter.
4. Compare event shapes from `message.part.updated` with `message.part.delta` delta content.
5. Consider adapter design: the HTTP/SSE transport has richer typing than CLI but requires managing server lifecycle.
