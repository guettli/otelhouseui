// Tiny hash-based router. The Go binary embeds a single index.html and does
// a fallback for unknown non-API paths (so both hash and history routing work
// after a hard refresh); hash keeps the client simple and avoids the extra
// history state ceremony.

import { writable } from 'svelte/store'

export interface Route {
  path: string
  params: Record<string, string>
}

function parseHash(): Route {
  const raw = window.location.hash.slice(1) || '/'
  return { path: raw, params: {} }
}

export const route = writable<Route>(parseHash())

if (typeof window !== 'undefined') {
  window.addEventListener('hashchange', () => route.set(parseHash()))
}

export function navigate(path: string): void {
  window.location.hash = path
}
