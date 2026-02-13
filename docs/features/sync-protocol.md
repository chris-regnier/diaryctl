# Sync Protocol

**Status:** Proposed

## Overview

Conflict-free synchronization of diary data between devices. Enables multi-device journaling without data loss.

## Core Concepts

- **Sync strategies:** Last-write-wins (simple), CRDTs (robust), or server-mediated merge
- **Transport options:** Direct file sync (Syncthing/rsync), Turso replication (SQLite backend), or custom HTTP API
- **Conflict resolution:** Entries are append-only by nature, reducing conflict surface
- **Identity:** Device ID for tracking origin of entries

## Proposed Approaches

1. **Turso replication** — Leverage existing SQLite/Turso backend for built-in multi-device sync
2. **File-based sync** — Compatible with Syncthing, iCloud Drive, Dropbox for markdown backend
3. **Custom sync server** — Purpose-built sync with conflict resolution (most complex)

## Related Features

- [SQLite Backend](sqlite-backend.md) — Turso already supports replication
- [Markdown Backend](markdown-backend.md) — File-based sync compatibility
- [Export/Import](export-import.md) — Manual data transfer fallback

---

*Proposed February 2026.*
