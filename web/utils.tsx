import React from 'react'
import ReactDOM from 'react-dom/client'
import { Alert } from './Alert'

export const iconProps = { size: 16 } as const
export const menuIconProps = { size: 14 } as const
export const popoverProps = { transitionDuration: 0 } as const

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ')
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
  try {
    const response = await fetch(url, options)
    const text = await response.text()
    if (response.ok) return text && text !== 'OK' && JSON.parse(text)
    throw new Error(text || `${response.status} ${response.statusText}`)
  } catch (error) {
    alert(String(error))
  }
}

export function alert(error: string): never {
  const container = document.body.appendChild(document.createElement('div'))
  const root = ReactDOM.createRoot(container)
  root.render(
    <React.StrictMode>
      <Alert error={error} root={root} container={container} />
    </React.StrictMode>,
  )
  throw new Error(error)
}
