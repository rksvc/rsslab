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
  headers?: string
  setHeaders?: Dispatch<SetStateAction<string>>
}) {
  const [newParamKey, setNewParamKey] = useState('')
  const newParamValue = useRef<HTMLInputElement>(null)
  const [newHeaderKey, setNewHeaderKey] = useState('')
  const newHeaderValue = useRef<HTMLInputElement>(null)

  const url = URL.parse(rawUrl)
  const headers =
    rawHeaders != null && (rawHeaders ? Object.entries(JSON.parse(rawHeaders) as Record<string, string>) : [])

  return (
    <div style={{ padding: length(3), textAlign: 'center' }}>
      {url || headers ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: length(2) }}>
          {url && (
            <div>
              <Label style={{ marginBottom: length(2) }}>Parameters</Label>
              {[...url.searchParams].map(([param, value], i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: expected
                <ControlGroup key={i} fill>
                  <InputGroup
                    value={param}
                    spellCheck="false"
                    onValueChange={newValue => {
                      url.search = `?${new URLSearchParams([...url.searchParams].with(i, [newValue, value]))}`
                      setUrl(url.href)
                    }}
                  />
                  <InputGroup
                    value={value}
                    spellCheck="false"
                    onValueChange={newValue => {
                      url.search = `?${new URLSearchParams([...url.searchParams].with(i, [param, newValue]))}`
                      setUrl(url.href)
                    }}
                  />
                  <Button
                    icon={<Trash2 />}
                    intent={Intent.DANGER}
                    variant={ButtonVariant.MINIMAL}
                    onClick={() => {
                      url.searchParams.delete(param, value)
                      setUrl(url.href)
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
                    url.searchParams.append(newParamKey, newParamValue.current.value)
                    setUrl(url.href)
                    setNewParamKey('')
                    newParamValue.current.value = ''
                  }}
                />
              </ControlGroup>
            </div>
          )}
          {headers ? (
            <div>
              <Label style={{ marginBottom: length(2) }}>Headers</Label>
              {headers.map(([key, value], i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: expected
                <ControlGroup key={i} fill>
                  <InputGroup
                    value={key}
                    spellCheck="false"
                    onValueChange={newValue => {
                      setHeaders?.(JSON.stringify(Object.fromEntries(headers.with(i, [newValue, value]))))
                    }}
                  />
                  <InputGroup
                    value={value}
                    spellCheck="false"
                    onValueChange={newValue => {
                      setHeaders?.(JSON.stringify(Object.fromEntries(headers.with(i, [key, newValue]))))
                    }}
                  />
                  <Button
                    icon={<Trash2 />}
                    intent={Intent.DANGER}
                    variant={ButtonVariant.MINIMAL}
                    onClick={() => {
                      headers.splice(i, 1)
                      setHeaders?.(headers.length ? JSON.stringify(Object.fromEntries(headers)) : '')
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
                    headers.push([newHeaderKey, newHeaderValue.current.value])
                    setHeaders?.(JSON.stringify(Object.fromEntries(headers)))
                    setNewHeaderKey('')
                    newHeaderValue.current.value = ''
                  }}
                />
              </ControlGroup>
            </div>
          ) : undefined}
          {url && (
            <AnchorButton
              text="Preview"
              style={{ marginTop: length(1) }}
              href={`api/proxy${param({
                url: rawUrl,
                headers: rawHeaders || undefined,
              })}`}
              target="_blank"
              intent={Intent.PRIMARY}
              endIcon={<ExternalLink />}
              variant={ButtonVariant.OUTLINED}
            />
          )}
        </div>
      ) : (
        <i style={{ userSelect: 'none' }}>No Data</i>
      )}
    </div>
  )
}
