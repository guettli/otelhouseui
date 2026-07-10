import { describe, expect, it } from 'vitest'
import { pickViz } from './autoViz'
import type { QueryResult } from './api'

function r(cols: [string, string][], rows: unknown[][]): QueryResult {
  return {
    columns: cols.map(([name, type]) => ({ name, type })),
    rows,
    elapsed_ms: 0,
  }
}

describe('pickViz', () => {
  it('table when there are no time columns', () => {
    const v = pickViz(r([['a', 'String'], ['b', 'Float64']], [['x', 1]]))
    expect(v.kind).toBe('table')
  })

  it('table when the result is empty', () => {
    const v = pickViz(r([['t', 'DateTime'], ['v', 'Float64']], []))
    expect(v.kind).toBe('table')
  })

  it('line when time + single numeric', () => {
    const v = pickViz(r([['t', 'DateTime'], ['v', 'Float64']], [['2026-01-01', 1]]))
    expect(v.kind).toBe('line')
    if (v.kind === 'line') {
      expect(v.timeIdx).toBe(0)
      expect(v.valueIdx).toBe(1)
    }
  })

  it('grouped when time + one low-cardinality string + one numeric', () => {
    const rows: unknown[][] = []
    for (let i = 0; i < 30; i++) {
      rows.push(['2026-01-01', i % 3 === 0 ? 'a' : 'b', i])
    }
    const v = pickViz(r([['t', 'DateTime'], ['svc', 'String'], ['v', 'Float64']], rows))
    expect(v.kind).toBe('grouped')
    if (v.kind === 'grouped') {
      expect(v.groupIdx).toBe(1)
      expect(v.valueIdx).toBe(2)
    }
  })

  it('line (not grouped) when the string cardinality is too high', () => {
    const rows: unknown[][] = []
    for (let i = 0; i < 50; i++) {
      rows.push(['2026-01-01', `svc-${i}`, i])
    }
    const v = pickViz(r([['t', 'DateTime'], ['svc', 'String'], ['v', 'Float64']], rows))
    expect(v.kind).toBe('line')
  })

  it('handles DateTime64 as a time type', () => {
    const v = pickViz(r([['t', 'DateTime64(3)'], ['v', 'Float64']], [['2026-01-01', 1]]))
    expect(v.kind).toBe('line')
  })
})
