// Convert a QueryResult + Viz into an ECharts option object.

import type { EChartsOption } from 'echarts'
import type { QueryResult } from './api'
import type { Viz } from './autoViz'

export function chartOption(result: QueryResult, viz: Viz): EChartsOption {
  if (viz.kind === 'line') {
    return {
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'time' },
      yAxis: { type: 'value' },
      dataZoom: [{ type: 'inside' }],
      series: [
        {
          type: 'line',
          name: result.columns[viz.valueIdx].name,
          showSymbol: false,
          data: result.rows.map((r) => [r[viz.timeIdx], r[viz.valueIdx]]),
        },
      ],
    }
  }
  if (viz.kind === 'grouped') {
    const series = new Map<string, [unknown, unknown][]>()
    for (const row of result.rows) {
      const g = String(row[viz.groupIdx])
      if (!series.has(g)) series.set(g, [])
      series.get(g)!.push([row[viz.timeIdx], row[viz.valueIdx]])
    }
    return {
      tooltip: { trigger: 'axis' },
      legend: { top: 0 },
      xAxis: { type: 'time' },
      yAxis: { type: 'value' },
      dataZoom: [{ type: 'inside' }],
      grid: { top: 32 },
      series: Array.from(series, ([name, data]) => ({
        type: 'line',
        name,
        showSymbol: false,
        data,
      })),
    }
  }
  return {}
}
