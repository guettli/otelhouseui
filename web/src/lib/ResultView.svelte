<script lang="ts">
  import type { QueryResult } from './api'
  import { pickViz } from './autoViz'
  import { chartOption } from './chartOption'
  import { echart } from './actions'

  interface Props {
    result: QueryResult
    override?: 'auto' | 'table' | 'line'
  }
  let { result, override = 'auto' }: Props = $props()

  const viz = $derived.by(() => {
    if (override === 'table') return { kind: 'table' as const }
    if (override === 'line') {
      // Force a line view even if the heuristic would fall back to table.
      const auto = pickViz(result)
      if (auto.kind !== 'table') return auto
      return { kind: 'table' as const }
    }
    return pickViz(result)
  })
</script>

{#if viz.kind === 'table' || result.rows.length === 0}
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          {#each result.columns as c (c.name)}
            <th title={c.type}>{c.name}<span class="type">{c.type}</span></th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each result.rows as row, i (i)}
          <tr>
            {#each row as cell, j (j)}
              <td>{cell === null ? '—' : String(cell)}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{:else}
  <div class="chart" use:echart={{ option: chartOption(result, viz) }}></div>
{/if}

<p class="meta">
  {result.rows.length.toLocaleString()} rows · {result.elapsed_ms} ms
</p>

<style>
  .table-wrap {
    overflow: auto;
    max-height: 60vh;
    border: 1px solid var(--border);
    border-radius: 8px;
  }
  table {
    border-collapse: collapse;
    width: 100%;
    font-size: 13px;
  }
  thead {
    position: sticky;
    top: 0;
    background: var(--card);
  }
  th,
  td {
    text-align: left;
    padding: 6px 10px;
    border-bottom: 1px solid var(--grid);
    vertical-align: top;
  }
  th {
    font-weight: 600;
    color: var(--muted);
    white-space: nowrap;
  }
  .type {
    display: block;
    font-weight: 400;
    color: var(--muted);
    font-size: 11px;
  }
  .chart {
    width: 100%;
    height: 420px;
  }
  .meta {
    color: var(--muted);
    font-size: 12px;
    margin: 6px 0 0;
  }
</style>
