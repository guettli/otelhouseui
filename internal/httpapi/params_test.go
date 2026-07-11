package httpapi

import (
	"testing"
	"time"

	"github.com/guettli/otelhouseview/internal/store"
)

func TestBindParams_typeCoercion(t *testing.T) {
	defs := []store.Param{
		{Name: "n", Type: "UInt32"},
		{Name: "s", Type: "String"},
		{Name: "f", Type: "Float64"},
		{Name: "t", Type: "DateTime"},
	}
	in := map[string]any{
		"n": float64(7), // JSON numbers decode as float64
		"s": "hello",
		"f": float64(1.5),
		"t": "2026-07-10 12:34:56",
	}
	out, err := bindParams(defs, in)
	if err != nil {
		t.Fatalf("bindParams: %v", err)
	}
	if v, ok := out["n"].(uint32); !ok || v != 7 {
		t.Errorf("n = %v (%T), want uint32(7)", out["n"], out["n"])
	}
	if v := out["s"]; v != "hello" {
		t.Errorf("s = %v", v)
	}
	if v := out["f"]; v != float64(1.5) {
		t.Errorf("f = %v", v)
	}
	ts, ok := out["t"].(time.Time)
	if !ok {
		t.Fatalf("t is not time.Time: %T", out["t"])
	}
	want := time.Date(2026, 7, 10, 12, 34, 56, 0, time.UTC)
	if !ts.Equal(want) {
		t.Errorf("t = %s, want %s", ts, want)
	}
}

func TestBindParams_missingUsesDefault(t *testing.T) {
	defs := []store.Param{{Name: "s", Type: "String", Default: "fallback"}}
	out, err := bindParams(defs, nil)
	if err != nil {
		t.Fatalf("bindParams: %v", err)
	}
	if v := out["s"]; v != "fallback" {
		t.Errorf("s = %v, want fallback", v)
	}
}

func TestBindParams_missingRequired(t *testing.T) {
	defs := []store.Param{{Name: "s", Type: "String"}}
	if _, err := bindParams(defs, nil); err == nil {
		t.Fatal("expected error for missing required param")
	}
}

func TestBindParams_unsupportedType(t *testing.T) {
	defs := []store.Param{{Name: "x", Type: "NopeType", Default: "v"}}
	if _, err := bindParams(defs, nil); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestSupportedParamType(t *testing.T) {
	for _, tn := range []string{"String", "UInt32", "Int64", "Float64", "DateTime", "Date"} {
		if !SupportedParamType(tn) {
			t.Errorf("%s should be supported", tn)
		}
	}
	if SupportedParamType("Enum8") {
		t.Errorf("Enum8 should not be supported yet")
	}
}
