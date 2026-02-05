# Public Release Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create all required documentation files (LICENSE, README.md, CONTRIBUTING.md, SECURITY.md) to prepare diaryctl for public release.

**Architecture:** Standard open-source project documentation structure following GitHub best practices. MIT license for maximum permissiveness. Documentation emphasizes the pluggable storage architecture (Markdown/SQLite) and TUI capabilities.

**Tech Stack:** Markdown, MIT License

---

## Task 1: Create MIT License File

**Files:**
- Create: `LICENSE`

**Step 1: Write LICENSE file**

Create the MIT License file with current year and copyright holder:

```text
MIT License

Copyright (c) 2026 Chris Regnier

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

**Step 2: Verify file exists**

Run: `cat LICENSE`
Expected: File contains MIT License text with correct year (2026)

**Step 3: Commit**

```bash
git add LICENSE
git commit -m "docs: add MIT license"
```

---

## Task 2: Create README.md

**Files:**
- Create: `README.md`

**Step 1: Write README.md content**

Create comprehensive README with project overview, features, installation, and usage:

```markdown
# diaryctl

A powerful command-line diary management tool with pluggable storage backends and an elegant TUI.

## Features

- **Pluggable Storage**: Choose between Markdown files or SQLite/Turso database
- **Rich CLI**: Full-featured commands for creating, editing, listing, and managing diary entries
- **Interactive TUI**: Beautiful terminal user interface built with Bubble Tea
- **Context Tracking**: Automatic context detection from git branches, manual context tagging
- **Template Support**: Create and use custom entry templates
- **Shell Integration**: Export environment variables for shell prompts
- **Daily Summaries**: View entries by date range with filtering

## Installation

### From Source

```bash
go install github.com/chris-regnier/diaryctl@latest
```

### Building Locally

```bash
git clone https://github.com/chris-regnier/diaryctl.git
cd diaryctl
go build -o diaryctl
```

## Quick Start

### Initialize Configuration

```bash
# Create default config (uses markdown storage in ~/.diaryctl)
diaryctl init

# Or specify SQLite backend
diaryctl init --storage sqlite
```

### Basic Usage

```bash
# Create a new entry interactively
diaryctl create

# Quick jot down a thought
diaryctl jot "Had a great idea for the project"

# List recent entries
diaryctl list

# Show today's entries
diaryctl today

# Launch interactive TUI
diaryctl
```

## Storage Backends

### Markdown (Default)

Stores entries as individual markdown files with YAML frontmatter:

```yaml
---
id: 01HQXYZ...
created: 2026-02-04T10:30:00Z
contexts:
  - feature/auth
  - sprint:23
template: daily
---

Entry content goes here...
```

### SQLite/Turso

Stores entries in a SQLite database compatible with Turso for remote sync.

## Context Tracking

diaryctl automatically tracks context from:

- **Git branches**: Current branch becomes a context
- **Manual tags**: Set custom contexts with `diaryctl context set project:auth`
- **Date/time**: Automatic datetime context provider

Filter entries by context:

```bash
diaryctl list --context feature/auth
```

## Templates

Create reusable entry templates:

```bash
# Create a new template
diaryctl template create standup

# Use template when creating entry
diaryctl create --template standup
```

## Configuration

Configuration file location: `~/.config/diaryctl/config.yaml`

```yaml
storage: markdown  # or sqlite
data_dir: ~/.diaryctl/data
editor: hx  # or vim, nano, etc.
context_providers:
  - git
  - datetime
context_resolvers:
  - git
```

## Commands

| Command | Description |
|---------|-------------|
| `diaryctl` | Launch interactive TUI (when in TTY) |
| `diaryctl create` | Create a new entry |
| `diaryctl edit <id>` | Edit an entry |
| `diaryctl show <id>` | Display an entry |
| `diaryctl list` | List entries |
| `diaryctl delete <id>` | Delete an entry |
| `diaryctl jot <text>` | Quick entry creation |
| `diaryctl today` | Show today's entries |
| `diaryctl daily` | Show entries in date range |
| `diaryctl context` | Manage contexts |
| `diaryctl template` | Manage templates |
| `diaryctl status` | Show current status |

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
diaryctl/
├── cmd/              # Cobra commands
├── internal/
│   ├── config/       # Configuration management
│   ├── context/      # Context providers and resolvers
│   ├── daily/        # Daily view logic
│   ├── editor/       # Editor integration
│   ├── entry/        # Entry domain model
│   ├── shell/        # Shell integration
│   ├── storage/      # Storage interface
│   │   ├── markdown/ # Markdown backend
│   │   └── sqlite/   # SQLite backend
│   ├── template/     # Template management
│   └── ui/           # TUI components
└── main.go
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security policy and vulnerability reporting.
```

**Step 2: Verify README is well-formatted**

Run: `head -20 README.md`
Expected: File contains header, badges section, and feature list

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README"
```

