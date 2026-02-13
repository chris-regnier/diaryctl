# Mood Sentiment

**Status:** Proposed

## Overview

Auto-detect mood/sentiment from entry content and store as a block attribute. Enables mood-over-time queries and visualization.

## Proposed Interface

```bash
# View mood trend
diaryctl mood

# Mood for a date range
diaryctl mood --from 2026-01-01 --to 2026-02-01

# Query entries by mood
diaryctl list --mood positive
```

## Core Concepts

- Lightweight local sentiment analysis (no external API required)
- Mood stored as block attribute: `mood: positive|neutral|negative` with optional numeric score
- Manual mood override via template prompts or `--mood` flag
- TUI visualization: mood indicators on date list, optional trend sparkline
- MCP tool: `get_mood_trend` for AI assistant queries

## Implementation Approaches

1. **Keyword-based** — Simple word list scoring (fast, no dependencies)
2. **Go NLP library** — Use `github.com/jdkato/prose` or similar for basic sentiment
3. **MCP delegation** — Let the AI assistant analyze mood and set it via `create_entry` attributes

## Related Features

- [Block-Based Model](block-based-model.md) — Mood as block attribute
- [Guided Capture](guided-capture.md) — Mood prompt type in templates
- [Streaks & Achievements](streaks-achievements.md) — Mood-based achievements

---

*Proposed February 2026.*
