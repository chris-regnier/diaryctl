# Time Tracking Blocks

**Status:** Proposed

## Overview

Special block type for logging time spent on activities. Provides start/stop commands and reporting, stored as diary entries with structured attributes.

## Proposed Interface

```bash
# Start a timer
diaryctl time start "API design review"

# Stop the current timer
diaryctl time stop

# Log a completed block
diaryctl time log "Code review" --duration 45m

# Report for today
diaryctl time report

# Report for a date range
diaryctl time report --from 2026-02-01 --to 2026-02-07
```

## Core Concepts

- Time entries stored as blocks with `type: time` attribute
- Active timer stored in cache file (similar to shell prompt cache)
- Reports aggregate by activity name or date
- Integrates with the block-based data model when implemented
- Output formats: table, JSON, markdown

## Related Features

- [Block-Based Model](block-based-model.md) — Time entries as typed blocks
- [Weekly/Monthly Digest](weekly-digest.md) — Include time summaries in digests

---

*Proposed February 2026.*
