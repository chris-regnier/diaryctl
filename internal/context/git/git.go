package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ContentProvider generates git status text for the editor buffer.
type ContentProvider struct {
	dir string // working directory; empty = current dir
}

// NewContentProvider creates a git content provider.
func NewContentProvider() *ContentProvider {
	return &ContentProvider{}
}

func (p *ContentProvider) Name() string { return "git" }

func (p *ContentProvider) Generate() (string, error) {
	branch := runGitCmd(p.dir, "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		return "", nil // not a git repo
	}

	// Count uncommitted files
	status := runGitCmd(p.dir, "status", "--porcelain")
	dirtyCount := 0
	if status != "" {
		dirtyCount = len(strings.Split(strings.TrimSpace(status), "\n"))
	}

	line1 := fmt.Sprintf("branch: %s | %d uncommitted files", branch, dirtyCount)

	// Most recent commit
	log := runGitCmd(p.dir, "log", "-1", "--format=%h %s (%ar)")
	if log == "" {
		return line1, nil
	}

	return line1 + "\n" + "latest: " + log, nil
}

// ContextResolver detects the current git branch as a context.
type ContextResolver struct {
	dir string
}

// NewContextResolver creates a git context resolver.
func NewContextResolver() *ContextResolver {
	return &ContextResolver{}
}

func (r *ContextResolver) Name() string { return "git" }

func (r *ContextResolver) Resolve() ([]string, error) {
	branch := runGitCmd(r.dir, "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" || branch == "HEAD" {
		return nil, nil // not a repo or detached HEAD
	}
	return []string{branch}, nil
}

func runGitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
