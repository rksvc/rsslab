import { AnchorButton, Button, ButtonGroup, Classes, Divider, H2 } from '@blueprintjs/core'
import type { CSSProperties, Dispatch, RefObject, SetStateAction } from 'react'
import { Circle, ExternalLink, Star } from 'react-feather'
import type { Feed, Item, Items, Status } from './types.ts'
import { cn, length, xfetch } from './utils.ts'

export default function ItemShow({
  style,
  setStatus,
  setItems,
  selectedItem,
  setSelectedItem,
  contentRef,
  feedsById,
}: {
  style: CSSProperties
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  setItems: Dispatch<SetStateAction<Items | undefined>>
  selectedItem: Item & { content: string }
  setSelectedItem: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById?: Map<number, Feed>
}) {
  const toggleStatus = (target: Item['status']) => async () => {
    const status = target === selectedItem.status ? 'read' : target
    await xfetch(`api/items/${selectedItem.id}`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    })
    const diff = (s: Item['status']) => (status === s ? +1 : selectedItem.status === s ? -1 : 0)
    setStatus(status => {
      if (!status) return
      const state = new Map(status.state)
      const s = state.get(selectedItem.feed_id)
      if (s)
        state.set(selectedItem.feed_id, {
          ...s,
          starred: s.starred + diff('starred'),
          unread: s.unread + diff('unread'),
        })
      return { ...status, state }
    })
    setItems(
      items =>
        items && {
          list: items.list.map(i => (i.id === selectedItem.id ? { ...i, status } : i)),
          has_more: items.has_more,
        },
    )
    setSelectedItem({ ...selectedItem, status })
  }

  return (
    <div style={style}>
      <ButtonGroup className="topbar" style={{ gap: length(0.5) }} minimal>
        <Button
          icon={<Star fill={selectedItem.status === 'starred' ? 'currentColor' : 'transparent'} />}
          onClick={toggleStatus('starred')}
          title="Mark Starred"
        />
        <Button
          icon={<Circle fill={selectedItem.status === 'unread' ? 'currentColor' : 'transparent'} />}
          onClick={toggleStatus('unread')}
          title="Mark Unread"
        />
        <AnchorButton
          icon={<ExternalLink />}
          href={selectedItem.link}
          target="_blank"
          title="Open Link"
          rel="noopener noreferrer"
          referrerPolicy="no-referrer"
        />
      </ButtonGroup>
      <Divider />
      <div style={{ padding: length(5), overflow: 'auto', wordBreak: 'break-all' }} ref={contentRef}>
        <H2 style={{ fontWeight: 700 }}>{selectedItem.title || 'untitled'}</H2>
        <div style={{ opacity: 0.95 }}>{feedsById?.get(selectedItem.feed_id)?.title}</div>
        <div style={{ opacity: 0.95 }}>{new Date(selectedItem.date).toLocaleString()}</div>
        <Divider style={{ marginTop: length(3), marginBottom: length(3) }} />
        <div
          style={{ fontSize: '1rem', lineHeight: '1.5rem' }}
          className={cn(Classes.RUNNING_TEXT, 'content')}
          dangerouslySetInnerHTML={{ __html: selectedItem.content }}
        />
      </div>
    </div>
  )
}
