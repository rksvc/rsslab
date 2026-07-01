import {
  AnchorButton,
  Button,
  ButtonVariant,
  ControlGroup,
  InputGroup,
  Intent,
  Label,
} from '@blueprintjs/core'
import { type Dispatch, type SetStateAction, useRef, useState } from 'react'
import { ExternalLink, Plus, Trash2 } from 'react-feather'

import { length, param } from './utils'

export default function HttpRequestParams({
  url: rawUrl,
  setUrl,
  headers: rawHeaders,
  setHeaders,
}: {
  url: string
  setUrl: Dispatch<SetStateAction<string>>
  headers: string
  setHeaders: Dispatch<SetStateAction<string>>
}) {
  const [newParamKey, setNewParamKey] = useState('')
  const newParamValue = useRef<HTMLInputElement>(null)
  const [newHeaderKey, setNewHeaderKey] = useState('')
  const newHeaderValue = useRef<HTMLInputElement>(null)

  const url = URL.parse(rawUrl)
  const headers = rawHeaders ? Object.entries(JSON.parse(rawHeaders) as Record<string, string>) : []

  return (
    <div
      style={{
        padding: length(3),
        textAlign: 'center',
        display: 'flex',
        flexDirection: 'column',
        gap: length(2),
      }}
    >
      {url && (
        <div>
          <Label style={{ marginBottom: length(2) }}>Parameters</Label>
          {[...url.searchParams].map(([param, value], i) => (
            <ControlGroup key={i} fill>
              <InputGroup
                value={param}
                spellCheck="false"
                onValueChange={newValue => {
                  const newUrl = new URL(url)
                  newUrl.search = `?${new URLSearchParams([...url.searchParams].with(i, [newValue, value]))}`
                  setUrl(newUrl.href)
                }}
              />
              <InputGroup
                value={value}
                spellCheck="false"
                onValueChange={newValue => {
                  const newUrl = new URL(url)
                  newUrl.search = `?${new URLSearchParams([...url.searchParams].with(i, [param, newValue]))}`
                  setUrl(newUrl.href)
                }}
              />
              <Button
                icon={<Trash2 />}
                intent={Intent.DANGER}
                variant={ButtonVariant.MINIMAL}
                onClick={() => {
                  const newUrl = new URL(url)
                  newUrl.searchParams.delete(param, value)
                  setUrl(newUrl.href)
                }}
              />
            </ControlGroup>
          ))}
          <ControlGroup fill>
            <InputGroup
              value={newParamKey}
              onValueChange={value => setNewParamKey(value)}
              spellCheck="false"
            />
            <InputGroup inputRef={newParamValue} spellCheck="false" />
            <Button
              icon={<Plus />}
              variant={ButtonVariant.MINIMAL}
              disabled={!newParamKey}
              onClick={() => {
                if (!newParamValue.current) return
                const newUrl = new URL(url)
                newUrl.searchParams.append(newParamKey, newParamValue.current.value)
                setUrl(newUrl.href)
                setNewParamKey('')
                newParamValue.current.value = ''
              }}
            />
          </ControlGroup>
        </div>
      )}
      <div>
        <Label style={{ marginBottom: length(2) }}>Headers</Label>
        {headers.map(([key, value], i) => (
          <ControlGroup key={i} fill>
            <InputGroup
              value={key}
              spellCheck="false"
              onValueChange={newValue => {
                setHeaders(JSON.stringify(Object.fromEntries(headers.with(i, [newValue, value]))))
              }}
            />
            <InputGroup
              value={value}
              spellCheck="false"
              onValueChange={newValue => {
                setHeaders(JSON.stringify(Object.fromEntries(headers.with(i, [key, newValue]))))
              }}
            />
            <Button
              icon={<Trash2 />}
              intent={Intent.DANGER}
              variant={ButtonVariant.MINIMAL}
              onClick={() => {
                const value = headers.toSpliced(i, 1)
                setHeaders(value.length ? JSON.stringify(Object.fromEntries(value)) : '')
              }}
            />
          </ControlGroup>
        ))}
        <ControlGroup fill>
          <InputGroup
            value={newHeaderKey}
            onValueChange={value => setNewHeaderKey(value)}
            spellCheck="false"
          />
          <InputGroup inputRef={newHeaderValue} spellCheck="false" />
          <Button
            icon={<Plus />}
            variant={ButtonVariant.MINIMAL}
            disabled={!newHeaderKey}
            onClick={() => {
              if (!newHeaderValue.current) return
              setHeaders(
                JSON.stringify(
                  Object.fromEntries([...headers, [newHeaderKey, newHeaderValue.current.value]]),
                ),
              )
              setNewHeaderKey('')
              newHeaderValue.current.value = ''
            }}
          />
        </ControlGroup>
      </div>
      {url && (
        <AnchorButton
          text="Preview"
          style={{ marginTop: length(1) }}
          href={`api/proxy${param({ url: rawUrl, headers: rawHeaders || undefined })}`}
          target="_blank"
          intent={Intent.PRIMARY}
          endIcon={<ExternalLink />}
          variant={ButtonVariant.OUTLINED}
        />
      )}
    </div>
  )
}
