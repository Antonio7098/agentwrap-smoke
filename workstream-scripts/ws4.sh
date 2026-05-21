#!/bin/bash
cd /home/antonioborgerees/coding/agentwrap-smoke

PROMPT='You are assigned Workstream 4: Session Lifecycle Coverage from the plan at AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md.

Read AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md and README.md first. Then read the agentwrap source: /home/antonioborgerees/coding/agentwrap/opencode/runtime.go, /home/antonioborgerees/coding/agentwrap/session.go (or wherever session action types live), and /home/antonioborgerees/coding/ultraplan/studies/go-cli-study/sources/opencode/ for reference.

Your goal: session metadata should match what OpenCode actually persisted, especially around continue, fresh, and unsupported fork behavior.

Your deliverables:
1. Find SessionAction types (Fresh, Continue, etc.) in the agentwrap source.
2. Add unit tests covering: (a) Fresh session success -> result has session ID, DB session row exists; (b) Continue existing session success -> requested ID equals result session row ID or metadata explains replacement; (c) Continue missing/invalid session -> explicit category, no silent fresh session; (d) Continue after failed run -> contract decision needed; (e) Fork unsupported -> expect configuration category; (f) Repair with SessionActionFresh -> metadata records fresh relationship; (g) Repair with SessionActionContinue -> continuity or documented fresh preference.
3. Capture evidence: RunResult.SessionID, RunMetadata.Session, DB session row, DB message rows, repair attempt session metadata, policy attempt session metadata.
4. Add smoke commands if real OpenCode runs are needed for session continue/repair scenarios.
5. Build: GOCACHE=/tmp/agentwrap-smoke-build go build -buildvcs=false -o agentwrap-run ./cmd/agentwrap-run

Write results summary to /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws4-results.md'

pi -p "$PROMPT" > /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws4-stdout.log 2>&1
echo "WS4 exit: $?" >> /home/antonioborgerees/coding/agentwrap-smoke/workstream-results/ws4-stdout.log