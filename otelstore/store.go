// Package otelstore is a read-only, typed client for OpenTelemetry traces and
// logs stored in ClickHouse by the upstream opentelemetry-collector-contrib
// clickhouseexporter (tables otel_traces and otel_logs, stock schema).
//
// The package exposes one interface — Store — and two implementations:
//
//   - ClickHouseStore reads the exporter's tables. Writes are owned by the
//     Collector; this package never issues DDL or INSERT.
//   - MemoryStore holds everything in memory. It is the test fallback and
//     keeps WriteSpans / WriteLogs as concrete helpers tests use to seed
//     fixture data — those helpers are deliberately not on the interface.
//
// # Tenancy
//
// This package is deliberately tenant-blind. In a multi-tenant deployment the
// isolation boundary is the ClickHouse identity in the DSN: the otelhouse
// gateway stamps ResourceAttributes['tenant'] on every record at write time,
// and reads are constrained by a ClickHouse row policy bound to a per-tenant
// read-only user. Callers therefore pass a DSN that is already scoped to one
// tenant, and no query here adds a tenant predicate. Do not add one: a filter
// in Go would look like a security boundary without being one.
package otelstore

import (
	"context"
	"errors"
	"time"
)

// Span is one OpenTelemetry span, flattened so each row carries enough resource
// context to be rendered on its own.
type Span struct {
	TraceID            string
	SpanID             string
	ParentSpanID       string
	Name               string
	Kind               int32
	StartTime          time.Time
	EndTime            time.Time
	StatusCode         int32
	StatusMessage      string
	Attributes         map[string]string
	ResourceAttributes map[string]string
	ServiceName        string
}

// IsRoot reports whether s is the root of its trace. The OTLP convention is
// that a root span has an empty (or all-zero) parent span id.
func (s Span) IsRoot() bool { return s.ParentSpanID == "" }

// Duration returns the wall-clock duration covered by the span.
func (s Span) Duration() time.Duration { return s.EndTime.Sub(s.StartTime) }

// LogRecord is one OpenTelemetry log record, flattened the same way Span is.
//
// SpanID may be empty (record not associated with a span); TraceID is also
// optional, but the viewer only renders logs that carry a trace id so the
// page always has somewhere to attach them.
type LogRecord struct {
	TraceID            string
	SpanID             string
	Time               time.Time
	SeverityNumber     int32
	SeverityText       string
	Body               string
	Attributes         map[string]string
	ResourceAttributes map[string]string
	ServiceName        string
}

// Trace is the materialised view the HTML viewer renders: every span plus
// every log record that share the trace id.
type Trace struct {
	TraceID string
	Spans   []Span
	Logs    []LogRecord
}

// Root returns the root span of the trace, or false if none was found. A
// trace can briefly have no root in-flight; once the Collector has written
// the root span, this returns true.
func (t Trace) Root() (Span, bool) {
	for _, s := range t.Spans {
		if s.IsRoot() {
			return s, true
		}
	}
	return Span{}, false
}

// TraceSummary is the per-trace row rendered on the viewer index page. The
// fields come from the root span; if a trace has no root yet (still
// in-flight) the summary is omitted from listings.
type TraceSummary struct {
	TraceID     string
	ServiceName string
	Name        string
	StartTime   time.Time
	EndTime     time.Time
	StatusCode  int32
}

// Store is the read interface over a trace store. Both MemoryStore and
// ClickHouseStore implement it. Writes are out of scope: the Collector lands
// data into ClickHouse directly, and MemoryStore exposes its seed helpers as
// concrete methods so they stay out of the interface.
type Store interface {
	// GetTrace returns every span and log row sharing traceID. The slices
	// are sorted by start / time so the viewer renders them in causal order.
	GetTrace(ctx context.Context, traceID string) (Trace, error)

	// ListTraces returns the most recent traces (by root-span start time),
	// newest first. Traces without a root span are skipped.
	ListTraces(ctx context.Context, limit int) ([]TraceSummary, error)

	// Close releases any underlying resources. Safe to call multiple times.
	Close() error
}

// ErrNotFound is returned by GetTrace when the trace id is unknown.
var ErrNotFound = errors.New("otelstore: trace not found")
