#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 6: Observability And Evidence Assertions from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read /home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go and /home/antonioborgerees/coding/agentwrap/opencode/runtime.go for evidence capture patterns.

Your goal: smoke runs should fail when critical evidence is silently missing.

Your deliverables:
1. Add a verify-evidence harness command in /home/antonioborgerees/coding/agentwrap-smoke/cmd/agentwrap-run/main.go that checks:
   - events.jsonl exists for real runs that emit events
   - opencode-db-snapshot-status.json is required for every OpenCode smoke scenario
   - If result has session ID, DB snapshot should be captured or explicitly failed
   - If result has no session ID, DB snapshot should be skipped with no session id reason
   - results.json includes status, category when failed, native metadata, cleanup metadata, and phase metadata
2. Run it against the latest smoke-all directory (look in .agentwrap-logs/ for most recent).
3. Make smoke-all optionally call verify-evidence at the end.
4. Add report output listing missing or partial evidence.
5. Build: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
6. Run verify-evidence against the existing smoke runs and capture output.

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws6-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws6-stdout.log 2>&1
echo "WS6 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws6-stdout.log