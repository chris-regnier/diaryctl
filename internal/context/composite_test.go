package context_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	icontext "github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

func TestCompositeProvider_Search(t *testing.T) {
	dir := t.TempDir()
	store, _ := markdown.New(dir)
	defer store.Close()

	_ = store.Create(entry.Entry{
		ID:        "search01",
		Content:   "Learning about MCP protocols",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	provider, err := icontext.NewCompositeProvider(store)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	results, err := provider.Search(context.Background(), "MCP", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "search01" {
		t.Errorf("expected search01, got %s", results[0].ID)
	}
}

func TestCompositeProvider_Filter(t *testing.T) {
	dir := t.TempDir()
	store, _ := markdown.New(dir)
	defer store.Close()

	// Use specific dates to avoid timezone issues
	today := time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local)
	yesterday := time.Date(2026, 1, 14, 12, 0, 0, 0, time.Local)

	_ = store.Create(entry.Entry{
		ID:        "today001",
		Content:   "Today's entry",
		CreatedAt: today,
		UpdatedAt: today,
	})
	_ = store.Create(entry.Entry{
		ID:        "yester01",
		Content:   "Yesterday's entry",
		CreatedAt: yesterday,
		UpdatedAt: yesterday,
	})

	provider, err := icontext.NewCompositeProvider(store)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Filter for today only
	results, err := provider.Filter(context.Background(), storage.ListOptions{
		StartDate: &today,
		EndDate:   &today,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("filter failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result for today, got %d", len(results))
		for i, r := range results {
			t.Logf("result[%d]: ID=%s, Date=%s", i, r.ID, r.Date)
		}
	}
	if len(results) > 0 && results[0].ID != "today001" {
		t.Errorf("expected today001, got %s (date=%s)", results[0].ID, results[0].Date)
	}
}
