#!/bin/bash
# Fake opencode for smoke testing process-boundary and DB failure scenarios.
# It intentionally behaves like a real child process instead of an in-memory
# runner, so tests can exercise stdout framing, stderr, exit codes, delays, and
# db reconciliation through the same process boundary as real OpenCode.
#
# Controlled by environment variables:
#
# RUN_MODE:
#   "final"             - emits step_start, text, step_finish
#   "partial"           - emits step_start and text only
#   "empty"             - emits no stdout
#   "malformed"         - emits a valid event followed by malformed JSON
#   "nonzero_final"     - emits final events and exits 7
#   "nonzero_partial"   - emits partial events and exits 7
#   "timeout"           - sleeps until killed
#
# FAKE_STDERR: stderr text emitted during run
# RUN_DELAY_MS: optional delay between emitted events
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
#   "db_only_proof"       - session+message+part rows with terminal finish; no other proof
#
# RUN_MODE:
#   "final"             - emits step_start, text, step_finish
#   "partial"           - emits step_start and text only
#   "empty"             - emits no stdout
#   "malformed"         - emits a valid event followed by malformed JSON
#   "nonzero_final"     - emits final events and exits 7
#   "nonzero_partial"   - emits partial events and exits 7
#   "timeout"           - sleeps until killed
#   "db_proof"          - step_start+text only, no step_finish; wrapper must use DB to prove completion

emit_delay() {
    if [ -n "$RUN_DELAY_MS" ]; then
        sleep "$(awk "BEGIN { printf \"%.3f\", $RUN_DELAY_MS / 1000 }")"
    fi
}

emit_run_event() {
    echo "$1"
    emit_delay
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
                cat << 'EOF'
[{"id":"ses_fake123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
                ;;
            assistant_no_finish)
                cat << 'EOF'
[{"id":"ses_fake123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
                echo '[{"id":"msg_1","session_id":"ses_fake123","role":"assistant","time_created":"2026-05-21T10:00:30Z","data":"{\"role\":\"assistant\",\"content\":\"thinking\"}"}]'
                ;;
            complete)
                query="$*"
                if [[ "$query" == *"from message"* ]]; then
                    cat << 'EOF'
[{"id":"msg_1","session_id":"ses_fake123","role":"assistant","time_created":"2026-05-21T10:00:30Z","data":"{\"role\":\"assistant\",\"finish\":\"stop\",\"content\":\"completed\"}"}]
EOF
                elif [[ "$query" == *"from part"* ]]; then
                    cat << 'EOF'
[{"id":"part_1","session_id":"ses_fake123","time_created":"2026-05-21T10:00:30Z","data":"{\"type\":\"step-finish\",\"finish\":\"stop\"}"}]
EOF
                else
                    cat << 'EOF'
[{"id":"ses_fake123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
                fi
                ;;
            db_only_proof)
                query="$*"
                if [[ "$query" == *"from message"* ]]; then
                    cat << 'EOF'
[{"id":"msg_1","session_id":"ses_fake123","role":"assistant","time_created":"2026-05-21T10:00:30Z","data":"{\"role\":\"assistant\",\"finish\":\"stop\",\"content\":\"completed\"}"}]
EOF
                elif [[ "$query" == *"from part"* ]]; then
                    cat << 'EOF'
[{"id":"part_1","session_id":"ses_fake123","time_created":"2026-05-21T10:00:30Z","data":"{\"type\":\"step-finish\",\"finish\":\"stop\"}"}]
EOF
                else
                    cat << 'EOF'
[{"id":"ses_fake123","time_created":"2026-05-21T10:00:00Z","time_updated":"2026-05-21T10:01:00Z","model":"deepseek-v4-flash-free","tokens_input":100,"tokens_output":50,"tokens_reasoning":0}]
EOF
                fi
                ;;
            *)
                # Default normal behavior
                if [ "$2" = "--format" ] && [ "$3" = "json" ]; then
                    shift 3
                    # Simple query response
                    echo '[{"id":"ses_default","tokens_input":100,"tokens_output":50}]'
                else
                    echo '{"error":"unknown db subcommand"}' >&2
                    exit 1
                fi
                ;;
        esac
        ;;
    run)
        if [ -n "$FAKE_STDERR" ]; then
            echo "$FAKE_STDERR" >&2
        fi
        if [[ "$*" == *"--format json"* ]]; then
            case "${RUN_MODE:-final}" in
                final)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    emit_run_event '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
                    emit_run_event '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
                    ;;
                partial)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    emit_run_event '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
                    ;;
                empty)
                    ;;
                malformed)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    echo '{"type":'
                    ;;
                nonzero_final)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    emit_run_event '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
                    emit_run_event '{"type":"step_finish","timestamp":1710000002000,"sessionID":"ses_fake123","part":{"type":"step-finish"}}'
                    exit 7
                    ;;
                nonzero_partial)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    emit_run_event '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
                    exit 7
                    ;;
                timeout)
                    sleep 30
                    ;;
                db_proof)
                    emit_run_event '{"type":"step_start","timestamp":1710000000000,"sessionID":"ses_fake123"}'
                    emit_run_event '{"type":"text","timestamp":1710000001000,"sessionID":"ses_fake123","part":{"type":"text","text":"completed"}}'
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
