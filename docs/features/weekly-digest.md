# Weekly/Monthly Digest

**Status:** Proposed

## Overview

Summarize diary entries over a configurable time period into a condensed digest. Useful for weekly reviews, monthly retrospectives, or sharing progress updates.

## Proposed Interface

```bash
# Generate a weekly digest
diaryctl digest --period week

# Monthly digest
diaryctl digest --period month --format markdown

# Custom date range
diaryctl digest --from 2026-02-01 --to 2026-02-07

# Output as JSON for scripting
diaryctl digest --period week --format json
```

## Core Concepts

- Aggregates entries by date range into a structured summary
- Groups by template name or context if available
- Outputs as markdown, plain text, or JSON
- Could integrate with MCP for AI-powered summarization

## Related Features

- [Full-Text Search](search.md) — Search within digest results
- [MCP Entry Creation](mcp-entry-creation.md) — AI could generate digest entries

---

*Proposed February 2026.*
