#!/bin/bash
# Simulates a rate-limit 429 response from provider
# Usage: fake-opencode rate-limit-429
# Exits with code 1 and emits rate-limit stderr

cat << 'EOF'
{"type":"error","statusCode":429,"message":"too many requests","responseHeaders":{"retry-after-ms":"1500","x-ratelimit-limit-requests":"500","x-ratelimit-remaining-requests":"0","x-ratelimit-reset-requests":"1s"},"error":{"type":"rate_limit_error","message":"usage limit exceeded"}}
EOF
echo '{"type":"final_result","timestamp":'$(date +%s)000',"sessionID":"fake-ses","part":{"type":"final-result","sessionID":"fake-ses","finish":"stop","usage":{"input_tokens":10,"output_tokens":5}}}'
exit 1
