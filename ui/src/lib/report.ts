// Report is the contract between the Go report builder (ci/report.go, which
// queries ClickHouse) and this UI. The builder writes src/lib/report-data.json
// in the exact shape below; `App.svelte` imports it at build time so the data
// is baked into the single-file HTML. Keep this in sync with ci/report.go.

export interface Report {
  // ISO-8601 UTC timestamp the report was generated (stamped by the builder).
  generatedAt: string
  source: {
    repo: string
    commit: string
    runURL: string
  }
  // Inclusive time window the queried telemetry falls into.
  window: { from: string; to: string }
  summary: {
    logs: number
    spans: number
    traces: number
    metricPoints: number
    errorLogs: number
  }
  // Log volume bucketed over time, split by severity. One entry per (bucket,
  // severity); the UI pivots it into stacked series.
  logVolume: Array<{ t: string; severity: string; count: number }>
  // Log count per severity, for the breakdown bar.
  logSeverity: Array<{ severity: string; count: number }>
  // Latest numeric samples per metric (name -> points over time).
  metrics: Array<{
    name: string
    unit: string
    latest: number
    points: Array<{ t: string; value: number }>
  }>
  // Recent traces, newest first.
  traces: Array<{
    traceId: string
    service: string
    rootSpan: string
    spanCount: number
    durationMs: number
    status: string
    startTime: string
  }>
}

export const SEVERITY_ORDER = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE', 'UNSET']

export const SEVERITY_COLOR: Record<string, string> = {
  ERROR: '#e5484d',
  WARN: '#f5a623',
  INFO: '#3e63dd',
  DEBUG: '#8b8d98',
  TRACE: '#6e56cf',
  UNSET: '#8b8d98',
}

export function statusColor(status: string): string {
  const s = status.toUpperCase()
  if (s.includes('ERROR')) return '#e5484d'
  if (s.includes('OK')) return '#30a46c'
  return '#8b8d98'
}
