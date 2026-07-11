# otelhouseview — design

## Goal

Query + visualize OTel data in ClickHouse (clickhouseexporter schema) — and
expose that read core as a Go library other repos import. AI assistance is a
later layer, not on the v1 path.

Each deployment of the UI/service serves **one tenant**: it is pointed at a
single ClickHouse read-only identity and has no tenant switcher, no per-request
tenant, no cross-tenant view. That is a property of the *deployment*, not of the
data — see [Tenancy](#tenancy).

## Core substrate

`SQL or saved-query (+params) → result grid → auto time-series chart`.

This single path serves metrics charts, log-volume-over-time, and span rate/latency
without per-signal UIs, and is the exact surface the future AI writes into.
Traces (waterfall) is the one signal needing a bespoke view — it is the single
carve-out, and it **shipped** as the `ciview` package (see
[Public packages](#public-packages-otelstore--ciview) and Out of scope below).

## Auto-chart heuristic

If column 0 is `Date`/`DateTime`/`DateTime64` and ≥1 remaining column is numeric →
time-series (line/area); an optional low-cardinality string column splits series.
Otherwise → table. The user can override the visualization per query.

## Data model (SQLite)

```
saved_query(id, name, description, sql_template, params_json, default_viz,
            created_by, created_at, updated_at)
```

`params_json`: `[{name, type (ClickHouse type), label, widget, default}]`

## Public packages: `otelstore/` + `ciview/`

This repo is not only a service. Two packages are **exported API with external
consumers**:

- **`otelstore/`** — read-only, typed ClickHouse client for the stock
  clickhouseexporter schema (`Store`, `GetTrace`, `ListTraces`, plus
  `MemoryStore` as the in-process fake).
- **`ciview/`** — renders a trace as a self-contained HTML page
  (`RenderTrace`, `RenderIndex`).

Importers today: **agentloop** (both packages) and **otelhouse**'s e2e harness
(`otelstore`). Both pin untagged pseudo-versions, so they pick up `main` on
`go get -u`.

**Consequence, and the rule for anyone working here:** changing the signatures
of `Store` / `GetTrace` / `ListTraces` / `RenderTrace` / `RenderIndex`, or the
shape of the types they carry (`Span`, `LogRecord`, `Trace`, `TraceSummary`),
breaks downstream repos. Prefer additive change; if a break is unavoidable, it
is a cross-repo change, not a local one. Everything under `internal/` is
private and free to churn — put anything that is not a deliberate contract
there.

`otelstore` also exposes `WithTracesTable` / `WithLogsTable` and a `DB()`
escape hatch (the raw `*sql.DB`, owned by the store). `DB()` exists because
`GetTrace` is **span-anchored** — it returns `ErrNotFound` when the spans query
is empty even if the trace id has logs — so a consumer that needs to assert
"logs landed" must query around it. otelhouse's e2e does exactly that.

## Tenancy

The library is deliberately **tenant-blind**, and the isolation boundary lives
in ClickHouse, not in Go:

- The [otelhouse](https://github.com/guettli/otelhouse) gateway authenticates a
  per-tenant JWT and stamps `ResourceAttributes['tenant']` on every record at
  write time. Tenancy is established there, fail-closed.
- Reads are constrained by a ClickHouse **row policy** bound to a per-tenant
  read-only user (`<tenant>_ro`). The DSN's identity *is* the tenant.
- Therefore no query in `otelstore` adds a tenant predicate, and none should: a
  filter in Go would look like a security boundary without being one.

A single service/UI process is still pointed at one DSN and so serves one
tenant. Multi-tenant means "many tenants share one ClickHouse behind row
policies", not "one process fans out across tenants" — there is no tenant
switcher in the UI and no plan for one.

## ClickHouse access

- One **read-only** CH user per deployment (creds from a k8s secret / env);
  that user is the tenant boundary (see [Tenancy](#tenancy)).
- Limits enforced on the CH user **profile** (server-side), not app code:
  `max_execution_time`, `max_rows_to_read`, `max_bytes_to_read`,
  `max_result_rows`, `max_memory_usage`.
- Params bound via **native ClickHouse query parameters** — no interpolation.
- Time-series starter queries use `toStartOfInterval(...) ... WITH FILL` for
  gap-free charts.

## Stack

Go + `clickhouse-go/v2`; embedded Svelte 5 SPA (CodeMirror 6 SQL editor,
ECharts); SQLite for saved queries + sessions. Single binary, embedded SPA
(like agentloop).

## Out of scope for v1

- **AI authoring assist** (phase 3): an agent limited to a read-only ClickHouse
  MCP tool, no shell/fs, no `--approve-all`. Rationale: OTel rows are
  attacker-influenced content, so least-privilege bounds prompt-injection blast
  radius to "ran a SELECT".
- ~~**Trace waterfall** (phase 2).~~ Shipped as the `ciview` package — it
  already existed inside agentloop and was moved here rather than rewritten.
- **Multi-chart dashboards** (phase 2).

## Static report pipeline (added)

The interactive query UI above is the v1 product. A second, self-contained path
produces a **static HTML report** for CI/e2e visibility and is the surface this
repo's own CI exercises end to end.

```
emit (OTLP) → OTel Collector → ClickHouse → genreport → report.json
            → Svelte build (single-file) → dist/index.html → cluster ConfigMap
```

- **`ui/`** — Vite + Svelte 5 app built with `vite-plugin-singlefile`. It
  `import`s `src/lib/report-data.json` (baked in, not fetched) so the build is
  one offline HTML file. `src/lib/report.ts` is the schema contract with the Go
  builder; keep the two in sync.
- **`ci/`** — the Dagger pipeline (Go), the single source of truth for tests.
  `cmd/emit` spreads records over a time window and varies severity / metric
  value / span duration so the report has a real time-series, a severity
  breakdown, a metric curve and an OK/ERROR trace mix. `cmd/genreport` queries
  the clickhouseexporter tables (`otel_logs`, `otel_traces`,
  `otel_metrics_gauge`) and fails loudly rather than emit a partial report.
- **Upload** — on pushes to `main`, the report is written into the
  `otelhouseui-report` ConfigMap (namespace `otelhouseui`) via a
  ServiceAccount token whose RBAC is limited to that one ConfigMap kind in that
  namespace. A caddy Deployment mounts the ConfigMap and serves it. The kube
  manifests live in the `guettli/gitops` repo (`k8s/plain/otelhouseui`); the
  gitops rename to `otelhouseview` has not landed yet, so the CI upload still
  targets the old slug.

Why `ui/` and `web/` are two apps: both are Svelte 5, so the split is not a
framework boundary — it is a delivery boundary. `web/` is a live SPA that
fetches from the Go service; `ui/` bakes its data in at build time and must
render as one offline HTML file with no backend, which rules out the shared
API client and the router. What they *could* share is the chart layer
(`web/src/lib/chartOption.ts` against `ui/src/lib/Sparkline.svelte` and
`StackedTimeSeries.svelte`); that is worth extracting once the report's chart
needs stop moving.
