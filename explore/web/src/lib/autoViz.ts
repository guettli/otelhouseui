// autoViz decides how to render a QueryResult. It mirrors the heuristic from
// docs/DESIGN.md so the same rule lives in exactly one place on the client:
//
//   column 0 is a Date/DateTime/DateTime64 AND ≥1 remaining column is numeric
//     → time-series (line)
//   plus: if there is exactly one low-cardinality string column, split series
//         by it and use the single numeric column as the value.
//   otherwise → table.

import type { Column, QueryResult } from './api'

export type Viz =
  | { kind: 'table' }
  | { kind: 'line'; timeIdx: number; valueIdx: number }
  | { kind: 'grouped'; timeIdx: number; groupIdx: number; valueIdx: number }

const GROUP_CARDINALITY_LIMIT = 20

const NUMERIC_TYPE = /^(U?Int(8|16|32|64|128|256)|Float(32|64)|Decimal(32|64|128|256)?)/
const TIME_TYPE = /^(Date|DateTime|DateTime64)/

function isTimeType(t: string): boolean {
  return TIME_TYPE.test(t)
}

function isNumericType(t: string): boolean {
  return NUMERIC_TYPE.test(t)
}

function isStringType(t: string): boolean {
  return t === 'String' || t.startsWith('FixedString') || t.startsWith('LowCardinality(String')
}

export function pickViz(result: QueryResult): Viz {
  const cols = result.columns
  if (cols.length < 2 || result.rows.length === 0) {
    return { kind: 'table' }
  }
  if (!isTimeType(cols[0].type)) {
    return { kind: 'table' }
  }

  const others = cols.slice(1)
  const numericIdxs: number[] = []
  const stringIdxs: number[] = []
  others.forEach((c: Column, i: number) => {
    const idx = i + 1
    if (isNumericType(c.type)) numericIdxs.push(idx)
    else if (isStringType(c.type)) stringIdxs.push(idx)
  })
  if (numericIdxs.length === 0) {
    return { kind: 'table' }
  }

  if (stringIdxs.length === 1 && numericIdxs.length === 1) {
    const groupIdx = stringIdxs[0]
    const distinct = new Set<unknown>()
    for (const row of result.rows) {
      distinct.add(row[groupIdx])
      if (distinct.size > GROUP_CARDINALITY_LIMIT) break
    }
    if (distinct.size <= GROUP_CARDINALITY_LIMIT) {
      return { kind: 'grouped', timeIdx: 0, groupIdx, valueIdx: numericIdxs[0] }
    }
  }

  return { kind: 'line', timeIdx: 0, valueIdx: numericIdxs[0] }
}
