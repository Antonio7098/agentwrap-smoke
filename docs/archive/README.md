# Archive: Historical Reports

This directory contains the original source documents that served as the primary reporting and planning mechanism before the issue/run tracking system was established.

## Archived Documents

### AGENTWRAP_REPORTING.md

- **Original size**: ~1459 lines
- **Content**: Comprehensive report tracking bug discoveries, fixes, and verification records for agentwrap adapter
- **Period**: Early development through 2026-05
- **Superseded by**: `issues/` directory and `runs/` directory

### AGENTWRAP_NEXT_ROBUSTNESS_PLAN.md

- **Original size**: ~963 lines
- **Content**: Future planning document with workstream evidence and execution order for robustness improvements
- **Period**: 2026-05
- **Superseded by**: `issues/` directory (open issues tracked individually)

### AGENTWRAP_REAL_OPENCODE_TEST_PLAN.md

- **Original size**: ~879 lines
- **Content**: Test plan, test matrix, and harness requirements for real OpenCode testing
- **Period**: 2026-05
- **Superseded by**: `runs/` directory and `docs/CURRENT_STATE.md`

## Why These Documents Were Archived

The three source files contained valuable historical evidence but were not suitable as primary operating documents going forward. Key issues:

1. **Append-only growth**: Adding new findings to large files made it difficult to track individual issues independently
2. **No stable IDs**: Finding a specific bug required searching through long documents
3. **No status tracking**: Cannot easily determine which items were verified, open, or blocked
4. **Redundant tracking**: Findings were also recorded in issue/run format

## New Tracking System

The current system uses:

- `issues/` - individual issue records with stable IDs (I-0001, I-0002, ...)
- `runs/` - run records with date-based IDs (R-YYYYMMDD-NNN)
- `docs/CURRENT_STATE.md` - compact dashboard of latest known state
- `docs/DECISIONS.md` - active contract decisions
- `docs/PROCESS.md` - tracking process documentation

## Archive Date

Archived: 2026-05-24
Archived by: Worker R-20260524-017 (I-0012)
