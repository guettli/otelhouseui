package starters

import (
	"context"
	"testing"

	"github.com/guettli/otelhouseui/internal/store"
)

func TestSeed_idempotent(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	for range 3 {
		if err := Seed(ctx, s); err != nil {
			t.Fatalf("Seed: %v", err)
		}
	}
	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != len(All) {
		t.Fatalf("List len = %d, want %d (seed inserted duplicates?)", len(list), len(All))
	}
}
