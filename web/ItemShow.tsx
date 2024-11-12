import {
  AnchorButton,
  Button,
  ButtonGroup,
  Classes,
  Colors,
  Divider,
  H2,
} from '@blueprintjs/core'
import type { Dispatch, RefObject, SetStateAction } from 'react'
import { Circle, ExternalLink, Star } from 'react-feather'
import type { Feed, Item, Status } from './types'
import { cn, iconProps, length, panelStyle, xfetch } from './utils'

export default function ItemShow({
  setStatus,
  setItems,
  selectedItemDetails,
  setSelectedItemDetails,
  contentRef,
  feedsById,
}: {
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  setItems: Dispatch<SetStateAction<Item[] | undefined>>
  selectedItemDetails: Item
  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById: Map<number, Feed>
}) {
  const toggleStatus = (targetStatus: string) => async () => {
    const status = targetStatus === selectedItemDetails.status ? 'read' : targetStatus
    await xfetch(`api/items/${selectedItemDetails.id}`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    })
    const diff = (s: string) =>
      status === s ? +1 : selectedItemDetails.status === s ? -1 : 0
    setStatus(
      status =>
        status && {
          ...status,
          state: new Map([
            ...status.state,
            [
              selectedItemDetails.feed_id,
              {
                starred:
                  (status.state.get(selectedItemDetails.feed_id)?.starred ?? 0) +
                  diff('starred'),
                unread:
                  (status.state.get(selectedItemDetails.feed_id)?.unread ?? 0) +
                  diff('unread'),
              },
            ],
          ]),
        },
    )
    setItems(items =>
      items?.map(i => (i.id === selectedItemDetails.id ? { ...i, status } : i)),
    )
    setSelectedItemDetails({ ...selectedItemDetails, status })
  }

  return (
    <div style={panelStyle}>
      <div style={{ display: 'flex', minHeight: length(10) }}>
        <ButtonGroup style={{ margin: length(1) }} minimal>
          <Button
            icon={
              <Star
                {...iconProps}
                fill={
                  selectedItemDetails.status === 'starred'
                    ? Colors.DARK_GRAY1
                    : Colors.WHITE
                }
              />
            }
            onClick={toggleStatus('starred')}
            title="Mark Starred"
          />
          <Button
            icon={
              <Circle
                {...iconProps}
                fill={
                  selectedItemDetails.status === 'unread'
                    ? Colors.DARK_GRAY1
                    : Colors.WHITE
                }
              />
            }
            onClick={toggleStatus('unread')}
            title="Mark Unread"
          />
          <AnchorButton
            className={Classes.INTENT_PRIMARY}
            icon={<ExternalLink {...iconProps} />}
            href={selectedItemDetails.link}
            target="_blank"
            title="Open Link"
          />
        </ButtonGroup>
      </div>
      <Divider />
      <div
        style={{ padding: length(5), overflow: 'auto', overflowWrap: 'break-word' }}
        ref={contentRef}
      >
        <H2 style={{ fontWeight: 700 }}>{selectedItemDetails.title || 'untitled'}</H2>
        <div style={{ opacity: 0.95 }}>
          {feedsById.get(selectedItemDetails.feed_id)?.title}
        </div>
        <div style={{ opacity: 0.95 }}>
          {new Date(selectedItemDetails.date).toLocaleString()}
        </div>
        <Divider style={{ marginTop: length(3), marginBottom: length(3) }} />
        <div
          style={{ fontSize: '1rem', lineHeight: '1.5rem' }}
          className={cn(Classes.RUNNING_TEXT, 'content')}
          dangerouslySetInnerHTML={{ __html: selectedItemDetails.content }}
        />
      </div>
    </div>
  )
}
