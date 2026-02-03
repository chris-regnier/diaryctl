package datetime

import (
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	p := &Provider{now: func() time.Time {
		return time.Date(2026, 2, 2, 10, 30, 0, 0, time.Local)
	}}
	got, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "# Monday, February 2, 2026"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestName(t *testing.T) {
	p := New()
	if p.Name() != "datetime" {
		t.Errorf("got name %q", p.Name())
	}
}
