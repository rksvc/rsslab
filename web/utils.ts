import type { CSSProperties } from 'react'
import type { Transformer } from './types.ts'

export const iconProps = { size: 16 } as const
export const menuIconProps = { size: 14 } as const
export const panelStyle = {
  display: 'flex',
  flexDirection: 'column',
  height: '100%',
} satisfies CSSProperties
export const statusBarStyle = {
  display: 'flex',
  alignItems: 'center',
  padding: length(1),
  overflowWrap: 'break-word',
} satisfies CSSProperties

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
}

export function length(n: number) {
  return `${n * 0.25}rem`
}

export function fromNow(date: Date, withSuffix = false) {
  let secs = (new Date().getTime() - date.getTime()) / 1000
  const neg = secs < 0

  secs = Math.abs(secs)
  const suffix = withSuffix ? ' ago' : ''
  const repr =
    secs < 45 * 60
      ? `${Math.round(secs / 60)}m${suffix}`
      : secs < 24 * 60 * 60
        ? `${Math.round(secs / 3600)}h${suffix}`
        : secs < 7 * 24 * 60 * 60
          ? `${Math.round(secs / 86400)}d${suffix}`
          : date.toLocaleDateString(undefined, {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })

  return neg ? `-${repr}` : repr
}

export function compareTitle(a: { title: string }, b: { title: string }) {
  const lhs = a.title.toLowerCase()
  const rhs = b.title.toLowerCase()
  return lhs === rhs ? 0 : lhs < rhs ? -1 : +1
}

export function parseFeedLink(link: string): [Transformer | undefined, string] {
  const i = link.indexOf(':')
  if (i !== -1) {
    const scheme = link.slice(0, i)
    switch (scheme) {
      case 'html':
      case 'json':
        return [scheme, link.slice(i + 1)]
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
