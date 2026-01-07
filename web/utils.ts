import type { PopperModifierOverrides } from '@blueprintjs/core'
import type { Transformer } from './types.ts'

export const menuModifiers = { offset: { options: { offset: [-70, 8] } } } satisfies PopperModifierOverrides

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
}

export function length(n: number) {
  return `${n * 0.25}rem`
}

export function fromNow(date: Date, suffix = ' ago') {
  let minutes = (Date.now() - date.getTime()) / 1000 / 60
  const sign = minutes < 0 ? '-' : ''

  minutes = Math.abs(minutes)
  return minutes < 60
    ? `${sign}${Math.round(minutes)}m${suffix}`
    : minutes < 24 * 60
      ? `${sign}${Math.round(minutes / 60)}h${suffix}`
      : minutes < 7 * 24 * 60
        ? `${sign}${Math.round(minutes / (24 * 60))}d${suffix}`
        : date.toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          })
}

export function parseFeedLink(link: string): [Transformer, URL] | [undefined, string] {
  try {
    const url = new URL(link)
    if (url.protocol === 'rsslab:') {
      const host = url.host
      switch (host) {
        case 'html':
        case 'json':
        case 'js':
          return [host, url]
      }
    }
  } catch {}
  return [undefined, link]
}

export function param(query: Record<string, string | number | boolean | undefined>) {
  const entries = Object.entries(query)
  if (!entries.length) return ''
  return `?${entries
    .filter(([_, value]) => value !== undefined)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value!)}`)
    .join('&')}`
}

export async function xfetch<T = unknown>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options)
  const text = await response.text()
  if (response.ok) return text && JSON.parse(text)
  throw new Error(text || `${response.status} ${response.statusText}`)
}
