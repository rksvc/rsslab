import { Button, Dialog, DialogBody, DialogFooter, Intent } from '@blueprintjs/core'
import { type ReactNode, useState } from 'react'

export function Confirm<T>({
  isOpen,
  close,
  title,
  callback,
  children,
  intent = Intent.PRIMARY,
}: {
  isOpen: T
  close: () => void
  title: string
  callback: () => Promise<void>
  children: ReactNode
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
    <Dialog
      title={title}
      isOpen={!!isOpen}
      isCloseButtonShown={false}
      onClose={close}
      canEscapeKeyClose
      canOutsideClickClose
      onOpening={node => node.getElementsByTagName('input')[0]?.focus()}
    >
      <DialogBody>
        <div onKeyDown={evt => evt.key === 'Enter' && onConfirm()}>{children}</div>
      </DialogBody>
      <DialogFooter
        actions={
          <>
            <Button className="select-none" text="Cancel" onClick={close} />
            <Button
              className="select-none"
              intent={intent}
              loading={loading}
              text="OK"
              onClick={onConfirm}
            />
          </>
        }
      />
    </Dialog>
  )
}
