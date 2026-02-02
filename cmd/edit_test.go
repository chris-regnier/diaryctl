package cmd

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	tmpl "github.com/chris-regnier/diaryctl/internal/template"
)

func TestEditContentUpdate(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "original", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	updated, err := s.Update(id, "edited content", nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Content != "edited content" {
		t.Errorf("content = %q, want %q", updated.Content, "edited content")
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("updated_at should advance")
	}
}

func TestEditNotFound(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.Get("nonexist")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEditWithTemplateAttribution(t *testing.T) {
	setupTestEnv(t)

	// Create template and entry
	tpl := createTestTemplate(t, "prompts", "## Prompts\n- Q1?")
	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "Original content", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Compose template
	_, refs, err := tmpl.Compose(store, []string{"prompts"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	// Update with template refs
	updated, err := store.Update(id, "Original content\n\n## Prompts\n- Q1?", refs)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Templates) != 1 || updated.Templates[0].TemplateID != tpl.ID {
		t.Errorf("expected 1 template ref, got %v", updated.Templates)
	}
}

func TestEditWithTemplateDeduplicate(t *testing.T) {
	setupTestEnv(t)

	// Create template and entry with existing ref
	tpl := createTestTemplate(t, "prompts", "## Prompts")
	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{
		ID:        id,
		Content:   "Content",
		CreatedAt: now,
		UpdatedAt: now,
		Templates: []entry.TemplateRef{
			{TemplateID: tpl.ID, TemplateName: "prompts"},
		},
	}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Simulate edit --template prompts: compose same template again
	_, newRefs, _ := tmpl.Compose(store, []string{"prompts"})

	// Merge: existing + new, deduplicated
	seen := make(map[string]bool)
	var merged []entry.TemplateRef
	for _, ref := range e.Templates {
		seen[ref.TemplateID] = true
		merged = append(merged, ref)
	}
	for _, ref := range newRefs {
		if !seen[ref.TemplateID] {
			merged = append(merged, ref)
		}
	}

	updated, err := store.Update(id, "Content\n\n## Prompts", merged)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Should have exactly 1 ref, not 2
	if len(updated.Templates) != 1 {
		t.Errorf("expected 1 template ref (deduplicated), got %d", len(updated.Templates))
	}
}

func TestEditPreservesExistingTemplateRefs(t *testing.T) {
	setupTestEnv(t)

	// Create template and entry with existing ref
	tpl := createTestTemplate(t, "daily", "# Daily")
	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{
		ID:        id,
		Content:   "Content",
		CreatedAt: now,
		UpdatedAt: now,
		Templates: []entry.TemplateRef{
			{TemplateID: tpl.ID, TemplateName: "daily"},
		},
	}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update content without template flag (nil preserves)
	updated, err := store.Update(id, "Updated content", nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Templates) != 1 || updated.Templates[0].TemplateName != "daily" {
		t.Errorf("expected preserved template ref 'daily', got %v", updated.Templates)
	}
}
