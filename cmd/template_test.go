package cmd

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

func createTestTemplate(t *testing.T, name, content string) storage.Template {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	now := time.Now().UTC()
	tmpl := storage.Template{
		ID:        id,
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	return tmpl
}

func TestTemplateListEmpty(t *testing.T) {
	setupTestEnv(t)

	templates, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}

func TestTemplateCreateAndList(t *testing.T) {
	setupTestEnv(t)

	createTestTemplate(t, "daily", "# Daily Entry")

	templates, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Name != "daily" {
		t.Errorf("name = %q, want %q", templates[0].Name, "daily")
	}
}

func TestTemplateShow(t *testing.T) {
	setupTestEnv(t)

	tmpl := createTestTemplate(t, "daily", "# Daily Entry")

	got, err := store.GetTemplateByName("daily")
	if err != nil {
		t.Fatalf("GetTemplateByName: %v", err)
	}
	if got.ID != tmpl.ID {
		t.Errorf("ID = %q, want %q", got.ID, tmpl.ID)
	}
	if got.Content != "# Daily Entry" {
		t.Errorf("content = %q, want %q", got.Content, "# Daily Entry")
	}
}

func TestTemplateShowNotFound(t *testing.T) {
	setupTestEnv(t)

	_, err := store.GetTemplateByName("nonexistent")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTemplateUpdate(t *testing.T) {
	setupTestEnv(t)

	tmpl := createTestTemplate(t, "daily", "# Daily Entry")

	updated, err := store.UpdateTemplate(tmpl.ID, "daily", "# Updated Daily Entry", nil)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Content != "# Updated Daily Entry" {
		t.Errorf("content = %q, want %q", updated.Content, "# Updated Daily Entry")
	}
}

func TestTemplateDelete(t *testing.T) {
	setupTestEnv(t)

	tmpl := createTestTemplate(t, "daily", "# Daily Entry")

	if err := store.DeleteTemplate(tmpl.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	_, err := store.GetTemplateByName("daily")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
