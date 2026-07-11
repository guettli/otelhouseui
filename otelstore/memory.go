package otelstore

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore keeps every span and log record in process memory. It is the
// test fallback used when no ClickHouse backend is configured.
//
// WriteSpans and WriteLogs are concrete helpers tests use to seed fixture
// data — they are deliberately not on the Store interface, because production
// writes go through the OpenTelemetry Collector and are never this package's
// responsibility.
type MemoryStore struct {
	mu    sync.Mutex
	spans map[string]Span // keyed by SpanID
	logs  []LogRecord
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{spans: make(map[string]Span)}
}

// WriteSpans seeds spans for tests. Spans with the same SpanID overwrite
// earlier rows so a test can iterate on a fixture without rebuilding it.
func (m *MemoryStore) WriteSpans(_ context.Context, spans []Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range spans {
		m.spans[s.SpanID] = s
	}
	return nil
}

// WriteLogs seeds log records for tests.
func (m *MemoryStore) WriteLogs(_ context.Context, logs []LogRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logs...)
	return nil
}

// GetTrace implements Store.
func (m *MemoryStore) GetTrace(_ context.Context, traceID string) (Trace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := Trace{TraceID: traceID}
	for _, s := range m.spans {
		if s.TraceID == traceID {
			out.Spans = append(out.Spans, s)
		}
	}
	if len(out.Spans) == 0 {
		return Trace{}, ErrNotFound
	}
	for _, l := range m.logs {
		if l.TraceID == traceID {
			out.Logs = append(out.Logs, l)
		}
	}
	sort.Slice(out.Spans, func(i, j int) bool { return out.Spans[i].StartTime.Before(out.Spans[j].StartTime) })
	sort.Slice(out.Logs, func(i, j int) bool { return out.Logs[i].Time.Before(out.Logs[j].Time) })
	return out, nil
}

// ListTraces implements Store. Iterates every span once; acceptable because
// MemoryStore is only used in tests.
func (m *MemoryStore) ListTraces(_ context.Context, limit int) ([]TraceSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var roots []Span
	for _, s := range m.spans {
		if s.IsRoot() {
			roots = append(roots, s)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].StartTime.After(roots[j].StartTime) })
	if limit > 0 && len(roots) > limit {
		roots = roots[:limit]
	}
	out := make([]TraceSummary, 0, len(roots))
	for _, r := range roots {
		out = append(out, TraceSummary{
			TraceID:     r.TraceID,
			ServiceName: r.ServiceName,
			Name:        r.Name,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
			StatusCode:  r.StatusCode,
		})
	}
	return out, nil
}

// Close implements Store.
func (m *MemoryStore) Close() error { return nil }
