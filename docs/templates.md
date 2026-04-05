# Templates

Templates provide reusable content scaffolding for diary entries. Use them for recurring formats like daily standups, retrospectives, or meeting notes.

## Creating a Template

Open your editor to write a template:

```bash
diaryctl template create standup
```

Or pipe content from stdin:

```bash
cat <<'EOF' | diaryctl template create standup -
## Standup — {{.date}}

### Yesterday
-

### Today
-

### Blockers
-
EOF
```

## Template Variables

Templates use Go `text/template` syntax. Variable names must start with a letter.

```
Hello {{.name}}, today is {{.date}}.
```

When you create an entry with a template containing variables, diaryctl prompts you for values.

## Using Templates

### With `create`

```bash
diaryctl create --template standup
diaryctl create --template daily,prompts    # multiple templates
```

### With `jot`

When jotting to today's entry, use `--template` to specify which template to use if today's entry needs to be created:

```bash
diaryctl jot --template daily "quick note"
```

### With `edit`

Append template content to an existing entry:

```bash
diaryctl edit a3kf9x2m --template prompts
```

### Default Template

Set a default template that is applied to every new entry:

```toml
# ~/.diaryctl/config.toml
default_template = "daily"
```

Skip the default template for a single entry with `--no-template`:

```bash
diaryctl create --no-template
```

## Managing Templates

### List All Templates

```bash
diaryctl template list
```

### Show a Template

```bash
diaryctl template show standup
```

### Edit a Template

```bash
diaryctl template edit standup
```

### Delete a Template

```bash
diaryctl template delete standup --force
```

## Template Storage

Templates are stored in the active storage backend:

- **Markdown**: `~/.diaryctl/templates/` as individual files
- **SQLite**: `templates` table in the database
