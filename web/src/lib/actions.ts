// Svelte actions: `use:codemirror` mounts a CodeMirror 6 SQL editor onto a DOM
// node, and `use:echart` mounts an ECharts instance and keeps its option in sync.

import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { sql } from '@codemirror/lang-sql'
import * as echarts from 'echarts'

export interface CodeMirrorParams {
  value: string
  onChange: (v: string) => void
}

export function codemirror(node: HTMLElement, params: CodeMirrorParams) {
  let current = params.value
  const state = EditorState.create({
    doc: params.value,
    extensions: [
      lineNumbers(),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      sql(),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) {
          current = u.state.doc.toString()
          params.onChange(current)
        }
      }),
    ],
  })
  const view = new EditorView({ state, parent: node })

  return {
    update(next: CodeMirrorParams) {
      params = next
      if (next.value !== current) {
        current = next.value
        view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: next.value } })
      }
    },
    destroy() {
      view.destroy()
    },
  }
}

export interface EChartParams {
  option: echarts.EChartsOption
}

export function echart(node: HTMLElement, params: EChartParams) {
  const chart = echarts.init(node)
  chart.setOption(params.option)

  const resize = () => chart.resize()
  window.addEventListener('resize', resize)

  return {
    update(next: EChartParams) {
      chart.setOption(next.option, { notMerge: true })
    },
    destroy() {
      window.removeEventListener('resize', resize)
      chart.dispose()
    },
  }
}
