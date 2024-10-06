import {
  Dialog as BlueprintDialog,
  Button,
  DialogBody,
  DialogFooter,
  Intent,
} from '@blueprintjs/core'
import { type ReactNode, useState } from 'react'
import { alert } from './utils'

export function Dialog<T>({
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
    } catch (error: any) {
      alert(error.toString())
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
      onOpening={node =>
        (
          node.getElementsByClassName('bp5-input') as HTMLCollectionOf<HTMLElement>
        )[0]?.focus()
      }
    >
      <DialogBody>
        <div
          style={{ userSelect: 'none' }}
          onKeyDown={evt => evt.key === 'Enter' && onConfirm()}
        >
          {children}
        </div>
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
    </BlueprintDialog>
  )
}
