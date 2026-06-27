import { Button, Classes, Intent, MenuItem, PopoverNext, TextArea } from '@blueprintjs/core'
import { type CSSProperties, type JSX, useRef, useState } from 'react'

export default function TextEditor({
  menuText,
  menuIcon,
  defaultValue,
  placeholder,
  textAreaStyle,
  onConfirm,
}: {
  menuText: string
  menuIcon: JSX.Element
  defaultValue?: string
  placeholder?: string
  textAreaStyle?: CSSProperties
  onConfirm: (value: string) => Promise<void>
}) {
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const closerRef = useRef<HTMLDivElement>(null)
  const confirm = () => {
    if (!inputRef.current) return
    setLoading(true)
    // https://github.com/react/react/issues/34131
    void onConfirm(inputRef.current.value)
      .then(() => closerRef.current?.click())
      .finally(() => setLoading(false))
  }

  return (
    <PopoverNext
      usePortal={false}
      placement="right"
      middleware={{ offset: { mainAxis: 4 } }}
      content={
        <>
          <TextArea
            defaultValue={defaultValue}
            placeholder={placeholder}
            inputRef={inputRef}
            spellCheck="false"
            disabled={loading}
            autoResize
            style={{ ...textAreaStyle, maxHeight: '90vh' }}
            onKeyDown={evt => {
              if (evt.key === 'Enter') {
                evt.preventDefault()
                confirm()
              }
            }}
          />
          <Button loading={loading} intent={Intent.PRIMARY} text="OK" onClick={confirm} fill />
          <div className={Classes.POPOVER_DISMISS} ref={closerRef} hidden />
        </>
      }
      onOpening={node => {
        const elem = node.querySelector<HTMLInputElement>(`.${Classes.INPUT}`)
        if (elem) {
          elem.focus()
          elem.setSelectionRange(elem.value.length, elem.value.length)
        }
      }}
    >
      <MenuItem text={menuText} icon={menuIcon} shouldDismissPopover={false} />
    </PopoverNext>
  )
}
