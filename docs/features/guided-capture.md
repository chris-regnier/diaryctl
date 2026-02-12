# Guided Capture

**Status:** Designed  
**Design Doc:** `docs/plans/2025-02-01-workflow-features-design.md` (Feature 4)

## Overview

Template-driven prompt flows that walk users through structured data entry. When invoked with `--guided`, diaryctl presents a TUI prompt for each template variable and assembles the entry from answers.

## Use Cases

- **Daily standups** — Prompt for "What did you do yesterday?", "What are you working on today?", "Any blockers?"
- **Mood tracking** — Guided prompts for mood, energy, sleep quality
- **Meeting notes** — Prompt for attendees, decisions, action items
- **Retrospectives** — Prompt for what went well, what didn't, action items

## Template Format

Templates declare prompts in their YAML front-matter:

```markdown
---
name: standup
prompts:
  - key: yesterday
    question: "What did you accomplish yesterday?"
  - key: today
    question: "What are you working on today?"
  - key: blockers
    question: "Any blockers?"
---

## Yesterday
{{yesterday}}

## Today
{{today}}

## Blockers
{{blockers}}
```

### Prompt Options

```markdown
---
name: mood-check
prompts:
  - key: mood
    question: "How are you feeling?"
    type: select
    options: ["Great", "Good", "Okay", "Bad", "Terrible"]
  - key: energy
    question: "Energy level (1-10)?"
    type: number
    min: 1
    max: 10
  - key: notes
    question: "Any additional notes?"
    type: text
    multiline: true
---

Mood: {{mood}}
Energy: {{energy}}/10

Notes:
{{notes}}
```

## CLI Interface

```bash
# Interactive guided mode
diaryctl create --template standup --guided

# Guided with editor review
diaryctl create --template standup --guided --edit

# JSON input for scripting
echo '{"yesterday":"auth work","today":"API endpoints","blockers":"none"}' \
  | diaryctl create --template standup --guided -

# Partial JSON (missing keys prompt interactively)
echo '{"yesterday":"auth work"}' | diaryctl create --template standup --guided -
```

## TUI Flow

```
┌─ Standup ──────────────────────┐
│                                 │
│  What did you accomplish        │
│  yesterday?                     │
│                                 │
│  > Fixed the auth bug and      │
│    started on API design       │
│                                 │
│  [1/3]  enter: next  esc: skip │
└─────────────────────────────────┘
```

```
┌─ Standup ──────────────────────┐
│                                 │
│  What are you working on        │
│  today?                         │
│                                 │
│  > Implementing the login      │
│    endpoint                     │
│                                 │
│  [2/3]  enter: next  esc: skip │
└─────────────────────────────────┘
```

```
┌─ Standup ──────────────────────┐
│                                 │
│  Any blockers?                  │
│                                 │
│  > Waiting for design review   │
│                                 │
│  [3/3]  enter: save  esc: skip │
└─────────────────────────────────┘
```

## Implementation

### Template Structure Extension

```go
type Template struct {
    ID         string
    Name       string
    Content    string
    Prompts    []Prompt          // New field
    Attributes map[string]string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type Prompt struct {
    Key       string
    Question  string
    Type      string  // "text" (default), "select", "number"
    Options   []string // For select type
    Multiline bool    // For text type
    Min       int     // For number type
    Max       int     // For number type
}
```

### Guided Flow Algorithm

```go
func GuidedCapture(tmpl Template, input map[string]string, flags Flags) (string, error) {
    answers := make(map[string]string)

    // 1. Pre-fill from JSON input
    for k, v := range input {
        answers[k] = v
    }

    // 2. Prompt for missing keys
    for _, prompt := range tmpl.Prompts {
        if _, ok := answers[prompt.Key]; !ok {
            answer, err := TUI.Prompt(prompt)  // Interactive prompt
            if err != nil {
                return "", err
            }
            answers[prompt.Key] = answer
        }
    }

    // 3. Render template with answers
    content, err := RenderTemplate(tmpl.Content, answers)
    if err != nil {
        return "", err
    }

    // 4. Optional: Open in editor for review
    if flags.Edit {
        content, err = OpenEditor(content)
    }

    return content, nil
}
```

### Template Rendering

Uses Go's `text/template` with user-provided values:

```go
func RenderTemplate(templateContent string, values map[string]string) (string, error) {
    tmpl, err := template.New("guided").Parse(templateContent)
    if err != nil {
        return "", err
    }

    var buf strings.Builder
    if err := tmpl.Execute(&buf, values); err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

## Input Modes

### Interactive Mode (Default)

Presents each question in a TUI:

```bash
diaryctl create --template standup --guided
```

### JSON Mode

Accepts answers via stdin:

```bash
echo '{"yesterday":"auth work","today":"API","blockers":"none"}' | \
  diaryctl create --template standup --guided -
```

### Partial JSON

Provide some answers, prompt for missing:

```bash
# Only provides "yesterday", prompts for "today" and "blockers"
echo '{"yesterday":"auth work"}' | diaryctl create --template standup --guided -
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Template has no prompts | Falls back to editor (regular create) |
| JSON has extra keys | Ignored (warn to stderr) |
| JSON has wrong type | Converts to string |
| User skips required prompt | Empty string for that key |
| Template syntax error | Error before prompts |

## Design Decisions

### Minimal Template Engine

- Only `{{key}}` replacement, no expressions/conditionals
- No loops (templates generate single blocks)
- No functions initially (can add Sprig later)

### Keys vs. Names

- `key` is the template variable (`{{yesterday}}`)
- `name` is the template identifier (filename/metadata)

### Optional vs. Required

All prompts are optional by default. Users can press `esc` to skip. This reduces friction and allows partial entries.

### No Validation

No input validation initially. Future versions could add:

```yaml
prompts:
  - key: mood
    question: "Mood?"
    validate: "^(great|good|okay|bad)$"
    validate_error: "Please enter: great, good, okay, or bad"
```

## Related Features

- [Block-Based Model](block-based-model.md) — Templates generate blocks with attributes
- [TUI Template Integration](tui-template-integration.md) — Template picker in TUI

---

*See design doc for original specification.*
