package cmd

import (
	"testing"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

func setupTestStore(t *testing.T) storage.Storage {
	t.Helper()
	dir := t.TempDir()
	s, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("creating test storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func setupTestEnv(t *testing.T) {
	t.Helper()
	store = setupTestStore(t)
	appConfig = &config.Config{}
	jsonOutput = false
}
