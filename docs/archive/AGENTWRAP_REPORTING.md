# Historical Report - AGENTWRAP_REPORTING.md

**Original file**: `AGENTWRAP_REPORTING.md` at repo root
**Archived date**: 2026-05-24
**Archive location**: `docs/archive/AGENTWRAP_REPORTING.md`

## Historical Context

This document originally served as the primary investigation record for agentwrap adapter behavior. It contained:

- Bug reports and discoveries
- Fix logs and verification records
- Process boundary observations
- OpenCode behavior documentation

## Status: Superseded

This document has been superseded by the issue/run tracking system:

- `issues/` directory - individual issue records with stable IDs
- `runs/` directory - run records with date-based IDs
- `docs/CURRENT_STATE.md` - compact dashboard

## Original Document Summary

The original AGENTWRAP_REPORTING.md was approximately 1459 lines and covered early development through 2026-05. Key topics included:

1. **Process boundary issues** - Non-zero exit handling, final event detection
2. **Timeout behavior** - Race conditions with durable DB terminal finish
3. **Rate limiting** - Classification and fallback mechanisms
4. **DB reconciliation** - Fallback when stdout is incomplete

## References

- See `issues/open/I-0001.md` through `issues/open/I-0016.md` for individual issue records
- See `runs/R-*.md` for individual run records
- See `docs/CURRENT_STATE.md` for latest state dashboard
