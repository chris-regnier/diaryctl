package entry

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEntryContextRefJSON(t *testing.T) {
	e := Entry{
		ID:        "abc12345",
		Content:   "test",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Contexts: []ContextRef{
			{ContextID: "ctx00001", ContextName: "feature/auth"},
		},
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Contexts) != 1 || got.Contexts[0].ContextName != "feature/auth" {
		t.Errorf("got contexts %v", got.Contexts)
	}
}

func TestNew(t *testing.T) {
	t.Run("creates entry with valid content", func(t *testing.T) {
		e, err := New("hello world", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Content != "hello world" {
			t.Errorf("content = %q, want %q", e.Content, "hello world")
		}
		if err := ValidateID(e.ID); err != nil {
			t.Errorf("invalid ID: %v", err)
		}
		if e.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if e.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		e, err := New("  trimmed  ", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Content != "trimmed" {
			t.Errorf("content = %q, want %q", e.Content, "trimmed")
		}
	})

	t.Run("attaches template refs", func(t *testing.T) {
		refs := []TemplateRef{{TemplateID: "t1", TemplateName: "standup"}}
		e, err := New("content", refs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(e.Templates) != 1 || e.Templates[0].TemplateName != "standup" {
			t.Errorf("templates = %v, want standup ref", e.Templates)
		}
	})

	t.Run("rejects empty content", func(t *testing.T) {
		_, err := New("", nil)
		if err == nil {
			t.Error("expected error for empty content")
		}
	})

	t.Run("rejects whitespace-only content", func(t *testing.T) {
		_, err := New("   \n\t  ", nil)
		if err == nil {
			t.Error("expected error for whitespace-only content")
		}
	})
}

func TestValidateContextName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "main", false},
		{"valid with hyphen", "feature-branch", false},
		{"valid with underscore", "feature_branch", false},
		{"valid with slash", "feature/auth", false},
		{"valid mixed case", "Feature/Auth", false},
		{"valid complex", "feature/auth-v2_final", false},
		{"invalid empty", "", true},
		{"invalid starts with slash", "/feature", true},
		{"invalid starts with hyphen", "-feature", true},
		{"invalid starts with underscore", "_feature", true},
		{"invalid special char", "feature@branch", true},
		{"invalid space", "feature branch", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContextName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContextName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
