# Media Attachments

**Status:** Proposed

## Overview

Attach images, files, and other media to diary entries. Supports inline references in markdown content and browsing in the TUI.

## Proposed Interface

```bash
# Create entry with attachment
diaryctl create --attach screenshot.png

# Add attachment to existing entry
diaryctl attach <entry-id> photo.jpg

# List attachments
diaryctl list --entry <id> --attachments
```

## Core Concepts

- **Storage:** Attachments stored alongside entries (markdown backend: `attachments/` subdirectory; SQLite: blob table or external files)
- **References:** `![description](attachment:filename.png)` syntax in entry content
- **TUI display:** Image preview via terminal image protocols (Kitty, iTerm2, Sixel) or fallback to filename listing
- **Size limits:** Configurable max attachment size, warn on large files
- **Types:** Images (png, jpg, gif), documents (pdf), and arbitrary files

## Design Considerations

- Keep entries portable — attachments should travel with exports
- Don't bloat the data directory — consider compression or linking
- TUI image rendering depends on terminal capabilities — graceful degradation needed

## Related Features

- [Export/Import](export-import.md) — Include attachments in exports
- [Block-Based Model](block-based-model.md) — Attachments as block attribute or typed block
- [Markdown Backend](markdown-backend.md) — File storage layout

---

*Proposed February 2026.*
