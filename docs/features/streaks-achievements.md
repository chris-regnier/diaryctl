# Streaks & Achievements

**Status:** Proposed

## Overview

Gamification layer with configurable milestones to encourage consistent journaling. Builds on the existing streak calculation in shell integration.

## Proposed Interface

```bash
# View current streaks and achievements
diaryctl stats

# Detailed achievement list
diaryctl stats achievements

# Output for scripting
diaryctl stats --format json
```

## Core Concepts

- **Streaks:** Current and longest consecutive day counts (already partially implemented in shell)
- **Milestones:** Configurable triggers (7-day streak, 30-day streak, 100 entries, etc.)
- **Achievement types:** Streak-based, count-based, template-based (e.g., "used 5 different templates")
- **TUI display:** Achievement badges or indicators on the Today screen
- **Notifications:** Optional shell notification when milestones are hit

## Configuration

```toml
[achievements]
enabled = true
milestones = [7, 30, 100, 365]  # Streak day milestones
```

## Related Features

- [Shell Integration](shell-integration.md) — Existing streak display
- [Entry Statistics](entry-statistics.md) — Broader analytics

---

*Proposed February 2026.*
