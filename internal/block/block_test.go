package block_test

import (
	"regexp"
	"testing"

	"github.com/chris-regnier/diaryctl/internal/block"
)

// TestNewID verifies that NewID generates valid 8-character IDs
// using the alphabet "abcdefghijklmnopqrstuvwxyz0123456789"
func TestNewID(t *testing.T) {
	idPattern := regexp.MustCompile(`^[a-z0-9]{8}$`)

	// Generate multiple IDs to check consistency
	for i := 0; i < 100; i++ {
		id := block.NewID()
		if !idPattern.MatchString(id) {
			t.Errorf("NewID() = %q, want 8-char lowercase alphanumeric", id)
		}
		if len(id) != 8 {
			t.Errorf("NewID() length = %d, want 8", len(id))
		}
	}

	// Check that IDs are unique (probabilistic test)
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := block.NewID()
		if ids[id] {
			t.Errorf("NewID() generated duplicate ID: %q", id)
		}
		ids[id] = true
	}
}

// TestValidateID verifies that ValidateID correctly validates ID format
func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "valid lowercase alphanumeric",
			id:      "abc12345",
			wantErr: false,
		},
		{
			name:    "valid all letters",
			id:      "abcdefgh",
			wantErr: false,
		},
		{
			name:    "valid all numbers",
			id:      "12345678",
			wantErr: false,
		},
		{
			name:    "invalid uppercase",
			id:      "ABC12345",
			wantErr: true,
		},
		{
			name:    "invalid too short",
			id:      "abc1234",
			wantErr: true,
		},
		{
			name:    "invalid too long",
			id:      "abc123456",
			wantErr: true,
		},
		{
			name:    "invalid special characters",
			id:      "abc-1234",
			wantErr: true,
		},
		{
			name:    "invalid empty",
			id:      "",
			wantErr: true,
		},
		{
			name:    "invalid with space",
			id:      "abc 1234",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := block.ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}
