import { Dialog as BlueprintDialog, Button, DialogBody, DialogFooter, Intent } from '@blueprintjs/core'
import { type ReactNode, useState } from 'react'

export function Dialog<T>({
  isOpen,
  close,
  title,
  callback,
  children,
  extraAction,
  intent = Intent.PRIMARY,
}: {
  isOpen: T
  close: () => void
  title: string
  callback: () => Promise<void>
  children: ReactNode
  extraAction?: ReactNode
  intent?: Intent
}) {
  const [loading, setLoading] = useState(false)
  const onConfirm = async () => {
    setLoading(true)
    try {
      await callback()
      close()
    } finally {
      setLoading(false)
    }
  }
  return (
    <BlueprintDialog
      title={title}
      isOpen={!!isOpen}
      isCloseButtonShown={false}
      onClose={close}
      canEscapeKeyClose
      canOutsideClickClose
      onOpening={node => {
        const elem = node.querySelector<HTMLInputElement>('.bp5-input')
        if (elem) {
          elem.focus()
          elem.setSelectionRange(elem.value.length, elem.value.length)
        }
      }}
    >
      <DialogBody>
        <div style={{ userSelect: 'none' }} onKeyDown={evt => evt.key === 'Enter' && onConfirm()}>
          {children}
        </div>
      </DialogBody>
      <DialogFooter
        actions={
          <>
            {extraAction}
            <Button text="Cancel" onClick={close} />
            <Button intent={intent} loading={loading} text="OK" onClick={onConfirm} />
          </>
        }
      />
    </BlueprintDialog>
  )
}
