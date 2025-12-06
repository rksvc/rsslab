import { Card, Divider, Intent, OverlayToaster, Position } from '@blueprintjs/core'
import { useEffect, useRef } from 'react'
import { useMyContext } from './Context.tsx'
import FeedList from './FeedList.tsx'
import ItemList from './ItemList.tsx'
import ItemShow from './ItemShow.tsx'
import { cn } from './utils.ts'

export default function App() {
  const caughtErrors = useRef(new Set<any>())
  const toaster = useRef<OverlayToaster>(null)
  useEffect(() => {
    window.addEventListener('unhandledrejection', evt => {
      if (!caughtErrors.current.has(evt.reason)) {
        caughtErrors.current.add(evt.reason)
        const message = evt.reason instanceof Error ? evt.reason.message : String(evt.reason)
        toaster.current?.show({
          intent: Intent.DANGER,
          message: message.split('\n', 2)[0],
          timeout: 0,
        })
      }
    })
  }, [])

  const { selected, selectedItemIndex } = useMyContext()
  return (
    <Card
      className={cn(selected !== undefined && 'feed-selected', selectedItemIndex != null && 'item-selected')}
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
      <OverlayToaster canEscapeKeyClear={false} position={Position.BOTTOM_RIGHT} ref={toaster} />
    </Card>
  )
}
