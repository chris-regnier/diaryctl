package day_test

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
)

// TestNormalizeDate verifies that NormalizeDate correctly zeros out
// time components and preserves the date in local timezone
func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
		want  string // Expected date string in format "2006-01-02"
	}{
		{
			name:  "already at midnight",
			input: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local),
			want:  "2024-01-15",
		},
		{
			name:  "afternoon time",
			input: time.Date(2024, 1, 15, 14, 30, 45, 123456789, time.Local),
			want:  "2024-01-15",
		},
		{
			name:  "just before midnight",
			input: time.Date(2024, 1, 15, 23, 59, 59, 999999999, time.Local),
			want:  "2024-01-15",
		},
		{
			name:  "start of year",
			input: time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local),
			want:  "2024-01-01",
		},
		{
			name:  "end of year",
			input: time.Date(2024, 12, 31, 23, 59, 59, 0, time.Local),
			want:  "2024-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := day.NormalizeDate(tt.input)
			gotDate := got.Format("2006-01-02")
			if gotDate != tt.want {
				t.Errorf("NormalizeDate() = %s, want %s", gotDate, tt.want)
			}
			// Verify time is at midnight
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
				t.Errorf("NormalizeDate() time = %02d:%02d:%02d.%d, want 00:00:00.0",
					got.Hour(), got.Minute(), got.Second(), got.Nanosecond())
			}
			// Verify timezone is local
			if got.Location() != time.Local {
				t.Errorf("NormalizeDate() location = %v, want Local", got.Location())
			}
		})
	}
}

// TestDay_AddBlock verifies that AddBlock correctly adds blocks,
// maintains sorting, and updates timestamps
func TestDay_AddBlock(t *testing.T) {
	now := time.Now()
	d := day.Day{
		Date: day.NormalizeDate(now),
	}

	// Add first block
	b1 := block.Block{
		ID:        block.NewID(),
		Content:   "First block",
		CreatedAt: now,
		UpdatedAt: now,
	}
	d.AddBlock(b1)

	if len(d.Blocks) != 1 {
		t.Errorf("After AddBlock, len(Blocks) = %d, want 1", len(d.Blocks))
	}
	if d.Blocks[0].Content != "First block" {
		t.Errorf("Block content = %q, want %q", d.Blocks[0].Content, "First block")
	}
	if d.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set after first block")
	}
	if d.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after adding block")
	}

	// Add second block with earlier timestamp (should be inserted before first)
	earlier := now.Add(-1 * time.Hour)
	b2 := block.Block{
		ID:        block.NewID(),
		Content:   "Second block (earlier)",
		CreatedAt: earlier,
		UpdatedAt: earlier,
	}
	d.AddBlock(b2)

	if len(d.Blocks) != 2 {
		t.Errorf("After second AddBlock, len(Blocks) = %d, want 2", len(d.Blocks))
	}
	if d.Blocks[0].Content != "Second block (earlier)" {
		t.Errorf("First block content = %q, want %q", d.Blocks[0].Content, "Second block (earlier)")
	}
	if d.Blocks[1].Content != "First block" {
		t.Errorf("Second block content = %q, want %q", d.Blocks[1].Content, "First block")
	}

	// Add third block with latest timestamp (should be appended)
	later := now.Add(1 * time.Hour)
	b3 := block.Block{
		ID:        block.NewID(),
		Content:   "Third block (later)",
		CreatedAt: later,
		UpdatedAt: later,
	}
	d.AddBlock(b3)

	if len(d.Blocks) != 3 {
		t.Errorf("After third AddBlock, len(Blocks) = %d, want 3", len(d.Blocks))
	}
	if d.Blocks[2].Content != "Third block (later)" {
		t.Errorf("Third block content = %q, want %q", d.Blocks[2].Content, "Third block (later)")
	}

	// Verify blocks are sorted by CreatedAt
	if !d.Blocks[0].CreatedAt.Before(d.Blocks[1].CreatedAt) {
		t.Error("Blocks not sorted: block 0 should be before block 1")
	}
	if !d.Blocks[1].CreatedAt.Before(d.Blocks[2].CreatedAt) {
		t.Error("Blocks not sorted: block 1 should be before block 2")
	}
}

