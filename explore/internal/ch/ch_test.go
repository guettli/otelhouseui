package ch

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormaliseValue_timeToRFC3339(t *testing.T) {
	ts := time.Date(2026, 7, 10, 12, 34, 56, 0, time.UTC)
	got := normaliseValue(ts)
	if got != "2026-07-10T12:34:56Z" {
		t.Errorf("normaliseValue(time.Time) = %v, want RFC3339 string", got)
	}
}

func TestNormaliseValue_pointerTimeNil(t *testing.T) {
	var p *time.Time
	if got := normaliseValue(p); got != nil {
		t.Errorf("normaliseValue(nil *time.Time) = %v, want nil", got)
	}
}

func TestNormaliseValue_passesThroughScalars(t *testing.T) {
	cases := []any{"str", int64(7), float64(3.14), uint32(1), true, nil}
	for _, in := range cases {
		if out := normaliseValue(in); out != in {
			t.Errorf("normaliseValue(%v) = %v, want passthrough", in, out)
		}
	}
}

func TestNormaliseRow_marshalsAsJSON(t *testing.T) {
	ts := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	row := normaliseRow([]any{ts, "svc-a", float64(1.5)})
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `["2026-07-10T00:00:00Z","svc-a",1.5]`
	if string(b) != want {
		t.Errorf("row json = %s, want %s", b, want)
	}
}