---

## Task 3: Create CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

**Step 1: Write CONTRIBUTING.md content**

Create contribution guidelines with code standards and PR process:

```markdown
# Contributing to diaryctl

Thank you for your interest in contributing to diaryctl! This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and constructive
- Welcome newcomers and help them learn
- Focus on what is best for the project and community

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/diaryctl.git
   cd diaryctl
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/chris-regnier/diaryctl.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```

## Development Workflow

### Creating a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Use branch naming conventions:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or fixes

### Making Changes

1. **Write tests first** (TDD approach preferred)
2. Implement your changes
3. Ensure all tests pass: `go test ./...`
4. Format code: `go fmt ./...`
5. Run linting (if configured): `golangci-lint run`

### Commit Messages

Follow conventional commit format:

```
type(scope): brief description

Longer description if needed.

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

Examples:
```
feat(storage): add postgres backend support
fix(context): handle detached HEAD state in git resolver
docs: update installation instructions
```

### Testing

- Write unit tests for new functionality
- Ensure test coverage doesn't decrease
- Test both storage backends (markdown and sqlite)
- Include table-driven tests for multiple scenarios

Example test structure:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Pull Request Process

1. Update documentation if needed
2. Add tests for new functionality
3. Ensure all tests pass
4. Update CHANGELOG.md (if applicable)
5. Push to your fork
6. Create a Pull Request with:
   - Clear title and description
   - Reference any related issues
   - Screenshots/examples if UI changes

### PR Review Process

- Maintainers will review within 3-5 business days
- Address feedback and update your PR
- Once approved, maintainers will merge

## Project Standards

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused (single responsibility)
- Prefer explicit over implicit
- Handle errors properly (no silent failures)

### Architecture Principles

- **DRY** (Don't Repeat Yourself)
- **YAGNI** (You Aren't Gonna Need It) - don't over-engineer
- **Pluggable design** - storage backends are interfaces
- **Testable code** - dependency injection for testability

### Storage Backend Guidelines

New storage backends must implement `storage.Storage` interface:

```go
type Storage interface {
    Create(entry *entry.Entry) error
    Read(id string) (*entry.Entry, error)
    Update(entry *entry.Entry) error
    Delete(id string) error
    List(opts ListOptions) ([]*entry.Entry, error)
    // ... context and template methods
}
```

## Areas for Contribution

### Good First Issues

Look for issues tagged `good-first-issue`:
- Documentation improvements
- Test coverage expansion
- Small bug fixes
- Example templates

### Larger Features

- New storage backends (PostgreSQL, MongoDB, etc.)
- Additional context providers (Jira, Linear, etc.)
- Export/import functionality
- Search improvements
- CLI enhancements

## Questions?

- Open a GitHub Discussion for questions
- Check existing issues before creating new ones
- Join community chat (if available)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
```

**Step 2: Verify CONTRIBUTING.md**

Run: `grep -E "^##" CONTRIBUTING.md`
Expected: Shows section headers including Code of Conduct, Development Workflow, PR Process

**Step 3: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add contribution guidelines"
```

---

## Task 4: Create SECURITY.md

**Files:**
- Create: `SECURITY.md`

**Step 1: Write SECURITY.md content**

Create security policy with vulnerability reporting instructions:

```markdown
# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### 1. Do Not Open a Public Issue

Please **do not** create a public GitHub issue for security vulnerabilities. Public disclosure could put users at risk.

### 2. Report Privately

Send details to: **[security contact - update with actual email]**

Include in your report:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)

### 3. Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 5 business days
- **Fix timeline**: Depends on severity
  - Critical: Within 7 days
  - High: Within 14 days
  - Medium: Within 30 days
  - Low: Next regular release

