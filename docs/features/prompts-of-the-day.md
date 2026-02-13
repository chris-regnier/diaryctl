# Prompts of the Day

**Status:** Proposed

## Overview

Rotating writing prompts from a configurable pool, shown in the TUI and optionally in the shell prompt. Reduces blank-page friction by suggesting what to write about.

## Proposed Interface

```bash
# Show today's prompt
diaryctl prompt

# Add custom prompts to the pool
diaryctl prompt add "What are you grateful for today?"

# List all prompts
diaryctl prompt list

# Create entry from today's prompt
diaryctl create --from-prompt
```

## Core Concepts

- Built-in prompt pool with common journaling questions
- User-defined prompts stored in config or data directory
- Deterministic daily selection (hash of date + pool size)
- TUI shows prompt on the Today screen when no entries exist
- Shell integration via `DIARYCTL_PROMPT` environment variable

## Related Features

- [Guided Capture](guided-capture.md) — Prompts feed into guided flows
- [Recurring Entries](recurring-entries.md) — Time-based prompt suggestions

---

*Proposed February 2026.*
