<script lang="ts">
  import { route } from './lib/router'
  import Workbench from './lib/Workbench.svelte'
  import SavedList from './lib/SavedList.svelte'
  import SavedRunner from './lib/SavedRunner.svelte'

  const savedIdMatch = $derived(($route.path.match(/^\/saved\/(\d+)$/)))
</script>

<main>
  <header>
    <div class="brand">
      <span class="logo">◇</span>
      <div>
        <h1>otelhouseui</h1>
        <p class="sub">Query OpenTelemetry data in ClickHouse.</p>
      </div>
    </div>
    <nav>
      <a href="#/" class:active={$route.path === '/'}>Workbench</a>
      <a href="#/saved" class:active={$route.path.startsWith('/saved')}>Saved</a>
    </nav>
  </header>

  {#if $route.path === '/'}
    <Workbench />
  {:else if savedIdMatch}
    <SavedRunner id={Number(savedIdMatch[1])} />
  {:else if $route.path === '/saved'}
    <SavedList />
  {:else}
    <p class="muted">Not found. <a href="#/">Home</a></p>
  {/if}
</main>

<style>
  :global(:root) {
    --bg: #f6f7f9;
    --card: #ffffff;
    --fg: #1a1d24;
    --muted: #6b7280;
    --border: #e4e6eb;
    --grid: #edeff2;
    --accent: #3e63dd;
  }
  @media (prefers-color-scheme: dark) {
    :global(:root) {
      --bg: #0e1116;
      --card: #171b22;
      --fg: #e6e8ec;
      --muted: #8b94a3;
      --border: #262b34;
      --grid: #222831;
      --accent: #6b8afd;
    }
  }
  :global(body) {
    margin: 0;
    background: var(--bg);
    color: var(--fg);
    font: 14px/1.5 -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  }
  :global(button) {
    font: inherit;
    padding: 6px 14px;
    border-radius: 6px;
    border: 1px solid var(--accent);
    background: var(--accent);
    color: white;
    cursor: pointer;
  }
  :global(button.ghost) {
    background: transparent;
    color: var(--accent);
  }
  :global(button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  main {
    max-width: 960px;
    margin: 0 auto;
    padding: 24px 20px 60px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
    margin-bottom: 24px;
  }
  .brand {
    display: flex;
    gap: 12px;
    align-items: center;
  }
  .logo {
    font-size: 28px;
    color: var(--accent);
  }
  h1 {
    font-size: 20px;
    margin: 0;
  }
  .sub {
    margin: 2px 0 0;
    color: var(--muted);
    font-size: 13px;
  }
  nav {
    display: flex;
    gap: 12px;
  }
  nav a {
    color: var(--muted);
    text-decoration: none;
    font-size: 13px;
    padding: 6px 10px;
    border-radius: 6px;
  }
  nav a.active {
    background: var(--card);
    color: var(--fg);
    border: 1px solid var(--border);
  }
  .muted {
    color: var(--muted);
  }
</style>
