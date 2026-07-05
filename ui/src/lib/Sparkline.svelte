<script lang="ts">
  interface Props {
    points: Array<{ t: string; value: number }>
    color?: string
    height?: number
  }
  let { points, color = '#3e63dd', height = 120 }: Props = $props()

  const width = 720
  const padL = 40
  const padB = 22
  const padT = 10
  const padR = 10

  const values = $derived(points.map((p) => p.value))
  const min = $derived(values.length ? Math.min(...values) : 0)
  const max = $derived(values.length ? Math.max(...values) : 1)
  const span = $derived(max - min || 1)

  function x(i: number): number {
    if (points.length <= 1) return padL
    return padL + (width - padL - padR) * (i / (points.length - 1))
  }
  function y(v: number): number {
    return padT + (height - padT - padB) * (1 - (v - min) / span)
  }
  const path = $derived(points.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(i)},${y(p.value)}`).join(' '))
  const area = $derived(
    points.length
      ? `${path} L${x(points.length - 1)},${height - padB} L${x(0)},${height - padB} Z`
      : '',
  )
  function fmt(t: string): string {
    const d = new Date(t)
    return `${String(d.getUTCHours()).padStart(2, '0')}:${String(d.getUTCMinutes()).padStart(2, '0')}`
  }
</script>

<svg viewBox="0 0 {width} {height}" class="chart" role="img" aria-label="Metric over time">
  {#each [0, 1] as f (f)}
    <line x1={padL} x2={width - padR} y1={y(min + span * f)} y2={y(min + span * f)} class="grid" />
    <text x={padL - 6} y={y(min + span * f) + 4} class="axis" text-anchor="end">{(min + span * f).toFixed(2)}</text>
  {/each}
  {#if area}
    <path d={area} fill={color} opacity="0.12" />
    <path d={path} fill="none" stroke={color} stroke-width="2" />
    {#each points as p, i (p.t)}
      <circle cx={x(i)} cy={y(p.value)} r="2.5" fill={color}><title>{fmt(p.t)}: {p.value}</title></circle>
    {/each}
  {/if}
  {#if points.length}
    <text x={padL} y={height - 4} class="axis" text-anchor="start">{fmt(points[0].t)}</text>
    <text x={width - padR} y={height - 4} class="axis" text-anchor="end">{fmt(points[points.length - 1].t)}</text>
  {/if}
</svg>

<style>
  .chart { width: 100%; height: auto; }
  .grid { stroke: var(--grid); stroke-width: 1; }
  .axis { fill: var(--muted); font-size: 11px; }
</style>
