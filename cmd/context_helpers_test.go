package cmd

import (
	"testing"
)

func TestBuildContentProviders(t *testing.T) {
	// Known providers
	providers := buildContentProviders([]string{"datetime", "git"})
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].Name() != "datetime" {
		t.Errorf("expected datetime, got %s", providers[0].Name())
	}
	if providers[1].Name() != "git" {
		t.Errorf("expected git, got %s", providers[1].Name())
	}
}

func TestBuildContentProvidersSkipsUnknown(t *testing.T) {
	providers := buildContentProviders([]string{"datetime", "nonexistent"})
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
}

func TestBuildContentProvidersEmpty(t *testing.T) {
	providers := buildContentProviders(nil)
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(providers))
	}
}

func TestBuildContextResolvers(t *testing.T) {
	resolvers := buildContextResolvers([]string{"git"})
	if len(resolvers) != 1 {
		t.Fatalf("expected 1 resolver, got %d", len(resolvers))
	}
	if resolvers[0].Name() != "git" {
		t.Errorf("expected git, got %s", resolvers[0].Name())
	}
}

func TestBuildContextResolversSkipsUnknown(t *testing.T) {
	resolvers := buildContextResolvers([]string{"nonexistent"})
	if len(resolvers) != 0 {
		t.Fatalf("expected 0 resolvers, got %d", len(resolvers))
	}
}
