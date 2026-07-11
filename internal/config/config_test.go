package config

import (
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", "clickhouse://u:p@h:9000/db")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("PORT", "")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.SQLitePath != "./otelhouseview.db" {
		t.Errorf("SQLitePath = %q, want default", c.SQLitePath)
	}
	if c.Port != 8080 {
		t.Errorf("Port = %d, want 8080", c.Port)
	}
}

func TestLoad_missingDSN(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when CLICKHOUSE_DSN is unset")
	}
}

func TestLoad_badPort(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", "clickhouse://u:p@h:9000/db")
	t.Setenv("PORT", "not-a-port")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for non-numeric PORT")
	}
}
