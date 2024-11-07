export const iconProps = { size: 16 } as const
export const menuIconProps = { size: 14 } as const
export const popoverProps = { transitionDuration: 0 } as const
export const panelStyle = {
  display: 'flex',
  flexDirection: 'column',
  minHeight: '100vh',
  maxHeight: '100vh',
} as const
export const statusBarStyle = {
  display: 'flex',
  alignItems: 'center',
  padding: length(1),
  overflowWrap: 'break-word',
} as const

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
}

export function length(n: number) {
  return `${n * 0.25}rem`
}

export function fromNow(date: Date) {
  let secs = (new Date().getTime() - date.getTime()) / 1000
  const neg = secs < 0

  secs = Math.abs(secs)
  const repr =
    secs < 45 * 60
      ? `${Math.round(secs / 60)}m`
      : secs < 24 * 60 * 60
        ? `${Math.round(secs / 3600)}h`
        : secs < 7 * 24 * 60 * 60
          ? `${Math.round(secs / 86400)}d`
          : date.toLocaleDateString(undefined, {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })

  return neg ? `-${repr}` : repr
}

export function param(query: Record<string, string | number | boolean | undefined>) {
  const keys = Object.keys(query)
  if (!keys.length) return ''
  return `?${keys
    .filter(key => query[key] !== undefined)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(query[key]!)}`)
    .join('&')}`
}

export async function xfetch(url: string, options?: RequestInit): Promise<unknown>
export async function xfetch<T>(url: string, options?: RequestInit): Promise<T>
export async function xfetch<T>(
  url: string,
  options?: RequestInit,
): Promise<T | unknown> {
  if (typeof options?.body === 'string')
    options.headers = { 'Content-Type': 'application/json' }
  const response = await fetch(url, options)
  const text = await response.text()
  if (response.ok) return text && text !== 'OK' && JSON.parse(text)
  throw new Error(text || `${response.status} ${response.statusText}`)
}
