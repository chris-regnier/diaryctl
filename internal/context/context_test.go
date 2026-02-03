package context

import (
	"fmt"
	"testing"
)

type stubProvider struct {
	name   string
	output string
	err    error
}

func (s *stubProvider) Name() string             { return s.name }
func (s *stubProvider) Generate() (string, error) { return s.output, s.err }

func TestComposeContent_empty(t *testing.T) {
	got := ComposeContent(nil, "template content")
	if got != "template content" {
		t.Errorf("got %q, want %q", got, "template content")
	}
}

func TestComposeContent_providersOnly(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday, February 2, 2026"},
	}
	got := ComposeContent(providers, "")
	if got != "# Monday, February 2, 2026" {
		t.Errorf("got %q", got)
	}
}

func TestComposeContent_providersAndTemplate(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday, February 2, 2026"},
		&stubProvider{name: "git", output: "branch: main | 0 uncommitted files"},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday, February 2, 2026\nbranch: main | 0 uncommitted files\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComposeContent_skipsEmptyProviders(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday"},
		&stubProvider{name: "git", output: ""},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComposeContent_skipsFailedProviders(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday"},
		&stubProvider{name: "git", output: "", err: fmt.Errorf("not a git repo")},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComposeContent_allEmpty(t *testing.T) {
	got := ComposeContent(nil, "")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
