#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 2: Deterministic Rate-Limit And Fallback Testing from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read the agentwrap source at /home/antonioborgerees/coding/agentwrap/opencode/runtime.go, /home/antonioborgerees/coding/agentwrap/opencode/rate_limit.go, and /home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go. Also read /home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode/ for reference.

Your goal: prove fallback behavior without depending on a provider account being rate-limited at the exact moment of the test.

Your deliverables:
1. Create a harness-only fake opencode executable (a shell script placed earlier on PATH) that deterministically outputs rate-limit stderr shapes for deterministic process-boundary tests. Place it in /home/antonioborgerees/coding/agentwrap-smoke/fake-opencode/ and document how to use it.
2. Add unit tests in /home/antonioborgerees/coding/agentwrap/opencode/runtime_test.go for: (a) stderr contains provider 429 / MiniMax usage limit -> expected rate_limit; (b) context timeout plus matching OpenCode log with rate limit -> expected rate_limit not generic timeout; (c) primary returns rate_limit, fallback completes -> policy unit test.
3. Add smoke commands for fallback path exercising deterministically (using the fake opencode).
4. Update AGENTWRAP_REPORTING.md documenting the approach and results.
5. Build: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws2-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws2-stdout.log 2>&1
echo "WS2 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws2-stdout.log