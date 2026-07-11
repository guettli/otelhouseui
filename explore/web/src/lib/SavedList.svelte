<script lang="ts">
  import { onMount } from 'svelte'
  import { listSaved, deleteSaved, type SavedQuery } from './api'

  let queries = $state<SavedQuery[]>([])
  let loading = $state(true)
  let error = $state<string | null>(null)

  async function load() {
    loading = true
    error = null
    try {
      queries = await listSaved()
    } catch (e) {
      error = (e as Error).message
    } finally {
      loading = false
    }
  }

  async function remove(q: SavedQuery) {
    if (!confirm(`Delete "${q.name}"?`)) return
    try {
      await deleteSaved(q.id)
      queries = queries.filter((x) => x.id !== q.id)
    } catch (e) {
      error = (e as Error).message
    }
  }

  onMount(load)
</script>

<h2>Saved queries</h2>
{#if loading}
  <p class="muted">Loading…</p>
{:else if error}
  <p class="error">{error}</p>
{:else if queries.length === 0}
  <p class="muted">No saved queries yet. Compose one in the workbench and hit “Save as…”.</p>
{:else}
  <ul class="saved">
    {#each queries as q (q.id)}
      <li>
        <a href={`#/saved/${q.id}`}>
          <b>{q.name}</b>
          <span class="desc">{q.description || '—'}</span>
        </a>
        <button class="ghost" onclick={() => remove(q)}>Delete</button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  h2 {
    margin: 0 0 12px;
  }
  ul.saved {
    list-style: none;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  ul.saved li {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px 14px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }
  ul.saved a {
    display: flex;
    flex-direction: column;
    color: inherit;
    text-decoration: none;
    gap: 2px;
  }
  .desc {
    color: var(--muted);
    font-size: 12px;
  }
  .muted {
    color: var(--muted);
  }
  .error {
    color: #e5484d;
  }
</style>
