# Working in otelhouseview

This repo is the **read path** of the otelhouse pipeline, and it is two things:

1. **A published Go library.** `otelstore/` (read-only typed ClickHouse client
   for the stock clickhouseexporter schema) and `ciview/` (renders a trace as a
   self-contained HTML page) are **exported packages with external consumers**:
   agentloop imports both, otelhouse's e2e harness imports `otelstore`. Both pin
   untagged pseudo-versions.
2. **A service + UI.** Go backend with an embedded Svelte 5 SPA, plus a static
   report pipeline. Deployed at `otelhouseview.thomas-guettler.de`.

Stack: Go, `clickhouse-go/v2`, SQLite, TS/Svelte 5 SPA (CodeMirror 6, ECharts).
Single binary; SPA embedded via `go:embed`.

Read [docs/DESIGN.md](docs/DESIGN.md) before adding features — it fixes the architecture.

## Rules

- **You have API-compat obligations.** Changing the signatures of `Store`,
  `GetTrace`, `ListTraces`, `RenderTrace`, `RenderIndex` — or the shape of
  `Span` / `LogRecord` / `Trace` / `TraceSummary` — breaks agentloop and
  otelhouse. Prefer additive change; a break is a cross-repo change, not a local
  one. `internal/` is private and free to churn: put anything that is not a
  deliberate contract there.
- ClickHouse access is **read-only**. Never add write/DDL paths. Limits live on
  the CH server profile; do not try to enforce them in app code instead.
- `otelstore` is **tenant-blind** by design — isolation is the ClickHouse
  identity in the DSN plus row policies. Never add a tenant predicate in Go.
- Saved-query params bind via ClickHouse native `{name:Type}` parameters. Never
  build SQL by string interpolation.
- Keep the **query** surface signal-agnostic (the generic SQL → grid →
  auto-chart path). The trace waterfall (`ciview`) is the one deliberate
  per-signal view and it is shipped; do not add further per-signal UIs without
  a DESIGN.md decision.
- Add/adjust tests (`go test ./...`) for behaviour changes; keep CI green.
