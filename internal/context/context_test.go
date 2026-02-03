package context

import (
	"fmt"
	"testing"

	"github.com/chris-regnier/diaryctl/internal/storage"
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

type stubResolver struct {
	name  string
	names []string
	err   error
}

func (s *stubResolver) Name() string               { return s.name }
func (s *stubResolver) Resolve() ([]string, error)  { return s.names, s.err }

type mockContextStore struct {
	contexts map[string]storage.Context
}

func (m *mockContextStore) GetContextByName(name string) (storage.Context, error) {
	c, ok := m.contexts[name]
	if !ok {
		return storage.Context{}, storage.ErrNotFound
	}
	return c, nil
}

func (m *mockContextStore) CreateContext(c storage.Context) error {
	m.contexts[c.Name] = c
	return nil
}

func TestResolveActiveContexts_empty(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	refs, warnings := ResolveActiveContexts(nil, nil, ms)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestResolveActiveContexts_manualOnly(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	refs, warnings := ResolveActiveContexts(nil, []string{"sprint:23"}, ms)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(refs) != 1 || refs[0].ContextName != "sprint:23" {
		t.Errorf("got refs %v", refs)
	}
	if _, ok := ms.contexts["sprint:23"]; !ok {
		t.Error("expected context to be auto-created")
	}
}

func TestResolveActiveContexts_resolverOnly(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: []string{"feature/auth"}},
	}
	refs, warnings := ResolveActiveContexts(resolvers, nil, ms)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(refs) != 1 || refs[0].ContextName != "feature/auth" {
		t.Errorf("got refs %v", refs)
	}
}

func TestResolveActiveContexts_deduplicates(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: []string{"feature/auth"}},
	}
	refs, warnings := ResolveActiveContexts(resolvers, []string{"feature/auth"}, ms)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(refs) != 1 {
		t.Errorf("expected 1 deduplicated ref, got %d", len(refs))
	}
}

func TestResolveActiveContexts_skipsFailedResolver(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: nil, err: fmt.Errorf("not a git repo")},
	}
	refs, warnings := ResolveActiveContexts(resolvers, []string{"sprint:23"}, ms)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %v", warnings)
	}
	if len(refs) != 1 || refs[0].ContextName != "sprint:23" {
		t.Errorf("got refs %v", refs)
	}
}

func TestResolveActiveContexts_reusesExisting(t *testing.T) {
	existing := storage.Context{ID: "existing1", Name: "sprint:23", Source: "manual"}
	ms := &mockContextStore{contexts: map[string]storage.Context{"sprint:23": existing}}
	refs, warnings := ResolveActiveContexts(nil, []string{"sprint:23"}, ms)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(refs) != 1 || refs[0].ContextID != "existing1" {
		t.Errorf("expected existing context ID, got %v", refs)
	}
}

func TestLookupContentProvider(t *testing.T) {
	p := LookupContentProvider("datetime")
	if p == nil {
		t.Fatal("expected datetime provider")
	}
	if p.Name() != "datetime" {
		t.Errorf("got name %q", p.Name())
	}
}

func TestLookupContentProvider_git(t *testing.T) {
	p := LookupContentProvider("git")
	if p == nil {
		t.Fatal("expected git provider")
	}
	if p.Name() != "git" {
		t.Errorf("got name %q", p.Name())
	}
}

func TestLookupContentProvider_unknown(t *testing.T) {
	p := LookupContentProvider("nonexistent")
	if p != nil {
		t.Errorf("expected nil for unknown provider")
	}
}

func TestLookupContextResolver(t *testing.T) {
	r := LookupContextResolver("git")
	if r == nil {
		t.Fatal("expected git resolver")
	}
	if r.Name() != "git" {
		t.Errorf("got name %q", r.Name())
	}
}

func TestLookupContextResolver_unknown(t *testing.T) {
	r := LookupContextResolver("nonexistent")
	if r != nil {
		t.Errorf("expected nil for unknown resolver")
	}
}