### 4. Disclosure Process

1. We'll acknowledge receipt of your vulnerability report
2. We'll investigate and validate the issue
3. We'll develop and test a fix
4. We'll release the fix in a new version
5. We'll publicly disclose the vulnerability (with credit to reporter if desired)

## Security Considerations

### Local Data Storage

diaryctl stores diary entries locally:
- **Markdown backend**: Plain text files in `~/.diaryctl/data/`
- **SQLite backend**: Database file in `~/.diaryctl/data/diary.db`

**Important**: These files are not encrypted. Do not store highly sensitive information without additional encryption at the filesystem level.

### Configuration Security

Configuration file: `~/.config/diaryctl/config.yaml`

- Uses standard file permissions (user read/write only)
- Does not store credentials or API keys
- Safe to commit example configs (without personal data)

### Command Injection Risks

diaryctl uses `exec.Command` for:
- Editor invocation (via `$EDITOR` env var)
- Git operations (hardcoded git commands only)

User input is **not** passed directly to shell commands. All command execution uses parameterized arguments.

### Dependencies

We regularly update dependencies to address security vulnerabilities. Dependency security is monitored via:
- GitHub Dependabot
- Go vulnerability database (`govulncheck`)

### Turso/SQLite Remote Sync

If using Turso for remote database sync:
- Connections use HTTPS/WSS encryption
- Authentication tokens should be stored securely
- Tokens should have minimal necessary permissions

## Security Best Practices for Users

1. **File Permissions**: Ensure data directory has restricted permissions:
   ```bash
   chmod 700 ~/.diaryctl/data
   ```

2. **Backups**: Regularly backup your diary data
   ```bash
   tar -czf diary-backup.tar.gz ~/.diaryctl/data
   ```

3. **Sensitive Data**: Consider full-disk encryption for highly sensitive diary entries

4. **Editor Security**: Use a trusted text editor via `$EDITOR` environment variable

5. **Remote Sync**: If using Turso, rotate access tokens periodically

## Known Security Limitations

- No built-in encryption for stored diary entries
- No authentication/authorization (single-user CLI tool)
- Editor invocation trusts `$EDITOR` environment variable

## Security Updates

Security fixes will be announced via:
- GitHub Security Advisories
- Release notes
- README.md security section

## Questions?

For security-related questions (not vulnerabilities), open a GitHub Discussion or issue.
```

**Step 2: Verify SECURITY.md structure**

Run: `grep -E "^##" SECURITY.md`
Expected: Shows major sections including Reporting, Security Considerations, Best Practices

**Step 3: Commit**

```bash
git add SECURITY.md
git commit -m "docs: add security policy"
```

---

## Task 5: Final Verification

**Step 1: Verify all documentation files exist**

Run: `ls -la LICENSE README.md CONTRIBUTING.md SECURITY.md`
Expected: All four files exist with reasonable sizes

**Step 2: Check git status**

Run: `git status`
Expected: Working tree clean, all documentation committed

**Step 3: Review commit history**

Run: `git log --oneline -5`
Expected: Shows 4 commits for documentation (LICENSE, README, CONTRIBUTING, SECURITY)

**Step 4: Verify README renders well**

Run: `head -50 README.md | grep -E "^#"`
Expected: Shows proper markdown headers with hierarchy

---

## Post-Implementation Checklist

After completing this plan:

- [ ] All four documentation files created (LICENSE, README.md, CONTRIBUTING.md, SECURITY.md)
- [ ] LICENSE contains MIT license text with correct year and copyright holder
- [ ] README.md includes installation instructions, feature list, and usage examples
- [ ] CONTRIBUTING.md explains development workflow and coding standards
- [ ] SECURITY.md provides vulnerability reporting process
- [ ] All files committed with descriptive commit messages
- [ ] Consider updating git remote to point to public repository URL
- [ ] Consider adding GitHub Actions CI/CD workflow
- [ ] Update SECURITY.md with actual security contact email
- [ ] Add repository badges to README once public (build status, Go version, license)

## Notes

- The MIT license provides maximum permissiveness for open source use
- Security policy emphasizes that local storage is unencrypted
- Documentation emphasizes pluggable storage architecture as key differentiator
- CONTRIBUTING.md follows Go best practices and emphasizes TDD
- All documentation can be extended/refined after initial public release
