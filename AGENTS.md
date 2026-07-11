# Working in otelhouseview

Stack: Go (backend + embedded SPA), `clickhouse-go/v2`, SQLite, TS/Svelte 5 SPA
(CodeMirror 6, ECharts). Single binary; SPA embedded via `go:embed`.

Read [docs/DESIGN.md](docs/DESIGN.md) before adding features — it fixes the architecture.

## Rules

- ClickHouse access is **read-only**. Never add write/DDL paths. Limits live on
  the CH server profile; do not try to enforce them in app code instead.
- Saved-query params bind via ClickHouse native `{name:Type}` parameters. Never
  build SQL by string interpolation.
- Keep the query surface signal-agnostic (the generic SQL → grid → auto-chart
  path). Do not add per-signal UIs in v1 except where DESIGN.md says.
- Add/adjust tests (`go test ./...`) for behaviour changes; keep CI green.
