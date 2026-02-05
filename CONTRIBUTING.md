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
