#!/bin/bash
# Enhanced fake-opencode for smoke testing Workstream 7: Process-Group Cleanup and Final-State Precedence.
#
# This script simulates various process-boundary edge cases including:
# - Non-zero exit with final event (should complete)
# - Non-zero exit with rate-limit stderr (should be rate_limit)
# - Final event after delayed exit (event should win)
# - Process group termination (no surviving children)
# - Malformed output before and after final event
# - Rate-limit error shapes for fallback fixture testing (I-0011)
#
# Controlled by environment variables:
#
# RUN_MODE:
#   "final"             - emits step_start, text, step_finish (default)
#   "partial"           - emits step_start and text only
#   "empty"             - emits no stdout
#   "malformed"         - emits a valid event followed by malformed JSON
#   "nonzero_final"     - emits final events and exits 7 (should be completed)
#   "nonzero_partial"   - emits partial events and exits 7 (should fail)
#   "nonzero_ratelimit" - emits final events with rate-limit stderr, exits 7 (should be rate_limit)
#   "rate_limit"        - emits HTTP 429 rate-limit shape via stderr, exits 1 (triggers fallback)
#   "rate_limit_nested" - emits nested error.type: rate_limit_error shape (I-0016 test)
#   "timeout"           - sleeps until killed
#   "final_delayed"     - emits step_start, text, sleeps, then step_finish
#   "malformed_before_final" - malformed output before final event
#   "malformed_after_final"  - final event followed by malformed output
#
# DB_MODE:
#   "normal"              - normal JSON output
#   "unavailable"         - exit with error
#   "nonjson"             - output plain text not JSON
#   "timeout"             - hang for a long time
#   "locked"              - output SQLite locked error
#   "checkpoint"          - output WAL checkpoint failure
#   "session_no_assistant"- session row without assistant terminal proof
#   "assistant_no_finish" - assistant message without finish
#   "complete"            - session/message/part queries return terminal proof
#
# HELP_CHILD or HELPER_CHILD: if set, spawns a helper child that exits when parent receives SIGTERM
#   Used to test process-group cleanup (no surviving children after cancel)
# OPENCODE_PID_DIR: optional directory where opencode.pid and helper.pid are written

emit_delay() {
	local delay_ms="${RUN_DELAY_MS:-50}"
	sleep "$(awk "BEGIN { printf \"%.3f\", $delay_ms / 1000 }")"
}

spawn_helper() {
	# Spawns a background helper that exits on SIGTERM
	# This is used to test that Cancel() terminates the whole process group
	(
		trap 'echo "HELPER: received SIGTERM, exiting" >&2; exit 0' TERM
		echo "HELPER: started, waiting for SIGTERM..." >&2
		while true; do
			sleep 0.1
		done
	) &
	HELPER_PID=$!
	if [ -n "$OPENCODE_PID_DIR" ]; then
		mkdir -p "$OPENCODE_PID_DIR"
		echo "$HELPER_PID" >"$OPENCODE_PID_DIR/helper.pid"
	fi
	echo "HELPER: spawned child PID=$HELPER_PID" >&2
}

case "$1" in
db)
	case "$DB_MODE" in
	unavailable)
		echo '{"error":"database command unavailable"}' >&2
		exit 1
		;;
	nonjson)
		echo 'This is not JSON output from database'
		exit 0
		;;
	timeout)
		sleep 30
		echo '[]'
		;;
	locked)
		echo 'database is locked' >&2
		exit 1
		;;
	checkpoint)
		echo 'Failed to run the query '\''PRAGMA wal_checkpoint(PASSIVE)'\''}' >&2
		exit 1
		;;
	session_no_assistant)
		cat <<'EOF'
