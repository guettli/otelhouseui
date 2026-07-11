package httpapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/guettli/otelhouseview/internal/store"
)

// bindParams converts a raw request body (arbitrary JSON) into typed Go values
// keyed by parameter name, ready to hand to clickhouse-go's Named binder.
// Missing values fall back to the saved query's default.
func bindParams(defs []store.Param, in map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(defs))
	for _, d := range defs {
		raw, ok := in[d.Name]
		if !ok || raw == nil {
			if d.Default == nil {
				return nil, fmt.Errorf("missing required parameter %q", d.Name)
			}
			raw = d.Default
		}
		v, err := convertParam(d.Type, raw)
		if err != nil {
			return nil, fmt.Errorf("param %q: %w", d.Name, err)
		}
		out[d.Name] = v
	}
	return out, nil
}

// SupportedParamType reports whether type name is one this v1 slice binds.
// Basic ClickHouse scalar types only — enough for the metrics starter query
// and typical dashboards. Extend as future starters demand.
func SupportedParamType(t string) bool {
	switch t {
	case "String",
		"Int8", "Int16", "Int32", "Int64",
		"UInt8", "UInt16", "UInt32", "UInt64",
		"Float32", "Float64",
		"Date", "DateTime":
		return true
	}
	return false
}

func convertParam(t string, raw any) (any, error) {
	switch t {
	case "String":
		return asString(raw)
	case "Int8":
		v, err := asInt64(raw)
		return int8(v), err
	case "Int16":
		v, err := asInt64(raw)
		return int16(v), err
	case "Int32":
		v, err := asInt64(raw)
		return int32(v), err
	case "Int64":
		return asInt64(raw)
	case "UInt8":
		v, err := asUint64(raw)
		return uint8(v), err
	case "UInt16":
		v, err := asUint64(raw)
		return uint16(v), err
	case "UInt32":
		v, err := asUint64(raw)
		return uint32(v), err
	case "UInt64":
		return asUint64(raw)
	case "Float32":
		v, err := asFloat64(raw)
		return float32(v), err
	case "Float64":
		return asFloat64(raw)
	case "Date":
		return asTime(raw, true)
	case "DateTime":
		return asTime(raw, false)
	}
	return nil, fmt.Errorf("unsupported type %q", t)
}

func asString(raw any) (string, error) {
	switch v := raw.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	}
	return "", fmt.Errorf("expected string, got %T", raw)
}

func asInt64(raw any) (int64, error) {
	switch v := raw.(type) {
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	}
	return 0, fmt.Errorf("expected integer, got %T", raw)
}

func asUint64(raw any) (uint64, error) {
	switch v := raw.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("expected unsigned integer, got %v", v)
		}
		return uint64(v), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	}
	return 0, fmt.Errorf("expected unsigned integer, got %T", raw)
}

func asFloat64(raw any) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	}
	return 0, fmt.Errorf("expected number, got %T", raw)
}

// asTime accepts common wire formats: RFC3339 (with or without zone), the
// classic "2006-01-02 15:04:05" ClickHouse literal, or a date-only string.
// For a Date parameter (`dateOnly`) it truncates to midnight UTC.
func asTime(raw any, dateOnly bool) (time.Time, error) {
	s, err := asString(raw)
	if err != nil {
		return time.Time{}, err
	}
	s = strings.TrimSpace(s)

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			if dateOnly {
				y, m, d := t.Date()
				return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time %q", s)
}
