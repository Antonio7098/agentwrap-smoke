#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 1: Process-Boundary Completion Semantics from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read the agentwrap source at /home/antonioborgerees/coding/agentwrap/opencode/runtime.go, /home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go, and /home/antonioborgerees/coding/agentwrap/opencode/projector.go. Also read /home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode/ for reference.

Your goal: define and test exactly when an OpenCode subprocess result should be considered completed, failed, cancelled, timed out, rate-limited, or locally unavailable.

Your deliverables:
1. Add unit tests in /home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go for every process-boundary case that can be simulated with the fake process runner. Cover all 8 cases listed in workstream 1: clean exit with final event, clean exit with text but no event, clean exit with DB terminal finish, clean exit with nothing, non-zero exit with DB finish, decode error, timeout with provider log match, timeout with DB finish.
2. Add smoke commands in /home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go for real cases requiring OpenCode.
3. Save per-scenario evidence: results.json, events.jsonl, DB snapshots.
4. Update AGENTWRAP_REPORTING.md with each observed state and the chosen contract.
5. Build the harness with: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
6. Run your tests and capture results.

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws1-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws1-stdout.log 2>&1
echo "WS1 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws1-stdout.log