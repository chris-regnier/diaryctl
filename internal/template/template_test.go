package template

import (
	"testing"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// mockStorage implements just the template methods needed for testing.
type mockStorage struct {
	templates map[string]storage.Template // keyed by name
}

func (m *mockStorage) GetTemplateByName(name string) (storage.Template, error) {
	t, ok := m.templates[name]
	if !ok {
		return storage.Template{}, storage.ErrNotFound
	}
	return t, nil
}

func TestComposeSingle(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{
		"daily": {ID: "abc12345", Name: "daily", Content: "# Daily Entry"},
	}}
	content, refs, err := Compose(ms, []string{"daily"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if content != "# Daily Entry" {
		t.Errorf("got content=%q", content)
	}
	if len(refs) != 1 || refs[0].TemplateName != "daily" {
		t.Errorf("got refs=%v", refs)
	}
}

func TestComposeMultiple(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{
		"daily":   {ID: "abc12345", Name: "daily", Content: "# Daily"},
		"prompts": {ID: "def67890", Name: "prompts", Content: "## Prompts\n- Q1?"},
	}}
	content, refs, err := Compose(ms, []string{"daily", "prompts"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	expected := "# Daily\n\n## Prompts\n- Q1?"
	if content != expected {
		t.Errorf("got content=%q, want %q", content, expected)
	}
	if len(refs) != 2 {
		t.Errorf("got %d refs, want 2", len(refs))
	}
}

func TestComposeNotFound(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{}}
	_, _, err := Compose(ms, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestComposeEmpty(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{}}
	content, refs, err := Compose(ms, []string{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if content != "" || len(refs) != 0 {
		t.Errorf("expected empty result, got content=%q refs=%v", content, refs)
	}
}

func TestParseNames(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"daily", []string{"daily"}},
		{"daily,prompts", []string{"daily", "prompts"}},
		{" daily , prompts ", []string{"daily", "prompts"}},
	}
	for _, tt := range tests {
		got := ParseNames(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseNames(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseNames(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// Ensure unused import doesn't fail
var _ entry.TemplateRef