// TestDay_SortBlocks verifies that SortBlocks correctly orders blocks
// by CreatedAt timestamp in ascending order
func TestDay_SortBlocks(t *testing.T) {
	now := time.Now()
	d := day.Day{
		Date: day.NormalizeDate(now),
		Blocks: []block.Block{
			{
				ID:        "block003",
				Content:   "Latest",
				CreatedAt: now.Add(2 * time.Hour),
				UpdatedAt: now.Add(2 * time.Hour),
			},
			{
				ID:        "block001",
				Content:   "Earliest",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "block002",
				Content:   "Middle",
				CreatedAt: now.Add(1 * time.Hour),
				UpdatedAt: now.Add(1 * time.Hour),
			},
		},
	}

	// Sort blocks
	d.SortBlocks()

	// Verify order
	if d.Blocks[0].Content != "Earliest" {
		t.Errorf("After sort, Blocks[0].Content = %q, want %q", d.Blocks[0].Content, "Earliest")
	}
	if d.Blocks[1].Content != "Middle" {
		t.Errorf("After sort, Blocks[1].Content = %q, want %q", d.Blocks[1].Content, "Middle")
	}
	if d.Blocks[2].Content != "Latest" {
		t.Errorf("After sort, Blocks[2].Content = %q, want %q", d.Blocks[2].Content, "Latest")
	}

	// Verify timestamps are in ascending order
	for i := 0; i < len(d.Blocks)-1; i++ {
		if d.Blocks[i].CreatedAt.After(d.Blocks[i+1].CreatedAt) {
			t.Errorf("Blocks[%d].CreatedAt > Blocks[%d].CreatedAt, want ascending order", i, i+1)
		}
	}
}

// TestDay_FindBlock verifies that FindBlock correctly locates blocks by ID
func TestDay_FindBlock(t *testing.T) {
	now := time.Now()
	d := day.Day{
		Date: day.NormalizeDate(now),
		Blocks: []block.Block{
			{
				ID:        "abc12345",
				Content:   "First",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "def67890",
				Content:   "Second",
				CreatedAt: now.Add(1 * time.Hour),
				UpdatedAt: now.Add(1 * time.Hour),
			},
			{
				ID:        "ghi13579",
				Content:   "Third",
				CreatedAt: now.Add(2 * time.Hour),
				UpdatedAt: now.Add(2 * time.Hour),
			},
		},
	}

	tests := []struct {
		name      string
		id        string
		wantIndex int
	}{
		{
			name:      "find first block",
			id:        "abc12345",
			wantIndex: 0,
		},
		{
			name:      "find middle block",
			id:        "def67890",
			wantIndex: 1,
		},
		{
			name:      "find last block",
			id:        "ghi13579",
			wantIndex: 2,
		},
		{
			name:      "block not found",
			id:        "notfound",
			wantIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.FindBlock(tt.id)
			if got != tt.wantIndex {
				t.Errorf("FindBlock(%q) = %d, want %d", tt.id, got, tt.wantIndex)
			}
		})
	}
}

// TestDay_RemoveBlock verifies that RemoveBlock correctly removes blocks by ID
func TestDay_RemoveBlock(t *testing.T) {
	now := time.Now()
	d := day.Day{
		Date: day.NormalizeDate(now),
		Blocks: []block.Block{
			{
				ID:        "abc12345",
				Content:   "First",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "def67890",
				Content:   "Second",
				CreatedAt: now.Add(1 * time.Hour),
				UpdatedAt: now.Add(1 * time.Hour),
			},
			{
				ID:        "ghi13579",
				Content:   "Third",
				CreatedAt: now.Add(2 * time.Hour),
				UpdatedAt: now.Add(2 * time.Hour),
			},
		},
	}

	// Remove middle block
	removed := d.RemoveBlock("def67890")
	if !removed {
		t.Error("RemoveBlock returned false, want true")
	}
	if len(d.Blocks) != 2 {
		t.Errorf("After RemoveBlock, len(Blocks) = %d, want 2", len(d.Blocks))
	}
	if d.Blocks[0].ID != "abc12345" {
		t.Errorf("Blocks[0].ID = %q, want %q", d.Blocks[0].ID, "abc12345")
	}
	if d.Blocks[1].ID != "ghi13579" {
		t.Errorf("Blocks[1].ID = %q, want %q", d.Blocks[1].ID, "ghi13579")
	}

	// Try to remove non-existent block
	removed = d.RemoveBlock("notfound")
	if removed {
		t.Error("RemoveBlock returned true for non-existent block, want false")
	}
	if len(d.Blocks) != 2 {
		t.Errorf("After failed RemoveBlock, len(Blocks) = %d, want 2", len(d.Blocks))
	}
}
