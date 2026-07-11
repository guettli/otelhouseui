import { describe, expect, it, afterEach } from 'vitest'
import { apiBase, url } from './api'

// The whole point of these two functions is that the SPA works both at the root
// (standalone binary) and under a host-chosen prefix (agentloop mounts explore
// at /explore). Regressing to absolute '/api/...' URLs would 404 against the
// host app, and that is only visible when mounted — hence the unit test.

function setBase(v: string | undefined) {
  ;(globalThis as unknown as { window: { __EXPLORE_BASE__?: string } }).window = {
    __EXPLORE_BASE__: v,
  }
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window
})

describe('apiBase', () => {
  it('defaults to / when the base is absent', () => {
    setBase(undefined)
    expect(apiBase()).toBe('/')
  })

  it('defaults to / when the placeholder was never substituted (vite dev)', () => {
    setBase('/__EXPLORE_BASE__/')
    expect(apiBase()).toBe('/')
  })

  it('uses the injected mount base', () => {
    setBase('/explore/')
    expect(apiBase()).toBe('/explore/')
  })

  it('adds the trailing slash if the host omitted it', () => {
    setBase('/explore')
    expect(apiBase()).toBe('/explore/')
  })
})

describe('url', () => {
  it('builds root-mounted URLs', () => {
    setBase('/')
    expect(url('/api/query')).toBe('/api/query')
  })

  it('builds prefix-mounted URLs without doubling slashes', () => {
    setBase('/explore/')
    expect(url('/api/saved-queries/7/run')).toBe('/explore/api/saved-queries/7/run')
    expect(url('api/query')).toBe('/explore/api/query')
  })
})
