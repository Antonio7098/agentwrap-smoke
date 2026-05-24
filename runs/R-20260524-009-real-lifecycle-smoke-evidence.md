# R-20260524-009: Real/Fake Lifecycle Smoke Evidence

Date: 2026-05-24
Type: smoke
Related issues: `I-0013`, `I-0014`
Related decisions: `D-0001`, `D-0004`, `D-0009`

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run cancel
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run process-group-cancel-children
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run process-group-final-delayed
cd /home/antonioborgerees/coding/agentwrap-smoke && go run ./cmd/agentwrap-run process-group-malformed-after
```

## Result

All targeted lifecycle smokes passed.

## Evidence

### Real OpenCode cancellation

Log directory: `.agentwrap-logs/cancel-20260524-141435`

```text
Cancel succeeded without error
Wait error: opencode run: cancellation: OpenCode run was cancelled
Final Status: cancelled
Expectation: status=cancelled category=cancellation passed=true
```

Result summary:

- status: `cancelled`
- primary category: `cancellation`
- cleanup: attempted/completed, no cleanup error
- native event categories: lifecycle only (`event_count=2`)

### Fake process-group cancellation

Log directory: `.agentwrap-logs/process-group-cancel-children-20260524-141459`

```text
Status: cancelled
Category: cancellation
Fake OpenCode PID: 323070 alive=false
Helper PID: 323073 alive=false
Expectation: status=cancelled category=cancellation passed=true
```

Result summary:

- status: `cancelled`
- primary category: `cancellation`
- cleanup: attempted/completed, no cleanup error
- process group cleanup evidence: fake OpenCode and helper child both not alive

### Final event before delayed process termination

Log directory: `.agentwrap-logs/process-group-final-delayed-20260524-141440`

```text
Status: completed
No error returned - run completed successfully
Expectation: status=completed category= passed=true
```

Result summary:

- status: `completed`
- native terminal evidence: `step_finish`
- native event types include `step_finish`, `step_start`, `text`

### Malformed output after final event

Log directory: `.agentwrap-logs/process-group-malformed-after-20260524-141441`

```text
Status: completed
No error returned - run completed successfully
Expectation: status=completed category= passed=true
```

Result summary:

- status: `completed`
- native terminal evidence: `step_finish`
- post-final malformed output preserved as warning and `NativeMetadata.post_final_decode_warning`

## Notes

This run confirms the `I-0013` cancellation contract with a real OpenCode cancellation and process-group fixture. It also confirms `I-0014` final-event precedence/post-final warning behavior with fake OpenCode process-group fixtures. It does not yet capture real OpenCode `step_finish` finish reasons or `session.status idle` payload shapes.
