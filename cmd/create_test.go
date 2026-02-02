package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	tmpl "github.com/chris-regnier/diaryctl/internal/template"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

func TestCreateInline(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{ID: id, Content: "Test inline entry", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "Test inline entry" {
		t.Errorf("content = %q, want %q", got.Content, "Test inline entry")
	}
}

func TestCreateEmptyContentRejected(t *testing.T) {
	err := entry.ValidateContent("   ")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCreateJSONOutput(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{ID: id, Content: "JSON test", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var buf bytes.Buffer
	ui.FormatJSON(&buf, e)

	var result entry.Entry
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if result.ID != id {
		t.Errorf("id = %q, want %q", result.ID, id)
	}
}

func TestCreateWithTemplateAttribution(t *testing.T) {
	setupTestEnv(t)

	// Create a template
	tpl := createTestTemplate(t, "daily", "# Daily Entry")

	// Compose template content
	content, refs, err := tmpl.Compose(store, []string{"daily"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if content != "# Daily Entry" {
		t.Errorf("content = %q", content)
	}

	// Create an entry with template attribution
	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{
		ID:        id,
		Content:   strings.TrimSpace(content),
		CreatedAt: now,
		UpdatedAt: now,
		Templates: refs,
	}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify attribution is stored
	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Templates) != 1 {
		t.Fatalf("expected 1 template ref, got %d", len(got.Templates))
	}
	if got.Templates[0].TemplateID != tpl.ID || got.Templates[0].TemplateName != "daily" {
		t.Errorf("template ref = %+v", got.Templates[0])
	}
}

func TestCreateWithMultipleTemplates(t *testing.T) {
	setupTestEnv(t)

	createTestTemplate(t, "daily", "# Daily")
	createTestTemplate(t, "prompts", "## Prompts\n- Q1?")

	content, refs, err := tmpl.Compose(store, []string{"daily", "prompts"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if !strings.Contains(content, "# Daily") || !strings.Contains(content, "## Prompts") {
		t.Errorf("expected composed content, got %q", content)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 refs, got %d", len(refs))
	}
}

func TestCreateWithNonexistentTemplate(t *testing.T) {
	setupTestEnv(t)

	_, _, err := tmpl.Compose(store, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestCreateInlineBypassesTemplates(t *testing.T) {
	setupTestEnv(t)

	createTestTemplate(t, "daily", "# Daily Entry")

	// When inline content is provided, no template refs should be set
	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{
		ID:        id,
		Content:   "Inline content",
		CreatedAt: now,
		UpdatedAt: now,
		// No Templates field — inline bypasses templates
	}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Templates) != 0 {
		t.Errorf("expected 0 template refs for inline content, got %d", len(got.Templates))
	}
}

func TestCreateDefaultTemplateConfig(t *testing.T) {
	setupTestEnv(t)
	appConfig.DefaultTemplate = "daily"

	createTestTemplate(t, "daily", "# Daily Entry")

	// Resolve default template from config
	names := tmpl.ParseNames(appConfig.DefaultTemplate)
	content, refs, err := tmpl.Compose(store, names)
	if err != nil {
		t.Fatalf("Compose with default: %v", err)
	}
	if content != "# Daily Entry" {
		t.Errorf("content = %q", content)
	}
	if len(refs) != 1 {
		t.Errorf("expected 1 ref, got %d", len(refs))
	}
}

func TestCreateDefaultTemplateMisconfigured(t *testing.T) {
	setupTestEnv(t)
	appConfig.DefaultTemplate = "nonexistent"

	// Should return error (the command handles graceful fallback)
	names := tmpl.ParseNames(appConfig.DefaultTemplate)
	_, _, err := tmpl.Compose(store, names)
	if err == nil {
		t.Fatal("expected error for nonexistent default template")
	}
	if !strings.Contains(err.Error(), storage.ErrNotFound.Error()) {
		t.Errorf("expected ErrNotFound in error, got: %v", err)
	}
}
