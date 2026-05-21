#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 3: OpenCode DB Hardening from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read the agentwrap source at /home/antonioborgerees/coding/agentwrap/opencode/runtime.go and look for DB query/reconciliation code. Also read /home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode/ for reference on DB schema.

Your goal: DB reconciliation and evidence capture should never hide failures or block final result reporting.

Your deliverables:
1. Find and examine the DB query code in agentwrap (look for opencode db, queryOpenCodeDB, etc).
2. Add unit tests covering: (a) DB command unavailable -> result preserved, DB snapshot status records failure; (b) DB query returns non-JSON -> reconciliation declines, evidence records parse failure; (c) DB query times out -> reconciliation declines quickly, final result does not hang; (d) DB locked / checkpoint failure -> runtime_unavailable when primary, snapshot failure when evidence only; (e) DB has session row but no assistant message -> not enough proof; (f) DB has assistant message with no finish -> not enough proof; (g) DB has assistant message with terminal finish and usage -> completion reconcilable.
3. If the DB reconciliation logic is not easily unit-testable, extract a testable helper.
4. Add smoke evidence assertions that every scenario either captures DB rows or records DB snapshot status reason.
5. Build: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
6. Run tests.

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws3-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws3-stdout.log 2>&1
echo "WS3 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws3-stdout.log