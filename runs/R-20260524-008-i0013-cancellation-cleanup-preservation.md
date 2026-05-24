# R-20260524-008: I-0013 Cancellation Cleanup Preservation

Date: 2026-05-24
Type: unit
Related issues: `I-0013`
Related decisions: none

## Commands

```bash
cd /home/antonioborgerees/coding/agentwrap && go test ./opencode
cd /home/antonioborgerees/coding/agentwrap && go test ./...
cd /home/antonioborgerees/coding/agentwrap-smoke && go test ./...
```

## Result

- OpenCode adapter tests: pass
- Full adapter repo tests: pass
- Smoke repo compile/tests: pass

## Implementation Summary

- `Cancel()` now preserves explicit cancellation as the primary caller-visible outcome even if process cleanup reports an error.
- `Cancel()` waits briefly for the run goroutine to finalize cancellation evidence, bounded by 100ms or caller context cancellation.
- Cleanup errors no longer overwrite completed/cancelled primary outcomes in `finalResult()`.
- Cleanup failures remain recorded in `RunMetadata.Cleanup`, `RunMetadata.Errors`, warnings, and `NativeMetadata.cleanup_warning`.
- Cleanup failure no longer emits a lifecycle transition to failed after the run outcome has already been determined.

## Coverage Added

- Normal completed run with cleanup failure remains `completed` and records cleanup failure as metadata/warning.
- Explicit cancelled run with cleanup failure remains `cancelled` / `cancellation`; cleanup failure is preserved as metadata.
