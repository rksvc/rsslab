import type { Transformer } from './types.ts'

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
}

export function length(n: number) {
  return `${n * 0.25}rem`
}

export function fromNow(date: Date, withSuffix = true) {
  let minutes = (new Date().getTime() - date.getTime()) / 1000 / 60
  const neg = minutes < 0

  minutes = Math.abs(minutes)
  const suffix = withSuffix ? ' ago' : ''
  const repr =
    minutes < 60
      ? `${Math.round(minutes)}m${suffix}`
      : minutes < 24 * 60
        ? `${Math.round(minutes / 60)}h${suffix}`
        : minutes < 7 * 24 * 60
          ? `${Math.round(minutes / (24 * 60))}d${suffix}`
          : date.toLocaleDateString(undefined, {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })

  return neg ? `-${repr}` : repr
}

export function fromNowVerbose(date: Date) {
  let minutes = (new Date().getTime() - date.getTime()) / 1000 / 60
  const neg = minutes < 0

  minutes = Math.abs(minutes)
  const parts = []
  if (minutes > 24 * 60) {
    parts.push(`${Math.floor(minutes / (24 * 60))}d`)
    minutes %= 24 * 60
  }
  if (minutes > 60) {
    parts.push(`${Math.floor(minutes / 60)}h`)
    minutes %= 60
  }
  minutes = Math.round(minutes)
  if (minutes > 0) parts.push(`${minutes}m`)

  const repr = `${parts.join('')} ago`
  return neg ? `-${repr}` : repr
}

export function compareTitle(a: { title: string }, b: { title: string }) {
  const lhs = a.title.toLowerCase()
  const rhs = b.title.toLowerCase()
  return lhs === rhs ? 0 : lhs < rhs ? -1 : +1
}

export function parseFeedLink(link: string, onlyURL = false): [Transformer | undefined, string] {
  const i = link.indexOf(':')
  if (i !== -1) {
    const scheme = link.slice(0, i)
    switch (scheme) {
      case 'html':
      case 'json':
        return [scheme, onlyURL ? JSON.parse(link.slice(i + 1)).url : link.slice(i + 1)]
    }
  }
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

export async function xfetch(url: string, options?: RequestInit): Promise<unknown>
export async function xfetch<T>(url: string, options?: RequestInit): Promise<T>
export async function xfetch<T>(url: string, options?: RequestInit): Promise<T | unknown> {
  const response = await fetch(url, options)
  const text = await response.text()
  if (response.ok) return text && JSON.parse(text)
  throw new Error(text || `${response.status} ${response.statusText}`)
}
