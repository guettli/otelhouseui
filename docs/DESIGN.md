# otelhouseview — design

## Goal

Query + visualize OTel data in ClickHouse (clickhouseexporter schema) for a small,
trusted team, one tenant. AI assistance is a later layer, not on the v1 path.

## Core substrate

`SQL or saved-query (+params) → result grid → auto time-series chart`.

This single path serves metrics charts, log-volume-over-time, and span rate/latency
without per-signal UIs, and is the exact surface the future AI writes into.
Traces (waterfall) is the one signal needing a bespoke view — deferred.

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

## ClickHouse access

- One **read-only** CH user (creds from a k8s secret / env), one tenant.
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
- **Trace waterfall** (phase 2).
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
  `otelhouseview-report` ConfigMap (namespace `otelhouseview`) via a
  ServiceAccount token whose RBAC is limited to that one ConfigMap kind in that
  namespace. A caddy Deployment mounts the ConfigMap and serves it. The kube
  manifests live in the `guettli/gitops` repo (`k8s/plain/otelhouseview`).

Why `ui/` and `web/` are two apps: both are Svelte 5, so the split is not a
framework boundary — it is a delivery boundary. `web/` is a live SPA that
fetches from the Go service; `ui/` bakes its data in at build time and must
render as one offline HTML file with no backend, which rules out the shared
API client and the router. What they *could* share is the chart layer
(`web/src/lib/chartOption.ts` against `ui/src/lib/Sparkline.svelte` and
`StackedTimeSeries.svelte`); that is worth extracting once the report's chart
needs stop moving.
