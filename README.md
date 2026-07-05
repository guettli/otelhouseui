# otelhouseui

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

See [docs/DESIGN.md](docs/DESIGN.md) for the architecture and the decisions behind it.
