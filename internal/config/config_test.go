package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultTheme(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Theme.Preset != "default-dark" {
		t.Errorf("expected preset 'default-dark', got %q", cfg.Theme.Preset)
	}
	if cfg.Theme.MarkdownStyle != "" {
		t.Errorf("expected empty markdown_style (uses preset default), got %q", cfg.Theme.MarkdownStyle)
	}
}

func TestLoadThemeFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[theme]
preset = "default-light"
primary = "#FF0000"
markdown_style = "light"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Theme.Preset != "default-light" {
		t.Errorf("expected preset 'default-light', got %q", cfg.Theme.Preset)
	}
	if cfg.Theme.Primary != "#FF0000" {
		t.Errorf("expected primary '#FF0000', got %q", cfg.Theme.Primary)
	}
	if cfg.Theme.MarkdownStyle != "light" {
		t.Errorf("expected markdown_style 'light', got %q", cfg.Theme.MarkdownStyle)
	}
}
