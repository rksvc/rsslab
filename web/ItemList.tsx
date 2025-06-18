import {
  Button,
  ButtonVariant,
  Card,
  CardList,
  Classes,
  Divider,
  InputGroup,
  Spinner,
} from '@blueprintjs/core'
import {
  type CSSProperties,
  type Dispatch,
  type RefObject,
  type SetStateAction,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { Check, Circle, RotateCw, Search, Star } from 'react-feather'
import type { Feed, Filter, FolderWithFeeds, Item, Items, Selected, Status } from './types.ts'
import { fromNow, length, param, xfetch } from './utils.ts'

export default function ItemList({
  style,

  items,
  setItems,
  status,
  setStatus,
  selectedItem,
  setSelectedItem,

  filter,
  selected,
  itemsOutdated,
  setItemsOutdated,
  contentRef,

  foldersById,
  feedsById,
}: {
  style: CSSProperties

  items?: Items
  setItems: Dispatch<SetStateAction<Items | undefined>>
  status?: Status
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selectedItem?: Item
  setSelectedItem: Dispatch<SetStateAction<Item | undefined>>

  filter: Filter
  selected: Selected
  itemsOutdated: boolean
  setItemsOutdated: Dispatch<SetStateAction<boolean>>
  contentRef: RefObject<HTMLDivElement>

  foldersById?: Map<number, FolderWithFeeds>
  feedsById?: Map<number, Feed>
}) {
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const timerId = useRef<ReturnType<typeof setTimeout>>()
  const searchRef = useRef<HTMLInputElement>(null)
  const itemListRef = useRef<HTMLDivElement>(null)

  const query = useCallback(() => {
    const query: Record<string, string | boolean> = {}
    if (selected) Object.assign(query, selected)
    if (filter !== 'Feeds') query.status = filter.toLowerCase()
    if (filter === 'Unread') query.oldest_first = true
    const search = searchRef.current?.value
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
        if (entry.target === sentryNodeRef.current && entry.isIntersecting) setIsIntersecting(true)
    })
  }
  if (!loading && isIntersecting && items?.has_more) {
    // biome-ignore lint/nursery/noFloatingPromises: expected
    ;(async () => {
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

  const refresh = useCallback(async () => {
    setItems(await xfetch<Items>(`api/items${param(query())}`))
    setSelectedItem(undefined)
    setItemsOutdated(false)
    itemListRef.current?.scrollTo(0, 0)
  }, [query, setItems, setSelectedItem, setItemsOutdated])
  useEffect(() => {
    // biome-ignore lint/nursery/noFloatingPromises: expected
    refresh()
  }, [refresh])

  const feedError = selected?.feed_id != null && status?.state.get(selected.feed_id)?.error
  return (
    <div style={style}>
      <div className="topbar" style={{ gap: length(1) }}>
        <InputGroup
          inputRef={searchRef}
          leftIcon={<Search style={{ pointerEvents: 'none' }} className={Classes.ICON} />}
          type="search"
          value={search}
          placeholder="Search..."
          onValueChange={value => {
            setSearch(value)
            clearTimeout(timerId.current)
            timerId.current = setTimeout(async () => {
              timerId.current = undefined
              setItems(await xfetch<Items>(`api/items${param(query())}`))
              setItemsOutdated(false)
            }, 200)
          }}
          fill
        />
        <Button
          icon={itemsOutdated ? <RotateCw strokeWidth={1.7} /> : <Check />}
          title={itemsOutdated ? 'Refresh Outdated' : 'Mark All Read'}
          disabled={filter === 'Starred' && !itemsOutdated}
          variant={ButtonVariant.MINIMAL}
          onClick={async () => {
            if (itemsOutdated) return await refresh()
            const after = items?.list.at(0)?.id
            if (after == null) return
            await xfetch(`api/items${param({ ...query(), after })}`, { method: 'PUT' })
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
            const isSelected = !selected
              ? (_: number) => true
              : selected.feed_id != null
                ? (id: number) => id === selected.feed_id
                : (() => {
                    const feeds = new Set(foldersById?.get(selected.folder_id)?.feeds.map(feed => feed.id))
                    return (id: number) => feeds.has(id)
                  })()
            setStatus(
              status =>
                status && {
                  ...status,
                  state: new Map(
                    status.state.entries().map(([id, state]) => [
                      id,
                      isSelected(id)
                        ? {
                            ...state,
                            unread: 0,
                          }
                        : state,
                    ]),
                  ),
                },
            )
            setSelectedItem(item => item && (item.status === 'unread' ? { ...item, status: 'read' } : item))
          }}
        />
      </div>
      <Divider />
      <CardList style={{ flexGrow: 1 }} ref={itemListRef} bordered={false} compact>
        {items?.list.map(item => (
          <CardItem
            key={item.id}
            item={item}
            setItems={setItems}
            setStatus={setStatus}
            selectedItem={selectedItem}
            setSelectedItem={setSelectedItem}
            contentRef={contentRef}
            feedsById={feedsById}
          />
        ))}
        {(loading || items?.has_more) && (
          <div
            style={{ marginTop: length(3.5), marginBottom: length(3) }}
            ref={node => {
              if (node) {
                sentryNodeRef.current = node
                observer.current?.observe(node)
              } else if (sentryNodeRef.current) {
                observer.current?.unobserve(sentryNodeRef.current)
              }
            }}
          >
            <Spinner className="loading-items" size={18} />
          </div>
        )}
      </CardList>
      {feedError && (
        <>
          <Divider />
          <div
            style={{
              padding: length(3),
              overflowWrap: 'break-word',
              color: 'var(--danger)',
            }}
          >
            {feedError}
          </div>
        </>
      )}
    </div>
  )
}

function CardItem({
  item,
  setItems,
  setStatus,
  selectedItem,
  setSelectedItem,
  contentRef,
  feedsById,
}: {
  item: Item
  setItems: Dispatch<SetStateAction<Items | undefined>>
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selectedItem?: Item
  setSelectedItem: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById?: Map<number, Feed>
}) {
  const prevStatus = usePrevious(item.status)
  const isSelected = item.id === selectedItem?.id
  return (
    <Card
      selected={isSelected}
      interactive
      onClick={async () => {
        if (isSelected) return
        setSelectedItem(await xfetch<Item>(`api/items/${item.id}`))
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
                list: items.list.map(i => (i.id === item.id ? { ...i, status: 'read' } : i)),
                has_more: items.has_more,
              },
          )
          setSelectedItem(item => item && { ...item, status: 'read' })
        }
      }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', width: '100%', minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', opacity: 0.7 }}>
          <span
            style={{
              transitionDuration: '150ms',
              ...(item.status === 'read'
                ? { width: 0 }
                : { width: '10px', marginRight: length(1), flexShrink: 0 }),
            }}
          >
            {(item.status === 'read' ? prevStatus : item.status) === 'unread' ? (
              <Circle style={{ transform: 'scale(0.8)' }} />
            ) : (
              <Star />
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
            {feedsById?.get(item.feed_id)?.title}
          </small>
          <small style={{ whiteSpace: 'nowrap', marginLeft: length(2) }}>
            <time dateTime={item.date} title={new Date(item.date).toLocaleString()}>
              {fromNow(new Date(item.date), false)}
            </time>
          </small>
        </div>
        <span
          // https://tailwindcss.com/docs/line-clamp
          style={{
            marginBottom: length(0.5),
            overflowWrap: 'break-word',
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 3,
          }}
        >
          {item.title}
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
