package store

import (
	"context"
	"errors"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestInsertGetListUpdateDelete(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	q := &SavedQuery{
		Name:        "q1",
		Description: "hello",
		SQLTemplate: "SELECT 1",
		Params: []Param{
			{Name: "n", Type: "UInt32", Label: "N", Widget: "number", Default: 7},
		},
		DefaultViz: "line",
	}
	if err := s.Insert(ctx, q); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if q.ID == 0 {
		t.Fatal("expected id to be set")
	}

	got, err := s.Get(ctx, q.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "q1" || got.SQLTemplate != "SELECT 1" || got.DefaultViz != "line" {
		t.Errorf("Get returned %+v", got)
	}
	if len(got.Params) != 1 || got.Params[0].Name != "n" || got.Params[0].Type != "UInt32" {
		t.Errorf("params round-trip failed: %+v", got.Params)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1", len(list))
	}

	q.Description = "updated"
	if err := s.Update(ctx, q); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(ctx, q.ID)
	if got.Description != "updated" {
		t.Errorf("Update did not persist description")
	}

	if err := s.Delete(ctx, q.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, q.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}
}

func TestInsertDuplicateName(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	q := &SavedQuery{Name: "dup", SQLTemplate: "SELECT 1"}
	if err := s.Insert(ctx, q); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	q2 := &SavedQuery{Name: "dup", SQLTemplate: "SELECT 2"}
	if err := s.Insert(ctx, q2); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("Insert dup: got %v, want ErrDuplicateName", err)
	}
}

func TestDefaultViz_backfilled(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	q := &SavedQuery{Name: "no-viz", SQLTemplate: "SELECT 1"}
	if err := s.Insert(ctx, q); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, _ := s.Get(ctx, q.ID)
	if got.DefaultViz != "auto" {
		t.Errorf("DefaultViz = %q, want 'auto'", got.DefaultViz)
	}
}
