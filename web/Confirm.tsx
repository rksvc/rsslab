import { Button, Dialog, DialogBody, DialogFooter, Intent } from '@blueprintjs/core'
import { type Dispatch, type ReactNode, type SetStateAction, useState } from 'react'

export function Confirm({
  open,
  setOpen,
  title,
  callback,
  children,
  intent = Intent.PRIMARY,
}: {
  open: boolean
  setOpen: Dispatch<SetStateAction<boolean>>
  title: string
  callback: () => Promise<void>
  children: ReactNode
  intent?: Intent
}) {
  const [loading, setLoading] = useState(false)
  const onClose = () => setOpen(false)
  const onConfirm = async () => {
    setLoading(true)
    try {
      await callback()
      onClose()
    } finally {
      setLoading(false)
    }
  }
  return (
    <Dialog
      title={title}
      isOpen={open}
      isCloseButtonShown={false}
      onClose={onClose}
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
            <Button className="select-none" text="Cancel" onClick={onClose} />
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
