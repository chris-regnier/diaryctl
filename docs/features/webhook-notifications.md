# Webhook Notifications

**Status:** Proposed

## Overview

Fire HTTP webhooks on diary events (entry creation, streaks, milestones) to enable external integrations with services like Slack, Discord, Notion, or custom endpoints.

## Proposed Interface

```bash
# Add a webhook
diaryctl webhook add --url https://hooks.slack.com/... --events create,milestone

# List webhooks
diaryctl webhook list

# Remove a webhook
diaryctl webhook remove <id>

# Test a webhook
diaryctl webhook test <id>
```

## Configuration

```toml
[[webhooks]]
url = "https://hooks.slack.com/services/..."
events = ["create", "milestone"]
format = "slack"  # "slack", "discord", "json"
```

## Core Concepts

- Event types: `create`, `update`, `delete`, `milestone`, `streak`
- Payload formats: generic JSON, Slack blocks, Discord embed
- Async fire-and-forget (don't block CLI on webhook failure)
- Retry with exponential backoff for transient failures
- Webhook secrets for HMAC signature verification

## Related Features

- [Streaks & Achievements](streaks-achievements.md) — Milestone events
- [Git Hook Integration](git-hooks.md) — Event-driven entry creation

---

*Proposed February 2026.*
