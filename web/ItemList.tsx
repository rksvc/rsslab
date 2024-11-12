import {
  Button,
  Card,
  CardList,
  Classes,
  Colors,
  Divider,
  InputGroup,
  Spinner,
  SpinnerSize,
} from '@blueprintjs/core'
import { Record, type SVGIconProps, Star } from '@blueprintjs/icons'
import {
  type Dispatch,
  type RefObject,
  type SetStateAction,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { Check, RotateCw, Search } from 'react-feather'
import type { Feed, Item, Items, Selected, Status } from './types'
import { fromNow, iconProps, length, panelStyle, param, xfetch } from './utils'

export default function ItemList({
  filter,
  status,
  setStatus,
  selected,

  items,
  setItems,
  itemsOutdated,
  setItemsOutdated,
  selectedItemId,
  setSelectedItemId,

  setSelectedItemDetails,
  contentRef,

  refreshStats,
  feedsById,
}: {
  filter: string
  status?: Status
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selected: Selected

  items?: Items
  setItems: Dispatch<SetStateAction<Items | undefined>>
  itemsOutdated: boolean
  setItemsOutdated: Dispatch<SetStateAction<boolean>>
  selectedItemId?: number
  setSelectedItemId: Dispatch<SetStateAction<number | undefined>>

  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>

  refreshStats: (loop?: boolean) => Promise<void>
  feedsById: Map<number, Feed>
}) {
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const itemListRef = useRef<HTMLDivElement>(null)

  const query = useCallback(() => {
    const query: Record<string, string | boolean> = {}
    if (selected) Object.assign(query, selected)
    if (filter !== 'Feeds') query.status = filter.toLowerCase()
    if (filter === 'Unread') query.oldest_first = true
    const search = inputRef.current?.value
    if (search) query.search = search
    return query
  }, [selected, filter])

  const sentryNodeRef = useRef<Element>()
  const [isIntersecting, setIsIntersecting] = useState(false)
  // https://react.dev/reference/react/useRef#avoiding-recreating-the-ref-contents
  const observer = useRef<IntersectionObserver>()
  if (!observer.current) {
    observer.current = new IntersectionObserver(entries => {
      for (const entry of entries)
        if (entry.target === sentryNodeRef.current && entry.isIntersecting)
          setIsIntersecting(true)
    })
  }
  if (!loading && isIntersecting && items?.has_more) {
    ;(async () => {
      if (!items) return
      setLoading(true)
      try {
        const { list, has_more } = await xfetch<Items>(
          `api/items${param({ ...query(), after: items.list.at(-1)?.id })}`,
        )
        setItems({ list: [...items.list, ...list], has_more })
      } finally {
        setLoading(false)
        setIsIntersecting(false)
      }
    })()
  }

  const timerId = useRef<number>()
  const onSearch = () => {
    clearTimeout(timerId.current)
    timerId.current = setTimeout(async () => {
      timerId.current = undefined
      setItems(await xfetch<Items>(`api/items${param(query())}`))
      setItemsOutdated(false)
    }, 200)
  }

  const refresh = useCallback(async () => {
    setItems(await xfetch<Items>(`api/items${param(query())}`))
    setSelectedItemId(undefined)
    setSelectedItemDetails(undefined)
    setItemsOutdated(false)
    itemListRef.current?.scrollTo(0, 0)
  }, [query, setItems, setSelectedItemId, setSelectedItemDetails, setItemsOutdated])
  useEffect(() => {
    refresh()
  }, [refresh])

  const error = selected?.feed_id != null && status?.state.get(selected.feed_id)?.error
  return (
    <div style={panelStyle}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          minHeight: length(10),
          paddingLeft: length(1),
          paddingRight: length(1),
        }}
      >
        <InputGroup
          inputRef={inputRef}
          leftIcon={<Search className={Classes.ICON} {...iconProps} />}
          type="search"
          value={search}
          onValueChange={value => {
            setSearch(value)
            onSearch()
          }}
          fill
        />
        <Button
          style={{
            marginLeft: length(1),
            color:
              filter === 'Starred'
                ? undefined
                : itemsOutdated
                  ? Colors.GRAY1
                  : Colors.DARK_GRAY5,
          }}
          icon={
            !itemsOutdated || filter === 'Starred' ? (
              <Check {...iconProps} />
            ) : (
              <RotateCw {...iconProps} />
            )
          }
          title={
            !itemsOutdated || filter === 'Starred' ? 'Mark All Read' : 'Refresh Outdated'
          }
          disabled={filter === 'Starred'}
          minimal
          onClick={async () => {
            if (itemsOutdated) return await refresh()
            await xfetch(`api/items${param(query())}`, { method: 'PUT' })
            setItems(
              items =>
                items && {
                  list: items.list.map(item => ({
                    ...item,
                    status: item.status === 'starred' ? 'starred' : 'read',
                  })),
                  has_more: items.has_more,
                },
            )
            await refreshStats(false)
          }}
        />
      </div>
      <Divider />
      <CardList style={{ flexGrow: 1 }} ref={itemListRef} bordered={false} compact>
        {items?.list.map(item => (
          <CardItem
            key={item.id}
            item={item}
            setStatus={setStatus}
            setItems={setItems}
            selectedItemId={selectedItemId}
            setSelectedItemId={setSelectedItemId}
            setSelectedItemDetails={setSelectedItemDetails}
            contentRef={contentRef}
            feedsById={feedsById}
          />
        ))}
        {(loading || items?.has_more) && (
          <div
            style={{ marginTop: length(4), marginBottom: length(3) }}
            ref={node => {
              if (node) {
                sentryNodeRef.current = node
                observer.current?.observe(node)
              } else {
                if (sentryNodeRef.current)
                  observer.current?.unobserve(sentryNodeRef.current)
              }
            }}
          >
            <Spinner size={SpinnerSize.SMALL} />
          </div>
        )}
      </CardList>
      {error && (
        <>
          <Divider />
          <div
            style={{ padding: length(3), overflowWrap: 'break-word', color: '#dc2626' }}
          >
            {error}
          </div>
        </>
      )}
    </div>
  )
}

