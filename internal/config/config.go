// Package config loads runtime configuration from environment variables.
//
// The whole surface is env-only on purpose (12-factor): no config file,
// no CLI flags. Secrets come from a k8s Secret in production and from the
// developer's shell locally.
package config

import (
	"errors"
	"os"
	"strconv"
)

// Config is the resolved runtime configuration.
type Config struct {
	// ClickHouseDSN is a clickhouse-go v2 DSN, e.g.
	// "clickhouse://ro:***@ch:9000/otel". Required.
	ClickHouseDSN string
	// SQLitePath is the local SQLite database file path.
	SQLitePath string
	// Port is the TCP port the HTTP server listens on.
	Port int
}

// Load reads configuration from the process environment and applies defaults.
func Load() (Config, error) {
	c := Config{
		ClickHouseDSN: os.Getenv("CLICKHOUSE_DSN"),
		SQLitePath:    envOr("SQLITE_PATH", "./otelhouseui.db"),
	}
	if c.ClickHouseDSN == "" {
		return Config{}, errors.New("CLICKHOUSE_DSN is required")
	}
	portStr := envOr("PORT", "8080")
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return Config{}, errors.New("PORT must be an integer")
	}
	c.Port = p
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
