// Thin wrapper over the otelhouseview HTTP API. Kept minimal — the SPA is small
// enough that generated clients or fetch libraries would be overkill.
//
// Every request URL is built from the *mount base*, because this SPA is served
// by an embeddable Go handler that a host app may mount under any prefix
// (agentloop mounts it at /explore). The Go handler substitutes the real base
// into the index.html it serves, which sets window.__EXPLORE_BASE__. Absolute
// paths like fetch('/api/query') would 404 against the host app; never
// reintroduce one.

declare global {
  interface Window {
    __EXPLORE_BASE__?: string
  }
}

// apiBase returns the mount base, always with a trailing slash. It falls back to
// '/' when the placeholder was not substituted — which is exactly the case under
// `vite dev`, where index.html is served by Vite at the root, not by Go.
export function apiBase(): string {
  const raw = typeof window === 'undefined' ? undefined : window.__EXPLORE_BASE__
  if (!raw || raw.includes('__EXPLORE_BASE__')) return '/'
  return raw.endsWith('/') ? raw : `${raw}/`
}

// url joins a handler-relative path (with or without a leading slash) onto the base.
export function url(path: string): string {
  return apiBase() + path.replace(/^\/+/, '')
}

export interface Column {
  name: string
  type: string
}

export interface QueryResult {
  columns: Column[]
  rows: unknown[][]
  elapsed_ms: number
}

export interface Param {
  name: string
  type: string
  label?: string
  widget?: string
  default?: unknown
}

export interface SavedQuery {
  id: number
  name: string
  description: string
  sql_template: string
  params: Param[]
  default_viz: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface SavedQueryInput {
  name: string
  description?: string
  sql_template: string
  params?: Param[]
  default_viz?: string
}

async function jsonOrThrow<T>(res: Response): Promise<T> {
  if (res.ok) {
    if (res.status === 204) return undefined as unknown as T
    return (await res.json()) as T
  }
  let msg = res.statusText
  try {
    const body = (await res.json()) as { error?: string }
    if (body?.error) msg = body.error
  } catch {
    /* swallow — not JSON */
  }
  throw new Error(msg)
}

export async function runQuery(sql: string, params?: Record<string, unknown>): Promise<QueryResult> {
  const res = await fetch(url('/api/query'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sql, params }),
  })
  return jsonOrThrow<QueryResult>(res)
}

export async function listSaved(): Promise<SavedQuery[]> {
  const res = await fetch(url('/api/saved-queries'))
  return jsonOrThrow<SavedQuery[]>(res)
}

export async function getSaved(id: number): Promise<SavedQuery> {
  const res = await fetch(url(`/api/saved-queries/${id}`))
  return jsonOrThrow<SavedQuery>(res)
}

export async function createSaved(q: SavedQueryInput): Promise<SavedQuery> {
  const res = await fetch(url('/api/saved-queries'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(q),
  })
  return jsonOrThrow<SavedQuery>(res)
}

export async function deleteSaved(id: number): Promise<void> {
  const res = await fetch(url(`/api/saved-queries/${id}`), { method: 'DELETE' })
  await jsonOrThrow<void>(res)
}

export async function runSaved(id: number, params: Record<string, unknown>): Promise<QueryResult> {
  const res = await fetch(url(`/api/saved-queries/${id}/run`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ params }),
  })
  return jsonOrThrow<QueryResult>(res)
}