function CardItem({
  item,
  setStatus,
  setItems,
  selectedItemId,
  setSelectedItemId,
  setSelectedItemDetails,
  contentRef,
  feedsById,
}: {
  item: Item
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  setItems: Dispatch<SetStateAction<Items | undefined>>
  selectedItemId?: number
  setSelectedItemId: Dispatch<SetStateAction<number | undefined>>
  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById: Map<number, Feed>
}) {
  const prevStatus = usePrevious(item.status)
  const isSelected = item.id === selectedItemId
  const iconProps = {
    style: { display: 'flex', width: '100%' },
    className: isSelected ? undefined : Classes.INTENT_PRIMARY,
  } satisfies SVGIconProps
  return (
    <Card
      selected={isSelected}
      interactive
      onClick={async () => {
        if (isSelected) return
        setSelectedItemId(item.id)
        setSelectedItemDetails(await xfetch<Item>(`api/items/${item.id}`))
        contentRef.current?.scrollTo(0, 0)
        if (item.status === 'unread') {
          await xfetch(`api/items/${item.id}`, {
            method: 'PUT',
            body: JSON.stringify({ status: 'read' }),
          })
          setStatus(status => {
            if (!status) return
            const state = new Map(status.state)
            const s = state.get(item.feed_id)
            if (s) state.set(item.feed_id, { ...s, unread: s.unread - 1 })
            return { ...status, state }
          })
          setItems(
            items =>
              items && {
                list: items.list.map(i =>
                  i.id === item.id ? { ...i, status: 'read' } : i,
                ),
                has_more: items.has_more,
              },
          )
          setSelectedItemDetails(item => item && { ...item, status: 'read' })
        }
      }}
    >
      <div
        style={{ display: 'flex', flexDirection: 'column', width: '100%', minWidth: 0 }}
      >
        <div style={{ display: 'flex', alignItems: 'center', opacity: 0.7 }}>
          <span
            style={{
              transitionDuration: '150ms',
              ...(item.status === 'read'
                ? { width: 0 }
                : { width: '10px', marginRight: length(1) }),
            }}
          >
            {(item.status === 'read' ? prevStatus : item.status) === 'unread' ? (
              <Record {...iconProps} />
            ) : (
              <Star {...iconProps} />
            )}
          </span>
          <small
            style={{
              flexGrow: 1,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {feedsById.get(item.feed_id)?.title}
          </small>
          <small style={{ whiteSpace: 'nowrap', marginLeft: length(2) }}>
            <time dateTime={item.date} title={new Date(item.date).toLocaleString()}>
              {fromNow(new Date(item.date))}
            </time>
          </small>
        </div>
        <span style={{ marginBottom: length(0.5), overflowWrap: 'break-word' }}>
          {item.title.length > 100
            ? `${item.title.slice(0, 100)}...`
            : item.title || 'untitled'}
        </span>
      </div>
    </Card>
  )
}

function usePrevious<T>(value: T) {
  const ref = useRef<T>()
  useEffect(() => {
    ref.current = value
  })
  return ref.current
}
