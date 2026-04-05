# CLI Reference

Complete reference for all diaryctl commands and flags.

## Global Flags

These flags apply to every command:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | | Path to config file |
| `--json` | bool | `false` | Output in JSON format |
| `--storage` | string | | Storage backend override (`markdown` or `sqlite`) |

## Commands

### diaryctl (root)

When run with no subcommand in an interactive terminal, launches the TUI date picker. In non-interactive mode, behaves like `diaryctl daily`.

```bash
diaryctl
```

---

### diaryctl create

Create a new diary entry.

```
diaryctl create [content...]
```

Content can be provided as arguments, piped via stdin with `-`, or entered interactively in your editor (default when no content is given).

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--template` | string | | Template(s) to use, comma-separated |
| `--no-template` | bool | `false` | Skip the default template |

**Examples:**

```bash
diaryctl create "Today was great"
diaryctl create Today was a good day
echo "piped content" | diaryctl create -
diaryctl create                          # opens editor
diaryctl create --template daily
diaryctl create --template daily,prompts
```

---

### diaryctl jot

Append a timestamped note to today's entry. Creates today's entry if it doesn't exist.

```
diaryctl jot [text...]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--template` | string | | Template to use when creating today's entry |

**Examples:**

```bash
diaryctl jot "bought groceries"
diaryctl jot meeting went well
echo "note from pipe" | diaryctl jot -
```

---

### diaryctl edit

Edit an existing diary entry in your editor.

```
diaryctl edit <id>
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--template` | string | | Template(s) to append, comma-separated |

**Examples:**

```bash
diaryctl edit a3kf9x2m
diaryctl edit a3kf9x2m --template prompts
```

---

### diaryctl show

Display the full content and metadata of an entry.

```
diaryctl show <id>
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--id-only` | bool | `false` | Print just the entry ID |
| `--content-only` | bool | `false` | Print just the entry content |

**Examples:**

```bash
diaryctl show a3kf9x2m
diaryctl show a3kf9x2m --json
diaryctl show a3kf9x2m --content-only
```

---

### diaryctl list

List diary entries with preview, sorted by date (newest first).

```
diaryctl list
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--date` | string | | Filter by date (`YYYY-MM-DD`) |
| `--template` | string | | Filter by template name |
| `--context` | string | | Filter by context name |
| `--id-only` | bool | `false` | Print just entry IDs, one per line |

**Examples:**

```bash
diaryctl list
diaryctl list --date 2026-01-31
diaryctl list --template daily
diaryctl list --context feature/auth
diaryctl list --json
diaryctl list --id-only
```

---

### diaryctl delete

Permanently delete a diary entry. Prompts for confirmation unless `--force` is used.

```
diaryctl delete <id>
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Skip confirmation prompt |

**Examples:**

```bash
diaryctl delete a3kf9x2m
diaryctl delete a3kf9x2m --force
```

---

### diaryctl update

Replace the content of an existing entry with new text.

```
diaryctl update <id> <content>
```

**Examples:**

```bash
diaryctl update a3kf9x2m "Updated content here"
echo "new content" | diaryctl update a3kf9x2m -
```

---

### diaryctl today

View or edit today's daily entry. Creates one automatically if none exists.

```
diaryctl today
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--edit` | bool | `false` | Open today's entry in the editor |
| `--id-only` | bool | `false` | Print just the entry ID |
| `--content-only` | bool | `false` | Print just the content |

**Examples:**

```bash
diaryctl today
diaryctl today --edit
diaryctl today --id-only
diaryctl today --json
```

---

### diaryctl daily

Browse diary entries by day. In an interactive terminal, launches a date picker TUI. In non-interactive mode (piped output, `--no-interactive`, or `--json`), prints a grouped-by-day summary.

```
diaryctl daily
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--from` | string | | Start date filter (`YYYY-MM-DD`, inclusive) |
| `--to` | string | | End date filter (`YYYY-MM-DD`, inclusive) |
| `--no-interactive` | bool | `false` | Force non-interactive output |
| `--template` | string | | Filter by template name |

**Examples:**

```bash
diaryctl daily
diaryctl daily --from 2026-01-01 --to 2026-01-31
diaryctl daily --template daily
diaryctl daily --no-interactive
diaryctl daily --json
```

---

### diaryctl template

Manage reusable content templates. See [Templates](templates.md) for a full guide.

#### template list

```bash
diaryctl template list
```

#### template show

```bash
diaryctl template show <name>
```

#### template create

Create a new template. Opens your editor, or reads from stdin with `-`.

Templates support Go `text/template` syntax for variables (e.g. `{{.name}}`).

```bash
diaryctl template create <name>
diaryctl template create daily
echo "# Daily Entry" | diaryctl template create daily -
```

#### template edit

```bash
diaryctl template edit <name>
```

#### template delete

Requires `--force` flag.

```bash
diaryctl template delete <name> --force
```

---

### diaryctl context

Manage semantic contexts for grouping entries. See [Contexts](contexts.md) for a full guide.

#### context list

```bash
diaryctl context list
```

#### context show

```bash
diaryctl context show <name>
```

#### context delete

```bash
diaryctl context delete <name> --force
```

#### context set

Activate a manual context. All new entries will be tagged with it.

```bash
diaryctl context set <name>
```

#### context unset

Deactivate a manual context.

```bash
diaryctl context unset <name>
```

#### context active

Show all currently active contexts (automatic + manual).

```bash
diaryctl context active
```

---

### diaryctl status

Show diary status for shell prompt integration. Outputs today indicator, streak count, and context information.

```
diaryctl status
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--env` | bool | `false` | Output shell environment variable assignments |
| `--refresh` | bool | `false` | Force cache refresh |
| `--format` | string | | Go template format string |

**Examples:**

```bash
diaryctl status                    # e.g. "✓ 5🔥 daily"
diaryctl status --env              # export DIARYCTL_TODAY=1 ...
diaryctl status --refresh
diaryctl status --format "{{.TodayIcon}} {{.Streak}}{{.StreakIcon}}"
```

---

### diaryctl init

Output shell integration script for `eval`. Sets up completions, prompt hooks, and the `diaryctl_prompt_info` helper function.

```
diaryctl init <shell>
```

Supported shells: `bash`, `zsh`

**Examples:**

```bash
# Add to ~/.bashrc
eval "$(diaryctl init bash)"

# Add to ~/.zshrc
eval "$(diaryctl init zsh)"
```

---

### diaryctl mcp-serve

Start a Model Context Protocol (MCP) server on stdio. Allows MCP clients like Claude Desktop to interact with your diary.

```
diaryctl mcp-serve
```

See [MCP Server](mcp-server.md) for setup details.
