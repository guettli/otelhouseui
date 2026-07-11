# otelhouseview

A lightweight UI to **query and visualize OpenTelemetry data** (logs, metrics, traces)
stored in ClickHouse by the OpenTelemetry Collector's
[clickhouseexporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/clickhouseexporter),
whose schema this tool targets (`otel_logs`, `otel_traces`,
`otel_metrics_{gauge,sum,histogram,exponential_histogram,summary}`).

## What it does

- Run ClickHouse SQL against the OTel tables and get a result grid.
- Automatically render a time-series chart when a result looks like `(time, [group], value)`.
- Save a query as a **named, parameterized template**; re-run it deterministically
  from a simple form — no SQL, no LLM.
- Ships a starter library of queries for the exporter schema.

## Safety model

The UI connects to ClickHouse as a **read-only** user; execution limits
(`max_execution_time`, `max_rows_to_read`, `max_bytes_to_read`, `max_result_rows`,
`max_memory_usage`) live on that user's **server-side profile**, so no query can
exceed them. Saved-query params bind via ClickHouse native parameters
(`{name:Type}`) — never string interpolation.

## Roadmap (not in v1)

- AI-assisted query authoring (agent restricted to a read-only ClickHouse MCP tool).
- Trace waterfall view.
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
   `otelhouseview-report` ConfigMap, served by a caddy Deployment.

`make ci` runs exactly what GitHub Actions runs (needs a reachable Dagger
engine). `make ui-build` builds the report locally against the committed sample
data. The report renders offline — it is one HTML file with everything inlined.

See [docs/DESIGN.md](docs/DESIGN.md) for the architecture and the decisions behind it.
