---
description: Run Gavel code analysis on the current branch diff against main.
---

## User Input

```text
$ARGUMENTS
```

## Goal

Run `gavel analyze` on the diff between the current branch and main, then present findings with triage.

## Steps

### 1. Generate the diff

Determine the base branch. Default is `main`, but the user can override via arguments (e.g., `/gavel develop`).

```bash
BASE="${arguments:-main}"
git diff "$BASE"...HEAD
```

If the diff is empty, report "No changes to analyze" and stop.

### 2. Run Gavel

Pipe the diff to gavel:

```bash
git diff "$BASE"...HEAD | gavel analyze --diff - --policies .gavel
```

### 3. Parse and present findings

Read the gavel output. Present a summary organized by severity:

**Format:**

```
## Gavel Analysis: [PASS/REJECT]

### Errors (N)
- **[rule]** file:line — message
  → recommendation

### Warnings (N)
- **[rule]** file:line — message
  → recommendation
```

### 4. Triage findings

For each error-level finding, assess whether it's:
- **Genuine bug** — propose a fix
- **False positive** — explain why (e.g., Bubble Tea value receiver pattern)
- **Needs investigation** — read the relevant code to determine

For warnings, briefly note which are worth addressing vs which are noise.

### 5. Act on genuine issues

If there are genuine bugs, fix them immediately. Commit with a message like:

```
fix(scope): address gavel finding — description
```

Re-run gavel after fixes to confirm the issue is resolved.
