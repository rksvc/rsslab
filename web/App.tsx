import { Card, Dialog, DialogBody, Divider } from '@blueprintjs/core'
import { useEffect, useRef, useState } from 'react'
import { useMyContext } from './Context.tsx'
import FeedList from './FeedList.tsx'
import ItemList from './ItemList.tsx'
import ItemShow from './ItemShow.tsx'
import { cn } from './utils.ts'

export default function App() {
  const caughtErrors = useRef(new Set<any>())
  const [errors, setErrors] = useState<string[]>([])
  useEffect(() => {
    window.addEventListener('unhandledrejection', evt => {
      if (!caughtErrors.current.has(evt.reason)) {
        caughtErrors.current.add(evt.reason)
        const message = evt.reason instanceof Error ? evt.reason.message : String(evt.reason)
        setErrors(errors => [...errors, message])
      }
    })
  }, [])

  const { selected, selectedItemId } = useMyContext()
  return (
    <Card
      className={cn(selected !== undefined && 'feed-selected', selectedItemId != null && 'item-selected')}
      style={{
        padding: 0,
        height: '100vh',
        display: 'flex',
        fontSize: '1rem',
        lineHeight: '1.5rem',
        boxShadow: 'none',
        borderRadius: 0,
      }}
    >
      <FeedList />
      <Divider id="list-divider" compact />
      <ItemList />
      <Divider id="item-divider" compact />
      <ItemShow />
      {errors.map((error, i) => (
        <Dialog
          title="Error"
          className="error"
          key={error}
          onClose={() => setErrors(errors => errors.toSpliced(i, 1))}
          canEscapeKeyClose={false}
          isOpen
        >
          <DialogBody>{error}</DialogBody>
        </Dialog>
      ))}
    </Card>
  )
}
