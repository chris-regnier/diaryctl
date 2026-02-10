// Package day provides the Day data structure for organizing diary blocks.
// Days are the primary organizing unit (a "canvas") for diary entries.
package day

import (
	"sort"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
)

// Day represents a single day in the diary, acting as a canvas for blocks.
// Each day is identified by its Date (normalized to midnight local time)
// and contains an ordered list of blocks.
type Day struct {
	// Date is the normalized date (midnight local time) for this day
	Date time.Time

	// Blocks is the list of blocks for this day, ordered by Block.CreatedAt
	Blocks []block.Block

	// CreatedAt is when the first block was added to this day
	CreatedAt time.Time

	// UpdatedAt is when the day was last modified (block added, removed, or updated)
	UpdatedAt time.Time
}

// NormalizeDate normalizes a time.Time to midnight (00:00:00) in the local timezone.
// This ensures that all times for a given date are normalized to the same value,
// making date comparisons and lookups consistent.
//
// Example:
//
//	input:  2024-01-15 14:30:45.123456789
//	output: 2024-01-15 00:00:00.0
func NormalizeDate(t time.Time) time.Time {
	// Extract year, month, day and reconstruct at midnight in local timezone
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

// AddBlock adds a block to the day, maintains sort order by Block.CreatedAt,
// and updates the day's timestamps.
//
// The method ensures:
//   - Blocks remain sorted by CreatedAt (ascending)
//   - CreatedAt is set on first block addition
//   - UpdatedAt is always updated
func (d *Day) AddBlock(b block.Block) {
	// Set CreatedAt if this is the first block
	if len(d.Blocks) == 0 {
		d.CreatedAt = time.Now()
	}

	// Add the block
	d.Blocks = append(d.Blocks, b)

	// Sort blocks by CreatedAt
	d.SortBlocks()

	// Update the day's UpdatedAt timestamp
	d.UpdatedAt = time.Now()
}

// SortBlocks sorts the day's blocks in ascending order by Block.CreatedAt.
// This ensures that blocks are always displayed in chronological order.
func (d *Day) SortBlocks() {
	sort.Slice(d.Blocks, func(i, j int) bool {
		return d.Blocks[i].CreatedAt.Before(d.Blocks[j].CreatedAt)
	})
}

// FindBlock searches for a block by ID and returns its index.
// Returns -1 if the block is not found.
//
// This is useful for locating a block before updating or removing it.
func (d *Day) FindBlock(id string) int {
	for i, b := range d.Blocks {
		if b.ID == id {
			return i
		}
	}
	return -1
}

// RemoveBlock removes a block by ID and returns true if successful.
// Returns false if the block was not found.
//
// The method maintains the sorted order of remaining blocks.
func (d *Day) RemoveBlock(id string) bool {
	index := d.FindBlock(id)
	if index == -1 {
		return false
	}

	// Remove the block at the found index
	d.Blocks = append(d.Blocks[:index], d.Blocks[index+1:]...)

	// Update the day's UpdatedAt timestamp
	d.UpdatedAt = time.Now()

	return true
}