[{"id":"ses_test123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
		;;
	assistant_no_finish)
		cat <<'EOF'
[{"id":"ses_test123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
		echo '[{"id":"msg_1","session_id":"ses_test123","role":"assistant","time_created":"2026-05-21T10:00:30Z","data":"{\"role\":\"assistant\",\"content\":\"thinking\"}"}]'
		;;
	complete)
		query="$*"
		if [[ "$query" == *"from message"* ]]; then
			cat <<'EOF'
[{"id":"msg_1","session_id":"ses_test456","role":"assistant","time_created":"2026-05-21T10:00:30Z","data":"{\"role\":\"assistant\",\"finish\":\"stop\",\"content\":\"completed\"}"}]
EOF
		elif [[ "$query" == *"from part"* ]]; then
			cat <<'EOF'
[{"id":"part_1","session_id":"ses_test456","time_created":"2026-05-21T10:00:30Z","data":"{\"type\":\"step-finish\",\"finish\":\"stop\"}"}]
EOF
		else
			cat <<'EOF'
[{"id":"ses_test456","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
		fi
		;;
	*)
		if [ "$2" = "--format" ] && [ "$3" = "json" ]; then
			shift 3
			echo '[{"id":"ses_default","tokens_input":100,"tokens_output":50}]'
		else
			echo '{"error":"unknown db subcommand"}' >&2
			exit 1
		fi
		;;
	esac
	;;
run)
	if [ -n "$OPENCODE_PID_DIR" ]; then
		mkdir -p "$OPENCODE_PID_DIR"
		echo "$$" >"$OPENCODE_PID_DIR/opencode.pid"
	fi

	# Emit stderr if configured (for rate-limit testing)
	if [ -n "$FAKE_STDERR" ]; then
		echo "$FAKE_STDERR" >&2
	fi

	# Spawn helper child if requested (for process-group cleanup testing)
	if [ -n "$HELP_CHILD" ] || [ -n "$HELPER_CHILD" ]; then
		spawn_helper
	fi

	if [[ "$*" == *"--format json"* ]]; then
		case "${RUN_MODE:-final}" in
		final)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			;;

		partial)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			;;

		empty)
			;;

		malformed)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			echo '{"type":'
			;;

		nonzero_final)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			exit 7
			;;

		nonzero_partial)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			exit 7
			;;

		nonzero_ratelimit)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			# Emit rate-limit stderr then exit non-zero
			echo '{"type":"rate_limit_error","statusCode":429,"message":"too many requests","responseHeaders":{"retry-after-ms":"1500"}}' >&2
			exit 7
			;;

		rate_limit)
			# Emit HTTP 429 rate-limit shape - triggers PolicyRunner fallback
			emit_delay
			echo '{"type":"error","statusCode":429,"message":"rate limit exceeded","error":{"type":"rate_limit_error","message":"OpenCode provider rate limit reached"}}' >&2
			exit 1
			;;

		rate_limit_nested)
			# Emit nested error.type: rate_limit_error shape (tests I-0016 classifier)
			# Note: message intentionally does NOT contain "rate limit" to test structural detection
			emit_delay
			echo '{"type":"error","message":"Model not found: opencode/gpt-5.5. Did you mean: gpt-5.5, gpt-5.5-pro?","error":{"type":"rate_limit_error","message":"usage limit exceeded","metadata":{"retry-after-ms":2000}}}' >&2
			exit 1
			;;

		timeout)
			sleep 30
			;;

		final_delayed)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			# Sleep for a bit (simulating delayed finish)
			sleep 1
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			;;

		malformed_before_final)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			# Malformed output before final events
			echo '{"type":'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			;;

		malformed_after_final)
			emit_delay
			echo '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
			emit_delay
			echo '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
			emit_delay
			echo '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
			emit_delay
			# Malformed output AFTER final event
			echo '{"type":'
			;;

		*)
			echo "Unknown RUN_MODE: $RUN_MODE" >&2
			exit 2
			;;
		esac
	fi
	;;
version)
	echo '{"version":"fake-opencode-1.0"}'
	;;
*)
	echo "Unknown command: $1" >&2
	exit 1
	;;
esac
