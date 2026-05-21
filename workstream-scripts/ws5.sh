#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 5: Error Category Contract Tightening from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read /home/antonioborgerees/coding/agentwrap/opencode/runtime.go, /home/antonioborgerees/coding/agentwrap/opencode/projector.go, and /home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode/ for reference.

Your goal: every stable expected-failure smoke scenario should assert both status and category. Current gaps: fallback-all-fail expects only failed, invalid model/provider failures often surface as runtime_exit.

Your deliverables:
1. Run existing smoke scenarios that cover invalid provider/model cases. Look at /home/antonioborgerees/coding/agentwrap-smoke/.agentwrap-logs/ for prior runs.
2. For each gap, gather real outputs: inspect OpenCode event JSON, stderr, DB rows, and logs.
3. Decide the public agentwrap category contract for: Invalid provider, Invalid model, OpenCode DB unavailable, Provider 429, Provider auth/account issue, Clean OpenCode fatal event with no provider detail.
4. Add unit tests for exact event/stderr shapes for each new category.
5. Tighten smoke expectations in the harness once contract is settled.
6. Build: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run
7. Run tests and document results.

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws5-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws5-stdout.log 2>&1
echo "WS5 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws5-stdout.log