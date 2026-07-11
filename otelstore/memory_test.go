package otelstore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreWriteReadTrace(t *testing.T) {
	m := NewMemoryStore()
	t0 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	root := Span{
		TraceID:     "trace1",
		SpanID:      "root",
		Name:        "build",
		StartTime:   t0,
		EndTime:     t0.Add(2 * time.Second),
		ServiceName: "dagger",
		StatusCode:  1,
	}
	child := Span{
		TraceID:      "trace1",
		SpanID:       "child",
		ParentSpanID: "root",
		Name:         "go test",
		StartTime:    t0.Add(100 * time.Millisecond),
		EndTime:      t0.Add(1 * time.Second),
		ServiceName:  "dagger",
	}
	if err := m.WriteSpans(context.Background(), []Span{child, root}); err != nil {
		t.Fatalf("WriteSpans: %v", err)
	}
	if err := m.WriteLogs(context.Background(), []LogRecord{{
		TraceID:      "trace1",
		SpanID:       "child",
		Time:         t0.Add(200 * time.Millisecond),
		SeverityText: "INFO",
		Body:         "hello",
	}}); err != nil {
		t.Fatalf("WriteLogs: %v", err)
	}
	got, err := m.GetTrace(context.Background(), "trace1")
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if len(got.Spans) != 2 {
		t.Fatalf("want 2 spans, got %d", len(got.Spans))
	}
	if !got.Spans[0].StartTime.Equal(root.StartTime) {
		t.Errorf("spans not sorted: first start %v, want %v", got.Spans[0].StartTime, root.StartTime)
	}
	r, ok := got.Root()
	if !ok || r.SpanID != "root" {
		t.Errorf("Root() = (%+v, %v), want root span", r, ok)
	}
	if len(got.Logs) != 1 || got.Logs[0].Body != "hello" {
		t.Errorf("logs not persisted, got %+v", got.Logs)
	}
}

func TestMemoryStoreUpsertSpan(t *testing.T) {
	m := NewMemoryStore()
	s := Span{TraceID: "t", SpanID: "s", Name: "v1", StartTime: time.Now()}
	if err := m.WriteSpans(context.Background(), []Span{s}); err != nil {
		t.Fatal(err)
	}
	s.Name = "v2"
	if err := m.WriteSpans(context.Background(), []Span{s}); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetTrace(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Spans) != 1 || got.Spans[0].Name != "v2" {
		t.Errorf("upsert failed: %+v", got.Spans)
	}
}

func TestMemoryStoreListTracesSkipsRootless(t *testing.T) {
	m := NewMemoryStore()
	t0 := time.Now()
	// Trace 'a' has only a child span (no root yet).
	if err := m.WriteSpans(context.Background(), []Span{{
		TraceID: "a", SpanID: "c", ParentSpanID: "missing",
		Name: "child", StartTime: t0, EndTime: t0.Add(time.Second),
	}}); err != nil {
		t.Fatal(err)
	}
	// Trace 'b' has a root.
	if err := m.WriteSpans(context.Background(), []Span{{
		TraceID: "b", SpanID: "r",
		Name: "root", StartTime: t0.Add(2 * time.Second), EndTime: t0.Add(3 * time.Second),
	}}); err != nil {
		t.Fatal(err)
	}
	got, err := m.ListTraces(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TraceID != "b" {
		t.Errorf("ListTraces = %+v, want only [b]", got)
	}
}

func TestMemoryStoreGetTraceNotFound(t *testing.T) {
	m := NewMemoryStore()
	if _, err := m.GetTrace(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
