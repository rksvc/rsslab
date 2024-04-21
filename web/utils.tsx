import React from 'react';
import ReactDOM from 'react-dom/client';
import { FetchOptions, ofetch } from 'ofetch';
import { Error as Err } from './Error';
import { Confirm } from './Confirm';

export const iconProps = { size: 16 };
export const menuIconProps = { size: 14 };
export const popoverProps = { transitionDuration: 0 };

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

export function confirmDeletion(title: string, callback: () => void) {
  const container = document.body.appendChild(document.createElement('div'));
  const root = ReactDOM.createRoot(container);
  root.render(
    <React.StrictMode>
      <Confirm
        text={`Are you sure you want to delete ${title}?`}
        callback={callback}
        root={root}
        container={container}
      />
    </React.StrictMode>,
  );
}
