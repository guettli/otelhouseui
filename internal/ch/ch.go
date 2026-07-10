// Package ch is the thin ClickHouse client used by the HTTP API.
//
// Queries run against a single, read-only ClickHouse user. Execution limits
// live on that user's server-side profile — this package never re-imposes them
// in Go — so any resource cap violation surfaces as a native ClickHouse error
// with an actionable message.
package ch

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Column describes one column of a query result.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Result is the JSON-serialisable shape returned by Query.
type Result struct {
	Columns   []Column `json:"columns"`
	Rows      [][]any  `json:"rows"`
	ElapsedMs int64    `json:"elapsed_ms"`
}

// Client owns the ClickHouse connection.
type Client struct {
	conn driver.Conn
}

// Open dials ClickHouse using a clickhouse-go DSN and pings once.
func Open(ctx context.Context, dsn string) (*Client, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Ping checks liveness of the ClickHouse connection.
func (c *Client) Ping(ctx context.Context) error { return c.conn.Ping(ctx) }

// Close releases the underlying connection.
func (c *Client) Close() error { return c.conn.Close() }

// Query executes sql with named parameters (bound via ClickHouse native
// {name:Type} placeholders — never string interpolation) and returns a
// JSON-friendly result. Time columns are normalised to RFC3339Nano strings so
// the wire representation is stable regardless of the client's location.
func (c *Client) Query(ctx context.Context, sql string, params map[string]any) (*Result, error) {
	started := time.Now()

	args := make([]any, 0, len(params))
	for k, v := range params {
		args = append(args, clickhouse.Named(k, v))
	}

	rows, err := c.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	types := rows.ColumnTypes()
	cols := make([]Column, len(types))
	for i, ct := range types {
		cols[i] = Column{Name: ct.Name(), Type: ct.DatabaseTypeName()}
	}

	var out [][]any
	for rows.Next() {
		vals := make([]any, len(types))
		ptrs := make([]any, len(types))
		for i := range types {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		out = append(out, normaliseRow(vals))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Result{
		Columns:   cols,
		Rows:      out,
		ElapsedMs: time.Since(started).Milliseconds(),
	}, nil
}

// normaliseRow converts values into JSON-friendly forms: time.Time becomes an
// RFC3339Nano string so charts and grids don't need to guess the layout.
func normaliseRow(vals []any) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = normaliseValue(v)
	}
	return out
}

func normaliseValue(v any) any {
	switch x := v.(type) {
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case *time.Time:
		if x == nil {
			return nil
		}
		return x.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}
