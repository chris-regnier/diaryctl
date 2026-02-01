# Quickstart: Basic Diary CRUD

**Feature**: 001-diary-crud
**Date**: 2026-01-31

## Prerequisites

- Go 1.22+ installed
- Git

## Setup

```bash
# Clone and enter repository
git clone <repo-url> && cd diaryctl

# Install dependencies
go mod tidy

# Build
go build -o diaryctl .

# (Optional) Install to GOPATH/bin
go install .
```

## Create Your First Entry

```bash
# Inline content
diaryctl create "My first diary entry. Today I started using diaryctl."

# Output: Created entry a3kf9x2m (2026-01-31 14:30)
```

## Create Entry with Editor

```bash
# Opens $EDITOR (or vi if unset)
diaryctl create

# Write your entry, save, and close the editor
# Output: Created entry b7np4q1w (2026-01-31 14:35)
```

## List Entries

```bash
# List all entries (auto-paged in terminal)
diaryctl list

# Filter by date
diaryctl list --date 2026-01-31

# JSON output (for scripting)
diaryctl list --json
```

## View a Single Entry

```bash
diaryctl show a3kf9x2m
```

## Edit an Entry

```bash
# Opens in editor with existing content
diaryctl edit a3kf9x2m

# Or replace inline
diaryctl update a3kf9x2m "Updated content for my first entry"
```

## Delete an Entry

```bash
# With confirmation
diaryctl delete a3kf9x2m

# Skip confirmation
diaryctl delete a3kf9x2m --force
```

## Configuration

Default config at `~/.diaryctl/config.toml`:

```toml
# Storage backend: "markdown" (default) or "sqlite"
storage = "markdown"

# Data directory (default: ~/.diaryctl/)
# data_dir = "/custom/path"

# Editor override (default: $EDITOR > $VISUAL > vi)
# editor = "nano"
```

## Verify Installation

```bash
# Run all tests
go test ./...

# Check the binary
diaryctl --help
diaryctl create "Test entry"
diaryctl list
```
