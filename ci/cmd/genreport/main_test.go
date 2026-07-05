package main

import "testing"

func TestRound2(t *testing.T) {
	cases := map[float64]float64{
		0.615:  0.62,
		0.614:  0.61,
		1.0:    1.0,
		-0.615: -0.62,
		142.44: 142.44,
	}
	for in, want := range cases {
		if got := round2(in); got != want {
			t.Errorf("round2(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("OTELHOUSEUI_TEST_KEY", "")
	if got := envOr("OTELHOUSEUI_TEST_KEY", "fallback"); got != "fallback" {
		t.Errorf("empty env should fall back, got %q", got)
	}
	t.Setenv("OTELHOUSEUI_TEST_KEY", "value")
	if got := envOr("OTELHOUSEUI_TEST_KEY", "fallback"); got != "value" {
		t.Errorf("set env should win, got %q", got)
	}
}
