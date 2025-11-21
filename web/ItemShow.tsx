import { AnchorButton, Button, ButtonGroup, ButtonVariant, Classes, Divider, H2 } from '@blueprintjs/core'
import type { Dispatch, RefObject, SetStateAction } from 'react'
import { ChevronLeft, ChevronRight, Circle, ExternalLink, Star, X } from 'react-feather'
import type { Updater } from 'use-immer'
import type { Feed, Item, Items, Status } from './types.ts'
import { cn, length, xfetch } from './utils.ts'

export default function ItemShow({
  item,
  selectedItemIndex,
  setSelectedItemIndex,
  setSelectedItemContent,
  items,
  contentRef,
  feedsById,
  updateStatus,
  updateItems,
  selectItem,
}: {
  item: Item & { content: string }
  selectedItemIndex: number
  setSelectedItemIndex: Dispatch<SetStateAction<number | undefined>>
  setSelectedItemContent: Dispatch<SetStateAction<string | undefined>>
  items: Items
  contentRef: RefObject<HTMLDivElement>
  feedsById?: Map<number, Feed>
  updateStatus: Updater<Status | undefined>
  updateItems: Updater<Items | undefined>
  selectItem: (index: number) => Promise<void>
}) {
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
          disabled={selectedItemIndex === 0}
          onClick={() => selectItem(selectedItemIndex - 1)}
        />
        <Button
          icon={<ChevronRight />}
          title={'Next Article'}
          variant={ButtonVariant.MINIMAL}
          disabled={selectedItemIndex + 1 >= items.list.length}
          onClick={() => selectItem(selectedItemIndex + 1)}
        />
        <Button
          icon={<X />}
          title={'Close Article'}
          variant={ButtonVariant.MINIMAL}
          onClick={() => {
            setSelectedItemIndex(undefined)
            setSelectedItemContent(undefined)
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
