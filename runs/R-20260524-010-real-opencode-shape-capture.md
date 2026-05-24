# R-20260524-010: Real OpenCode Lifecycle Shape Capture

Date: 2026-05-24
Type: raw-capture | smoke | source-backed
Related issues: `I-0014`, `I-0015`
Related decisions: `D-0004`, `D-0009`

## Commands

Direct CLI JSON-mode captures:

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke
/home/antonioborgerees/.opencode/bin/opencode run --format json --model opencode/deepseek-v4-flash-free --dir /tmp 'Reply with exactly: OK' > .agentwrap-logs/raw-opencode-shapes-20260524-1420/simple-ok.raw.jsonl 2> .agentwrap-logs/raw-opencode-shapes-20260524-1420/simple-ok.stderr.txt
/home/antonioborgerees/.opencode/bin/opencode run --format json --model opencode/deepseek-v4-flash-free --dir /tmp 'Use no tools. Reply with exactly: DONE' > .agentwrap-logs/raw-opencode-shapes-20260524-1420/no-tools-done.raw.jsonl 2> .agentwrap-logs/raw-opencode-shapes-20260524-1420/no-tools-done.stderr.txt
/home/antonioborgerees/.opencode/bin/opencode run --format json --model opencode/deepseek-v4-flash-free --dir /tmp 'Reply with exactly: SHAPE' > .agentwrap-logs/raw-opencode-shapes-20260524-1420/shape.raw.jsonl 2> .agentwrap-logs/raw-opencode-shapes-20260524-1420/shape.stderr.txt
```

DB evidence for those sessions:

```bash
/home/antonioborgerees/.opencode/bin/opencode db --format json "select * from part where session_id='<session>' order by time_created"
/home/antonioborgerees/.opencode/bin/opencode db --format json "select * from message where session_id='<session>' order by time_created"
```

HTTP/SSE capture:

```bash
/home/antonioborgerees/.opencode/bin/opencode serve --hostname 127.0.0.1 --port 41037 --print-logs
curl -sS -X POST -H 'content-type: application/json' -d '{}' http://127.0.0.1:41037/session
curl -sS -N http://127.0.0.1:41037/event > .agentwrap-logs/raw-opencode-shapes-20260524-1420/http/event-stream.sse
curl -sS -X POST -H 'content-type: application/json' -d '{"providerID":"opencode","modelID":"deepseek-v4-flash-free","parts":[{"type":"text","text":"Reply with exactly: HTTPDONE"}]}' http://127.0.0.1:41037/session/<session>/prompt_async
curl -sS http://127.0.0.1:41037/session/status
```

Validation after extractor update:

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./opencode
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
```

## Evidence Location

Raw artifacts:

- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/*.raw.jsonl`
- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/*.parts.json`
- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/*.messages-all.json`
- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/http/event-stream.sse`
- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/http/status-*.json`
- `.agentwrap-logs/raw-opencode-shapes-20260524-1420/shape-summary.json`

## Findings

### Direct `opencode run --format json` stdout

Across three real runs, stdout event types were limited to `step_start` and/or `text`:

| File                      | Session                          | stdout event types   | stdout `step_finish` | stdout `session.status` |
| ------------------------- | -------------------------------- | -------------------- | -------------------- | ----------------------- |
| `simple-ok.raw.jsonl`     | `ses_1a5d9c378ffe8im7wZtgByoi72` | `step_start`         | no                   | no                      |
| `no-tools-done.raw.jsonl` | `ses_1a5d9abffffe8l1Zh43FRLeysb` | `step_start`, `text` | no                   | no                      |
| `shape.raw.jsonl`         | `ses_1a5d92aa1ffedwtw80ly0Tsq4j` | `step_start`         | no                   | no                      |

This re-confirms `I-0006`: current CLI JSON stdout can omit final structured events even when the run completed.

### Direct-run DB terminal evidence

The same sessions had durable DB terminal evidence. Example part shape from `*.parts.json`:

```json
{
  "reason": "stop",
  "type": "step-finish",
  "tokens": {
    "total": 8214,
    "input": 8199,
    "output": 3,
    "reasoning": 12,
    "cache": { "write": 0, "read": 0 }
  },
  "cost": 0
}
```

Example assistant message shape from `*.messages-all.json`:

```json
{
  "role": "assistant",
  "finish": "stop",
  "tokens": {
    "total": 8214,
    "input": 8199,
    "output": 3,
    "reasoning": 12,
    "cache": { "write": 0, "read": 0 }
  },
  "cost": 0,
  "modelID": "deepseek-v4-flash-free",
  "providerID": "opencode",
  "time": { "created": 1779628955558, "completed": 1779628957447 }
}
```

This backs the existing DB reconciliation contract: assistant `finish: "stop"` is durable terminal proof.

### HTTP/SSE event shapes

The real HTTP/SSE stream emitted `session.status` with `busy` transitions and final `idle`:

```json
{
  "id": "evt_e5a289538001ERv3nzQKKatk4d",
  "type": "session.status",
  "properties": {
    "sessionID": "ses_1a5d7b59effenzw1301sORj0kY",
    "status": { "type": "idle" }
  }
}
```

The same stream emitted step finish as a `message.part.updated` event with nested part shape:

```json
{
  "type": "message.part.updated",
  "properties": {
    "sessionID": "ses_1a5d7b59effenzw1301sORj0kY",
    "part": {
      "id": "prt_e5a28943c0011HT91yfUH1rqfv",
      "reason": "stop",
      "snapshot": "5f659eaaf871bd61f07f8fbe9ccfd6d350aeff14",
      "messageID": "msg_e5a2889da001Zwy3NBM2Tti2xW",
      "sessionID": "ses_1a5d7b59effenzw1301sORj0kY",
      "type": "step-finish",
      "tokens": {
        "total": 8636,
        "input": 8619,
        "output": 4,
        "reasoning": 13,
        "cache": { "write": 0, "read": 0 }
      },
      "cost": 0
    }
  }
}
```

`/session/status` polling returned the session as `busy` while active, then `{}` after idle. This matches OpenCode source behavior where missing status item is treated as idle by `stream.transport.ts`.

## Implementation Follow-up

Based on the real nested shape, `opencode/projector.go` was updated to extract finish reasons from nested `part.reason`/`part.finish*` fields as well as top-level fields. Unit coverage was added for nested `part.reason: "stop"`.

## Verification

- `/home/antonioborgerees/coding/agentwrap`: `go test ./opencode` passed
- `/home/antonioborgerees/coding/agentwrap`: `go test ./...` passed
- `/home/antonioborgerees/coding/agentwrap-smoke`: `go test ./...` passed

## Conclusion

All `I-0014` lifecycle semantics are now backed by real OpenCode evidence:

- CLI JSON stdout may omit final events; output/DB fallback remains necessary.
- DB terminal proof uses assistant `finish: "stop"` and/or part `type: "step-finish", reason: "stop"`.
- HTTP/SSE terminal event shape is `session.status` with `properties.status.type: "idle"`.
- HTTP/SSE step-finish shape carries terminal reason as nested `properties.part.reason`.
