import { Button, Classes, Intent, MenuItem, Popover, TextArea } from '@blueprintjs/core'
import { type CSSProperties, useRef, useState } from 'react'

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
  const confirm = async () => {
    if (!inputRef.current) return
    setLoading(true)
    try {
      await onConfirm(inputRef.current.value)
      closerRef.current?.click()
    } finally {
      setLoading(false)
    }
  }

  return (
    <Popover
      usePortal={false}
      placement="right"
      transitionDuration={0}
      modifiers={{
        flip: { enabled: true },
        offset: { enabled: true, options: { offset: [0, 4] } },
      }}
      shouldReturnFocusOnClose
      content={
        <>
          <TextArea
            defaultValue={defaultValue}
            placeholder={placeholder}
            inputRef={inputRef}
            cols={30}
            spellCheck="false"
            disabled={loading}
            autoResize
            style={{
              borderBottomLeftRadius: 0,
              borderBottomRightRadius: 0,
              ...textAreaStyle,
            }}
            onKeyDown={async evt => {
              if (evt.key === 'Enter') {
                evt.preventDefault()
                await confirm()
              }
            }}
          />
          <Button loading={loading} intent={Intent.PRIMARY} text="OK" onClick={confirm} fill />
          <div className={Classes.POPOVER_DISMISS} ref={closerRef} hidden />
        </>
      }
      onOpening={node => {
        const elem = node.querySelector<HTMLInputElement>('.bp6-input')
        if (elem) {
          elem.focus()
          elem.setSelectionRange(elem.value.length, elem.value.length)
        }
      }}
    >
      <MenuItem text={menuText} icon={menuIcon} shouldDismissPopover={false} />
    </Popover>
  )
}
