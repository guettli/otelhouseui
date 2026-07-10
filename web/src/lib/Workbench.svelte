<script lang="ts">
  import { codemirror } from './actions'
  import { runQuery, createSaved, type QueryResult, type Param } from './api'
  import ResultView from './ResultView.svelte'
  import { navigate } from './router'

  let sql = $state(`SELECT
  toStartOfInterval(TimeUnix, INTERVAL 1 MINUTE) AS t,
  ResourceAttributes['service.name']            AS service,
  sum(Value)                                    AS v
FROM otel_metrics_sum
WHERE MetricName = 'http.server.request.duration'
  AND TimeUnix >= now() - INTERVAL 1 HOUR
GROUP BY t, service
ORDER BY t`)

  let result = $state<QueryResult | null>(null)
  let error = $state<string | null>(null)
  let running = $state(false)
  let viz = $state<'auto' | 'table' | 'line'>('auto')

  // Save dialog state.
  let showSave = $state(false)
  let saveName = $state('')
  let saveDescription = $state('')
  let saveParamsJson = $state('[]')
  let saveError = $state<string | null>(null)

  async function run() {
    running = true
    error = null
    try {
      result = await runQuery(sql)
    } catch (e) {
      error = (e as Error).message
      result = null
    } finally {
      running = false
    }
  }

  async function save() {
    saveError = null
    let params: Param[] = []
    try {
      const parsed = JSON.parse(saveParamsJson)
      if (!Array.isArray(parsed)) throw new Error('params must be a JSON array')
      params = parsed as Param[]
    } catch (e) {
      saveError = `params_json: ${(e as Error).message}`
      return
    }
    try {
      const q = await createSaved({
        name: saveName,
        description: saveDescription,
        sql_template: sql,
        params,
      })
      showSave = false
      navigate(`/saved/${q.id}`)
    } catch (e) {
      saveError = (e as Error).message
    }
  }
</script>

<section class="editor-card">
  <div class="editor" use:codemirror={{ value: sql, onChange: (v) => (sql = v) }}></div>
  <div class="editor-actions">
    <button onclick={run} disabled={running}>{running ? 'Running…' : 'Run'}</button>
    <button class="ghost" onclick={() => (showSave = true)} disabled={!sql.trim()}>Save as…</button>
    <label class="viz">
      Viz:
      <select bind:value={viz}>
        <option value="auto">Auto</option>
        <option value="table">Table</option>
        <option value="line">Line</option>
      </select>
    </label>
  </div>
</section>

{#if error}
  <p class="error">{error}</p>
{/if}
{#if result}
  <section class="result-card">
    <ResultView {result} override={viz} />
  </section>
{/if}

{#if showSave}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={() => (showSave = false)} role="presentation">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <h3>Save query</h3>
      <label>Name<input bind:value={saveName} placeholder="e.g. my_metric_over_time" /></label>
      <label>Description<input bind:value={saveDescription} placeholder="What this query answers" /></label>
      <label>
        params_json (array of {`{name,type,label,widget,default}`})
        <textarea rows="6" bind:value={saveParamsJson}></textarea>
      </label>
      {#if saveError}<p class="error">{saveError}</p>{/if}
      <div class="modal-actions">
        <button class="ghost" onclick={() => (showSave = false)}>Cancel</button>
        <button onclick={save} disabled={!saveName.trim()}>Save</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .editor-card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px;
    margin-bottom: 16px;
  }
  .editor {
    border: 1px solid var(--grid);
    border-radius: 8px;
    min-height: 200px;
    overflow: hidden;
  }
  .editor-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 10px;
  }
  .viz {
    margin-left: auto;
    color: var(--muted);
    font-size: 12px;
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .result-card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px;
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
    margin: 0 0 16px;
  }

  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: grid;
    place-items: center;
    z-index: 10;
  }
  .modal {
    background: var(--card);
    border-radius: 12px;
    padding: 20px;
    width: min(480px, 92vw);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .modal h3 {
    margin: 0 0 4px;
  }
  .modal label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: var(--muted);
  }
  .modal input,
  .modal textarea {
    font: inherit;
    padding: 6px 8px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg);
    color: var(--fg);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 8px;
  }
</style>
