<script lang="ts">
  import { SEVERITY_ORDER, SEVERITY_COLOR } from './report'

  interface Props {
    data: Array<{ t: string; severity: string; count: number }>
    height?: number
  }
  let { data, height = 220 }: Props = $props()

  const width = 720
  const padL = 40
  const padB = 28
  const padT = 10
  const padR = 10

  // Pivot rows into one column per time bucket, each a map severity -> count.
  const buckets = $derived.by(() => {
    const byTime = new Map<string, Map<string, number>>()
    for (const r of data) {
      if (!byTime.has(r.t)) byTime.set(r.t, new Map())
      byTime.get(r.t)!.set(r.severity, (byTime.get(r.t)!.get(r.severity) ?? 0) + r.count)
    }
    return [...byTime.entries()]
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([t, m]) => ({ t, sev: m, total: [...m.values()].reduce((a, b) => a + b, 0) }))
  })

  const severities = $derived(
    SEVERITY_ORDER.filter((s) => data.some((r) => r.severity === s)),
  )
  const maxTotal = $derived(Math.max(1, ...buckets.map((b) => b.total)))
  const bw = $derived(
    buckets.length ? (width - padL - padR) / buckets.length : width - padL - padR,
  )

  function y(v: number): number {
    return padT + (height - padT - padB) * (1 - v / maxTotal)
  }
  function fmt(t: string): string {
    const d = new Date(t)
    return `${String(d.getUTCHours()).padStart(2, '0')}:${String(d.getUTCMinutes()).padStart(2, '0')}`
  }
</script>

<svg viewBox="0 0 {width} {height}" class="chart" role="img" aria-label="Log volume over time by severity">
  <!-- y axis gridlines -->
  {#each [0, 0.5, 1] as f (f)}
    <line x1={padL} x2={width - padR} y1={y(maxTotal * f)} y2={y(maxTotal * f)} class="grid" />
    <text x={padL - 6} y={y(maxTotal * f) + 4} class="axis" text-anchor="end">{Math.round(maxTotal * f)}</text>
  {/each}

  {#each buckets as b, i (b.t)}
    {@const x = padL + i * bw}
    {@const acc = { yTop: y(0) }}
    {#each severities as sev (sev)}
      {@const c = b.sev.get(sev) ?? 0}
      {#if c > 0}
        {@const h = (height - padT - padB) * (c / maxTotal)}
        {@const yTop = acc.yTop - h}
        <rect x={x + bw * 0.12} y={yTop} width={bw * 0.76} height={Math.max(0, h)}
          fill={SEVERITY_COLOR[sev] ?? '#8b8d98'}>
          <title>{fmt(b.t)} · {sev}: {c}</title>
        </rect>
        {(acc.yTop = yTop, '')}
      {/if}
    {/each}
    {#if i % Math.ceil(buckets.length / 8 || 1) === 0}
      <text x={x + bw / 2} y={height - padB + 16} class="axis" text-anchor="middle">{fmt(b.t)}</text>
    {/if}
  {/each}
</svg>

<style>
  .chart { width: 100%; height: auto; }
  .grid { stroke: var(--grid); stroke-width: 1; }
  .axis { fill: var(--muted); font-size: 11px; }
</style>
