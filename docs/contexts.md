# Contexts

Contexts are semantic tags that group related diary entries. They help you filter and organize entries by project, sprint, topic, or any other grouping.

## How Contexts Work

Contexts are attached to entries at creation time. They come from two sources:

1. **Automatic** — detected by context resolvers (e.g. current git branch)
2. **Manual** — activated by you with `diaryctl context set`

Both types are combined and attached to every new entry you create.

## Context Resolvers

Resolvers automatically detect contexts from your environment. Enable them in your config:

```toml
context_resolvers = ["git"]
```

### Git Resolver

Detects the current git branch and uses it as a context. For example, if you're on branch `feature/auth`, entries created in that directory are automatically tagged with `feature/auth`.

## Context Providers

Providers inject content into the editor when creating entries. They are separate from resolvers — providers add text, resolvers add tags.

```toml
context_providers = ["git", "datetime"]
```

| Provider | Injected Content |
|----------|-----------------|
| `git` | Current branch, commit count, latest commit |
| `datetime` | Current date and time |

## Managing Contexts

### View Active Contexts

See which contexts will be attached to your next entry:

```bash
diaryctl context active
```

### Set a Manual Context

Activate a context manually. It stays active until you unset it.

```bash
diaryctl context set sprint:23
diaryctl context set project/backend
```

Manual context state is stored in `~/.diaryctl/active-contexts.json`.

### Unset a Manual Context

```bash
diaryctl context unset sprint:23
```

### List All Contexts

View all contexts that have been used:

```bash
diaryctl context list
```

### Show Context Details

```bash
diaryctl context show feature/auth
```

### Delete a Context

Remove a context definition (does not remove it from existing entries):

```bash
diaryctl context delete feature/auth --force
```

## Filtering by Context

Use `--context` to filter entries:

```bash
diaryctl list --context feature/auth
```

## Context Naming

Context names are alphanumeric with slashes allowed, making them natural for hierarchical grouping:

- `feature/auth`
- `sprint:23`
- `project/backend`
- `personal`
