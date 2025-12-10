import type { PopperModifierOverrides } from '@blueprintjs/core'
import type { Transformer } from './types.ts'

export const menuModifiers = { offset: { options: { offset: [-70, 8] } } } satisfies PopperModifierOverrides

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
}

export function length(n: number) {
  return `${n * 0.25}rem`
}

export function fromNow(date: Date) {
  let minutes = (Date.now() - date.getTime()) / 1000 / 60
  const sign = minutes < 0 ? '-' : ''

  minutes = Math.abs(minutes)
  return minutes < 60
    ? `${sign}${Math.round(minutes)}m`
    : minutes < 24 * 60
      ? `${sign}${Math.round(minutes / 60)}h`
      : minutes < 7 * 24 * 60
        ? `${sign}${Math.round(minutes / (24 * 60))}d`
        : date.toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          })
}

export function fromNowVerbose(date: Date) {
  let minutes = (Date.now() - date.getTime()) / 1000 / 60
  const neg = minutes < 0

  minutes = Math.abs(minutes)
  const parts = []
  if (minutes > 24 * 60) {
    const days = Math.floor(minutes / (24 * 60))
    if (days > 7)
      return date.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    parts.push(`${days}d`)
    minutes %= 24 * 60
  }
  if (minutes > 60) {
    parts.push(`${Math.floor(minutes / 60)}h`)
    minutes %= 60
  }
  minutes = Math.round(minutes)
  if (minutes > 0 || !parts.length) parts.push(`${minutes}m`)

  const repr = `${parts.join('')} ago`
  return neg ? `-${repr}` : repr
}

export function compareTitle(a: { title: string }, b: { title: string }) {
  const lhs = a.title.toLowerCase()
  const rhs = b.title.toLowerCase()
  return lhs === rhs ? 0 : lhs < rhs ? -1 : +1
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
