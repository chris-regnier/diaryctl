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
