import React from 'react';
import ReactDOM from 'react-dom/client';
import { FetchOptions, ofetch } from 'ofetch';
import { Error as Err } from './Error';

export const iconProps = { size: 16 } as const;
export const menuIconProps = { size: 14 } as const;
export const popoverProps = { transitionDuration: 0 } as const;

export function cn(...classNames: (string | undefined | null | false)[]) {
  return classNames.filter(Boolean).join(' ');
}

export async function xfetch(
  request: RequestInfo,
  options?: FetchOptions<'json'>,
): Promise<unknown>;
export async function xfetch<T>(
  request: RequestInfo,
  options?: FetchOptions<'json'>,
): Promise<T>;
export async function xfetch<T>(
  request: RequestInfo,
  options?: FetchOptions<'json'>,
): Promise<T | unknown> {
  const response = await ofetch.raw<T | string>(request, {
    ...options,
    ignoreResponseError: true,
  });
  const data = response._data;
  if (typeof data === 'string' && !response.ok) alert(data.trim() || response.statusText);
  return data;
}

function alert(error: string): never {
  const container = document.body.appendChild(document.createElement('div'));
  const root = ReactDOM.createRoot(container);
  root.render(
    <React.StrictMode>
      <Err error={error} root={root} container={container} />
    </React.StrictMode>,
  );
  throw new Error(error);
}
