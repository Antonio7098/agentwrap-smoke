# Historical Test Plan - AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md

**Original file**: `AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md` at repo root
**Archived date**: 2026-05-24
**Archive location**: `docs/archive/AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md`

## Historical Context

This document originally served as the primary test planning mechanism for real OpenCode testing. It contained:

- Test plan documentation
- Test matrix definitions
- Harness requirements
- Scenario definitions

## Status: Superseded

This document has been superseded by the run tracking system:

- `runs/` directory - individual run records with test results
- `docs/CURRENT_STATE.md` - latest verified baseline

## Original Document Summary

The original AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md was approximately 879 lines and covered test planning for real OpenCode testing in 2026-05.

## Test Categories (Historical Reference)

1. **Process boundary tests** - Non-zero exit, final event detection
2. **Timeout tests** - Timeout classification, DB reconciliation
3. **Fallback tests** - Rate limit fallback, repair exhaustion
4. **Cancellation tests** - Process cancellation, cleanup handling

## References

- See `runs/R-*.md` for individual run records
- See `docs/CURRENT_STATE.md` "Latest Verified Baseline" for current test status
- See `cmd/agentwrap-run/main.go` for smoke harness implementation
