package context

import "testing"

func TestLoadManualContexts_noFile(t *testing.T) {
	dir := t.TempDir()
	names, err := LoadManualContexts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}
}

func TestSetAndLoadManualContexts(t *testing.T) {
	dir := t.TempDir()
	if err := SetManualContext(dir, "sprint:23"); err != nil {
		t.Fatalf("SetManualContext: %v", err)
	}
	if err := SetManualContext(dir, "project:auth"); err != nil {
		t.Fatalf("SetManualContext: %v", err)
	}
	names, err := LoadManualContexts(dir)
	if err != nil {
		t.Fatalf("LoadManualContexts: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2, got %d", len(names))
	}
}

func TestSetManualContext_idempotent(t *testing.T) {
	dir := t.TempDir()
	_ = SetManualContext(dir, "sprint:23")
	_ = SetManualContext(dir, "sprint:23")
	names, _ := LoadManualContexts(dir)
	if len(names) != 1 {
		t.Errorf("expected 1 after duplicate set, got %d", len(names))
	}
}

func TestUnsetManualContext(t *testing.T) {
	dir := t.TempDir()
	_ = SetManualContext(dir, "sprint:23")
	_ = SetManualContext(dir, "project:auth")
	if err := UnsetManualContext(dir, "sprint:23"); err != nil {
		t.Fatalf("UnsetManualContext: %v", err)
	}
	names, _ := LoadManualContexts(dir)
	if len(names) != 1 || names[0] != "project:auth" {
		t.Errorf("expected [project:auth], got %v", names)
	}
}

func TestUnsetManualContext_notSet(t *testing.T) {
	dir := t.TempDir()
	err := UnsetManualContext(dir, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
