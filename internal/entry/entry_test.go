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
