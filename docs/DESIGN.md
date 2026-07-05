# otelhouseui — design

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

Go + `clickhouse-go/v2`; embedded TS SPA (CodeMirror 6 SQL editor, ECharts);
SQLite for saved queries + sessions. Single binary, embedded SPA (like agentloop).

## Out of scope for v1

- **AI authoring assist** (phase 3): an agent limited to a read-only ClickHouse
  MCP tool, no shell/fs, no `--approve-all`. Rationale: OTel rows are
  attacker-influenced content, so least-privilege bounds prompt-injection blast
  radius to "ran a SELECT".
- **Trace waterfall** (phase 2).
- **Multi-chart dashboards** (phase 2).
