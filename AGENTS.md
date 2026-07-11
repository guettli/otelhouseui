# Working in otelhouseview

This repo is the **read path** of the otelhouse pipeline, and it is a
**library, not a service**. It ships three exported packages with external
consumers, all pinned at untagged pseudo-versions:

| Package | What it is | Imported by |
| ------- | ---------- | ----------- |
| `otelstore/` | read-only typed ClickHouse client for the stock clickhouseexporter schema | agentloop, otelhouse e2e |
| `ciview/` | renders a trace as a self-contained HTML page | agentloop |
| `explore/` | the SQL workbench as an **embeddable sub-application**: one `http.Handler` serving its own JSON API + its own embedded Svelte SPA, mountable under any prefix | agentloop (at `/explore`) |

Plus `cmd/otelhouseview` (thin wrapper that mounts `explore` at `/`, for local
dev only — **not** the production shape) and a static-report pipeline (`ui/` +
`ci/`).

Stack: Go, `clickhouse-go/v2`, SQLite, TS/Svelte 5 SPA (CodeMirror 6, ECharts);
SPA embedded via `go:embed`.

Read [docs/DESIGN.md](docs/DESIGN.md) before adding features — it fixes the architecture.

## The usage pattern (internalize this before changing anything)

A **host application** (today: agentloop) imports these packages, passes a
**tenant-scoped ClickHouse DSN**, and mounts `explore` behind **its own** auth:

```go
svc, _ := explore.New(ctx, explore.Config{DSN: dsn, StorePath: p, Prefix: "/explore"})
mux.Handle("/explore/", app.RequireSession(http.StripPrefix("/explore", svc.Handler())))
```

otelhouseview never authenticates and never enforces tenancy. The boundary is
the **ClickHouse identity in the DSN**: a `<tenant>_ro` user with row policies
pinned to `ResourceAttributes['tenant']`, plus a server-side settings profile
capping query cost. That is *why* a tenant can safely be handed arbitrary SQL.

## Rules

- **You have API-compat obligations.** Changing the signatures of `Store`,
  `GetTrace`, `ListTraces`, `RenderTrace`, `RenderIndex`, `explore.New` /
  `Config` / `Handler` / `Close` — or the shape of `Span` / `LogRecord` /
  `Trace` / `TraceSummary` — breaks agentloop and otelhouse. Prefer additive
  change; a break is a cross-repo change, not a local one. `internal/` and
  `explore/internal/` are private and free to churn: put anything that is not a
  deliberate contract there.
- **`explore` does no auth, on purpose. Never add one.** It is an
  arbitrary-SQL endpoint; the host wraps it. Equally: never add a tenant
  predicate or Go-side query limits — both would *look* like a security
  boundary without being one, and would drift from the real one in ClickHouse.
- **`explore` must not assume it owns `/`.** The host may mount it anywhere.
  The SPA builds every request URL from `window.__EXPLORE_BASE__`, which the Go
  handler substitutes into `index.html` (Vite bakes the `/__EXPLORE_BASE__/`
  placeholder into every absolute URL it emits). Never write `fetch('/api/...')`.
- **The SPA build is committed** (`explore/web/build/`). `go:embed` ships what
  is in the module and importers do not run pnpm. Run `make web` and commit the
  result whenever `explore/web/src` changes — every importer is stale otherwise.
- ClickHouse access is **read-only**. Never add write/DDL paths. Limits live on
  the CH server profile; do not try to enforce them in app code instead.
- Saved-query params bind via ClickHouse native `{name:Type}` parameters. Never
  build SQL by string interpolation.
- Keep the **query** surface signal-agnostic (the generic SQL → grid →
  auto-chart path). The trace waterfall (`ciview`) is the one deliberate
  per-signal view and it is shipped; do not add further per-signal UIs without
  a DESIGN.md decision.
- Add/adjust tests (`go test ./...`) for behaviour changes; keep CI green.
