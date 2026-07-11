# otelhouseview

The **read path** for OpenTelemetry data (logs, metrics, traces) stored in
ClickHouse by the OpenTelemetry Collector's
[clickhouseexporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/clickhouseexporter),
whose stock schema this repo targets (`otel_logs`, `otel_traces`,
`otel_metrics_{gauge,sum,histogram,exponential_histogram,summary}`).
The write path is [otelhouse](https://github.com/guettli/otelhouse).

It ships **two** things:

1. **A published Go library** — the exported packages
   [`otelstore`](otelstore) (a read-only, typed ClickHouse client for that
   schema) and [`ciview`](ciview) (renders a trace as a self-contained HTML
   page: Gantt waterfall + span tree + logs). Other repos import them:
   [agentloop](https://github.com/guettli/agentloop) (both) and otelhouse's e2e
   harness (`otelstore`). See [Go library](#go-library-otelstore--ciview) —
   these are **API-compat surfaces**, not internals.
2. **A UI** — a query service + embedded Svelte 5 SPA, plus a static-report
   pipeline. The deployed report/UI is served at
   **<https://otelhouseview.thomas-guettler.de>**.

## What the UI does

- Run ClickHouse SQL against the OTel tables and get a result grid.
- Automatically render a time-series chart when a result looks like `(time, [group], value)`.
- Save a query as a **named, parameterized template**; re-run it deterministically
  from a simple form — no SQL, no LLM.
- Ships a starter library of queries for the exporter schema.
- Renders a trace as a waterfall (that is `ciview`, shipped and exported).

## Safety model

The UI connects to ClickHouse as a **read-only** user; execution limits
(`max_execution_time`, `max_rows_to_read`, `max_bytes_to_read`, `max_result_rows`,
`max_memory_usage`) live on that user's **server-side profile**, so no query can
exceed them. Saved-query params bind via ClickHouse native parameters
(`{name:Type}`) — never string interpolation.

## Go library: `otelstore` + `ciview`

The read core is importable — so consumers do not each grow their own copy of
"SELECT spans out of the clickhouseexporter schema":

```go
import "github.com/guettli/otelhouseview/otelstore"

store, err := otelstore.OpenClickHouse(ctx, dsn) // dsn = a read-only CH user
trace, err := store.GetTrace(ctx, traceID)       // spans + logs, one trace
recent, err := store.ListTraces(ctx, 50)         // newest-first summaries
```

`MemoryStore` is the in-process fake, so consumers can test their rendering
without a live ClickHouse.

`ciview` renders a trace as a self-contained HTML page — a Gantt waterfall, a
collapsible span tree, and the logs emitted under each span, with no backend
and no framework. Mount the bytes on any route:

```go
import "github.com/guettli/otelhouseview/ciview"

page, err := ciview.RenderTrace(trace)        // one trace, full waterfall
index, err := ciview.RenderIndex(recent)      // listing of recent traces
```

### These packages have external consumers

`otelstore/` and `ciview/` are **exported API with out-of-repo importers**:

| Consumer | Imports |
| -------- | ------- |
| [agentloop](https://github.com/guettli/agentloop) | `otelstore` + `ciview` |
| [otelhouse](https://github.com/guettli/otelhouse) e2e harness | `otelstore` |

Both pin untagged pseudo-versions, so a breaking change to `Store`, `GetTrace`,
`ListTraces`, `RenderTrace` or `RenderIndex` (signatures, or the `Span` /
`LogRecord` / `Trace` / `TraceSummary` shapes they carry) breaks them on their
next `go get -u`. Change these deliberately, additively where possible.
Everything under `internal/` is private and free to churn.

### Options and the `DB()` escape hatch

```go
store, err := otelstore.OpenClickHouse(ctx, dsn,
    otelstore.WithTracesTable("otel_traces"), // only if the exporter was
    otelstore.WithLogsTable("otel_logs"),     // configured off its defaults
)

db := store.DB() // *sql.DB, owned by the store — do NOT close it
```

`DB()` exposes the underlying `database/sql` handle so a caller can run its own
read-only queries (e.g. aggregating `otel_metrics_sum`) without this package
having to grow an API for them.

**Sharp edge — `GetTrace` is span-anchored.** It reports `ErrNotFound` when the
**spans** query comes back empty, even if the trace id has log rows. A trace
that produced only logs (or whose spans have not landed yet) therefore looks
absent. This is deliberate — the viewer renders a waterfall and has nothing to
hang logs on without spans — but it means "did my data arrive?" is not a
question `GetTrace` can answer. otelhouse's e2e asserts log ingestion through
`DB()` for exactly this reason.

### Tenancy: the library is deliberately tenant-blind

In a multi-tenant deployment the isolation boundary is the ClickHouse identity
in the DSN — the [otelhouse](https://github.com/guettli/otelhouse) gateway
stamps `ResourceAttributes['tenant']` at write time, and reads are constrained
by a row policy bound to a per-tenant `<tenant>_ro` read-only user. No query
here adds a tenant predicate, and none should: a filter in Go would *look* like
a security boundary without being one. Callers pass a DSN already scoped to one
tenant.

## Roadmap (not in v1)

- AI-assisted query authoring (agent restricted to a read-only ClickHouse MCP tool).
- Multi-chart dashboards.

## Static report pipeline (CI)

Alongside the interactive UI, otelhouseview ships a **static report** path: a
self-contained HTML report (Svelte + Vite, `ui/`) rendered entirely from real
OTel data, with no live backend. The Dagger CI pipeline (`ci/`) is the single
source of truth and, end to end:

1. runs an **ephemeral ClickHouse** and the **upstream OTel Collector**;
2. runs `ci/cmd/emit` to push spread **logs, metrics and traces** over OTLP;
3. runs `ci/cmd/genreport` to query ClickHouse and write `report.json`;
4. bakes that JSON into a single `dist/index.html` via the Svelte build;
5. on pushes to `main`, uploads the report into the cluster's
   `otelhouseui-report` ConfigMap, served by a caddy Deployment at
   <https://otelhouseview.thomas-guettler.de>.

`make ci` runs exactly what GitHub Actions runs (needs a reachable Dagger
engine). `make ui-build` builds the report locally against the committed sample
data. The report renders offline — it is one HTML file with everything inlined.

See [docs/DESIGN.md](docs/DESIGN.md) for the architecture and the decisions behind it.
