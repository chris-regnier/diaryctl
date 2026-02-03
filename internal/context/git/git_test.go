package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")
	return dir
}

func TestContentProvider_InRepo(t *testing.T) {
	dir := setupGitRepo(t)
	p := &ContentProvider{dir: dir}
	out, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "branch: main") {
		t.Errorf("expected branch info, got %q", out)
	}
	if !strings.Contains(out, "initial commit") {
		t.Errorf("expected commit info, got %q", out)
	}
}

func TestContentProvider_NotARepo(t *testing.T) {
	p := &ContentProvider{dir: t.TempDir()}
	out, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty string for non-repo, got %q", out)
	}
}

func TestContentProvider_DirtyFiles(t *testing.T) {
	dir := setupGitRepo(t)
	// Create an untracked file
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}
	p := &ContentProvider{dir: dir}
	out, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1 uncommitted files") {
		t.Errorf("expected 1 uncommitted file, got %q", out)
	}
}

func TestContextResolver_InRepo(t *testing.T) {
	dir := setupGitRepo(t)
	r := &ContextResolver{dir: dir}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "main" {
		t.Errorf("expected [main], got %v", names)
	}
}

func TestContextResolver_OnBranch(t *testing.T) {
	dir := setupGitRepo(t)
	cmd := exec.Command("git", "checkout", "-b", "feature/auth")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout: %v\n%s", err, out)
	}
	r := &ContextResolver{dir: dir}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "feature/auth" {
		t.Errorf("expected [feature/auth], got %v", names)
	}
}

func TestContextResolver_NotARepo(t *testing.T) {
	r := &ContextResolver{dir: t.TempDir()}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty for non-repo, got %v", names)
	}
}

func TestContextResolver_Name(t *testing.T) {
	r := NewContextResolver()
	if r.Name() != "git" {
		t.Errorf("got name %q", r.Name())
	}
}

func TestContentProvider_Name(t *testing.T) {
	p := NewContentProvider()
	if p.Name() != "git" {
		t.Errorf("got name %q", p.Name())
	}
}
