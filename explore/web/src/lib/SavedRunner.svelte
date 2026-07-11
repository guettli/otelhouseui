<script lang="ts">
  import { onMount } from 'svelte'
  import { getSaved, runSaved, type SavedQuery, type QueryResult } from './api'
  import ResultView from './ResultView.svelte'

  interface Props {
    id: number
  }
  let { id }: Props = $props()

  let query = $state<SavedQuery | null>(null)
  let values = $state<Record<string, string>>({})
  let result = $state<QueryResult | null>(null)
  let error = $state<string | null>(null)
  let running = $state(false)

  onMount(async () => {
    try {
      const q = await getSaved(id)
      query = q
      const initial: Record<string, string> = {}
      for (const p of q.params) {
        initial[p.name] = p.default != null ? String(p.default) : ''
      }
      values = initial
    } catch (e) {
      error = (e as Error).message
    }
  })

  function widgetFor(type: string): 'text' | 'number' | 'datetime-local' | 'date' {
    if (type === 'Date') return 'date'
    if (type === 'DateTime') return 'datetime-local'
    if (type.startsWith('Int') || type.startsWith('UInt') || type.startsWith('Float')) return 'number'
    return 'text'
  }

  async function run() {
    if (!query) return
    running = true
    error = null
    try {
      result = await runSaved(query.id, values)
    } catch (e) {
      error = (e as Error).message
      result = null
    } finally {
      running = false
    }
  }
</script>

{#if !query && !error}
  <p class="muted">Loading…</p>
{:else if error && !query}
  <p class="error">{error}</p>
{:else if query}
  <h2>{query.name}</h2>
  {#if query.description}<p class="muted">{query.description}</p>{/if}

  <div class="param-card">
    {#if query.params.length === 0}
      <p class="muted">No parameters.</p>
    {:else}
      <div class="params">
        {#each query.params as p (p.name)}
          <label>
            <span>{p.label || p.name} <small>({p.type})</small></span>
            <input
              type={widgetFor(p.type)}
              bind:value={values[p.name]}
              placeholder={p.default != null ? String(p.default) : ''}
            />
          </label>
        {/each}
      </div>
    {/if}
    <div class="actions">
      <button onclick={run} disabled={running}>{running ? 'Running…' : 'Run'}</button>
    </div>
  </div>

  {#if error}<p class="error">{error}</p>{/if}
  {#if result}
    <section class="result-card">
      <ResultView {result} override={query.default_viz === 'line' ? 'line' : 'auto'} />
    </section>
  {/if}

  <details class="sql">
    <summary>SQL template</summary>
    <pre>{query.sql_template}</pre>
  </details>
{/if}

<style>
  h2 {
    margin: 0 0 4px;
  }
  .muted {
    color: var(--muted);
  }
  .param-card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 14px;
    margin: 12px 0;
  }
  .params {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 10px;
  }
  .params label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: var(--muted);
  }
  .params input {
    font: inherit;
    padding: 6px 8px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg);
    color: var(--fg);
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
  }
  .result-card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px;
  }
  .sql {
    margin-top: 20px;
    color: var(--muted);
    font-size: 12px;
  }
  .sql pre {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 10px;
    overflow: auto;
  }
  .error {
    color: #e5484d;
    background: rgba(229, 72, 77, 0.08);
    border: 1px solid rgba(229, 72, 77, 0.2);
    padding: 8px 12px;
    border-radius: 8px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 12px;
    white-space: pre-wrap;
  }
</style>
