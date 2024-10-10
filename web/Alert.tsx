import { Alert as BlueprintAlert } from '@blueprintjs/core'
import { useState } from 'react'
import type ReactDOM from 'react-dom/client'

export function Alert({
  error,
  root,
  container,
}: {
  error: string
  root: ReactDOM.Root
  container: HTMLDivElement
}) {
  const [open, setOpen] = useState(true)
  return (
    <BlueprintAlert
      isOpen={open}
      canEscapeKeyCancel
      onClose={() => {
        setOpen(false)
        // https://blueprintjs.com/docs/#core/components/alert
        setTimeout(() => {
          root.unmount()
          container.remove()
        }, 300)
      }}
    >
      {error.includes('\n') ? (
        <pre className="mt-0" style={{ fontFamily: 'inherit' }}>
          {error}
        </pre>
      ) : (
        <p>{error}</p>
      )}
    </BlueprintAlert>
  )
}
