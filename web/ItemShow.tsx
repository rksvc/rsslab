import { AnchorButton, Button, ButtonGroup, ButtonVariant, Classes, Divider, H2 } from '@blueprintjs/core'
import { ChevronLeft, ChevronRight, Circle, ExternalLink, Star, X } from 'react-feather'
import { useMyContext } from './Context.tsx'
import type { Item } from './types.ts'
import { cn, length, xfetch } from './utils.ts'

export default function ItemShow() {
  const {
    selectedItemId,
    setSelectedItemId,
    setSelectedItem,
    items,
    contentRef,
    feedsById,
    updateStatus,
    updateItems,
    selectItem,
    selectedItem: item,
  } = useMyContext()
  if (!items || selectedItemId == null || !item) return undefined

  const toggleStatus = (target: Item['status']) => async () => {
    const status = target === item.status ? 'read' : target
    await xfetch(`api/items/${item.id}`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    })
    const diff = (s: Item['status']) => (status === s ? +1 : item.status === s ? -1 : 0)
    updateStatus(status => {
      const state = status?.state.get(item.feed_id)
      if (state) {
        state.starred += diff('starred')
        state.unread += diff('unread')
      }
    })
    updateItems(items => {
      for (const i of items?.list ?? []) if (i.id === item.id) i.status = status
    })
    setSelectedItem(item => item && { ...item, status })
  }
  const shift = (shift: number) => {
    const index = items.list.findIndex(item => item.id === selectedItemId)
    if (index === -1) {
      if (items.list.length) selectItem(items.list[0])
    } else selectItem(items.list[index + shift])
  }

  return (
    <div id="item">
      <ButtonGroup className="topbar" style={{ gap: length(1) }} variant={ButtonVariant.MINIMAL}>
        <Button
          icon={<Star fill={item.status === 'starred' ? 'currentColor' : 'transparent'} />}
          onClick={toggleStatus('starred')}
          title="Mark Starred"
        />
        <Button
          icon={<Circle fill={item.status === 'unread' ? 'currentColor' : 'transparent'} />}
          onClick={toggleStatus('unread')}
          title="Mark Unread"
        />
        <AnchorButton
          icon={<ExternalLink />}
          href={item.link}
          target="_blank"
          title="Open Link"
          rel="noopener noreferrer"
          referrerPolicy="no-referrer"
        />
        <div style={{ flexGrow: 1 }} />
        <Button
          icon={<ChevronLeft />}
          title={'Previous Article'}
          variant={ButtonVariant.MINIMAL}
          disabled={!items.list.length || selectedItemId === items.list.at(0)?.id}
          onClick={() => shift(-1)}
        />
        <Button
          icon={<ChevronRight />}
          title={'Next Article'}
          variant={ButtonVariant.MINIMAL}
          disabled={!items.list.length || selectedItemId === items.list.at(-1)?.id}
          onClick={() => shift(+1)}
        />
        <Button
          icon={<X />}
          title={'Close Article'}
          variant={ButtonVariant.MINIMAL}
          onClick={() => {
            setSelectedItemId(undefined)
            setSelectedItem(undefined)
          }}
        />
      </ButtonGroup>
      <Divider compact />
      <div style={{ padding: length(5), overflow: 'auto', overflowWrap: 'break-word' }} ref={contentRef}>
        <H2 style={{ fontWeight: 700 }}>{item.title || 'untitled'}</H2>
        <div style={{ opacity: 0.95 }}>{feedsById?.get(item.feed_id)?.title}</div>
        <div style={{ opacity: 0.95 }}>{new Date(item.date).toLocaleString()}</div>
        <Divider compact style={{ marginTop: length(3), marginBottom: length(3) }} />
        <div
          style={{ fontSize: '1rem', lineHeight: '1.5rem' }}
          className={cn(Classes.RUNNING_TEXT, 'content')}
          dangerouslySetInnerHTML={{ __html: item.content }}
        />
      </div>
    </div>
  )
}
