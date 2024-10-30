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

export function param(query: Record<string, string | number | boolean | undefined>) {
  const keys = Object.keys(query)
  if (!keys.length) return ''
  return `?${keys
    .filter(key => query[key] !== undefined)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(query[key]!)}`)
    .join('&')}`
}

export type Xfetch = {
  (url: string, options?: RequestInit): Promise<unknown>
  <T>(url: string, options?: RequestInit): Promise<T>
}
