import { Button, Card, CardList, Classes, Divider, InputGroup, Spinner, SpinnerSize } from '@blueprintjs/core'
import { Record, type SVGIconProps, Star } from '@blueprintjs/icons'
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
import { Check, RotateCw, Search } from 'react-feather'
import type { Feed, Filter, FolderWithFeeds, Item, ItemStatus, Items, Selected, Status } from './types.ts'
import { fromNow, iconProps, length, panelStyle, param, xfetch } from './utils.ts'

export default function ItemList({
  style,

  filter,
  status,
  setStatus,
  selected,

  items,
  setItems,
  itemsOutdated,
  setItemsOutdated,
  selectedItem,
  setSelectedItem,

  contentRef,

  foldersById,
  feedsById,
}: {
  style?: CSSProperties

  filter: Filter
  status?: Status
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selected: Selected

  items?: Items
  setItems: Dispatch<SetStateAction<Items | undefined>>
  itemsOutdated: boolean
  setItemsOutdated: Dispatch<SetStateAction<boolean>>
  selectedItem?: Item
  setSelectedItem: Dispatch<SetStateAction<Item | undefined>>

  contentRef: RefObject<HTMLDivElement>

  foldersById: Map<number, FolderWithFeeds>
  feedsById: Map<number, Feed>
}) {
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [itemsInited, setItemsInited] = useState(false)
  const [lastUnread, setLastUnread] = useState<number>()
  const timerId = useRef<ReturnType<typeof setTimeout>>()
  const inputRef = useRef<HTMLInputElement>(null)
  const itemListRef = useRef<HTMLDivElement>(null)

  const query = useCallback(
    (status?: ItemStatus) => {
      const query: Record<string, string | boolean> = {}
      if (selected) Object.assign(query, selected)
      if (status) query.status = status
      else if (filter !== 'Feeds') query.status = filter.toLowerCase()
      if (query.status === 'unread') query.oldest_first = true
      const search = inputRef.current?.value
      if (search) query.search = search
      return query
    },
    [selected, filter],
  )

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
  const prevQuery = usePrevious(query)
  const needInitReadItems =
    items &&
    query === prevQuery &&
    itemsInited &&
    filter === 'Unread' &&
    selected?.feed_id != null &&
    lastUnread == null
  if (!loading && isIntersecting && (items?.has_more || needInitReadItems)) {
    ;(async () => {
      setLoading(true)
      try {
        if (items.has_more) {
          const status = lastUnread == null ? undefined : 'read'
          const { list, has_more } = await xfetch<Items>(
            `api/items${param({ ...query(status), after: items.list.at(-1)?.id })}`,
          )
          setItems({ list: [...items.list, ...list], has_more })
        } else {
          const { list, has_more } = await xfetch<Items>(`api/items${param(query('read'))}`)
          setLastUnread(items.list.length)
          setItems({ list: [...items.list, ...list], has_more })
        }
      } finally {
        setLoading(false)
        setIsIntersecting(false)
      }
    })()
  }

  const refresh = useCallback(async () => {
    setItems(await xfetch<Items>(`api/items${param(query())}`))
    setItemsInited(true)
    setLastUnread(undefined)
    setSelectedItem(undefined)
    setItemsOutdated(false)
    itemListRef.current?.scrollTo(0, 0)
  }, [query, setItems, setSelectedItem, setItemsOutdated])
  useEffect(() => {
    refresh()
    return () => setItemsInited(false)
  }, [refresh])

  const feedError = selected?.feed_id != null && status?.state.get(selected.feed_id)?.error
  const readItems = lastUnread == null ? null : items?.list.slice(lastUnread)
  return (
    <div style={{ ...style, ...panelStyle }}>
      <div className="topbar" style={{ gap: length(1) }}>
        <InputGroup
          inputRef={inputRef}
          leftIcon={<Search style={{ pointerEvents: 'none' }} className={Classes.ICON} {...iconProps} />}
          type="search"
          value={search}
          placeholder="Search..."
          onValueChange={value => {
            setSearch(value)
            clearTimeout(timerId.current)
            timerId.current = setTimeout(async () => {
              timerId.current = undefined
              setItems(await xfetch<Items>(`api/items${param(query())}`))
              setLastUnread(undefined)
              setItemsOutdated(false)
            }, 200)
          }}
          fill
        />
        <Button
          icon={
            !itemsOutdated || filter === 'Starred' ? (
              <Check {...iconProps} />
            ) : (
              <RotateCw {...iconProps} strokeWidth={1.7} />
            )
          }
          title={!itemsOutdated || filter === 'Starred' ? 'Mark All Read' : 'Refresh Outdated'}
          disabled={filter === 'Starred'}
          minimal
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
                    const feeds = new Set(foldersById.get(selected.folder_id)?.feeds.map(feed => feed.id))
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
          }}
        />
      </div>
      <Divider />
      <CardList style={{ flexGrow: 1 }} ref={itemListRef} bordered={false} compact>
        {items?.list.slice(0, lastUnread).map(item => (
          <CardItem
            key={item.id}
            item={item}
            setStatus={setStatus}
            setItems={setItems}
            selectedItem={selectedItem}
            setSelectedItem={setSelectedItem}
            contentRef={contentRef}
            feedsById={feedsById}
          />
        ))}
        {!!readItems?.length && (
          <>
            <div
              style={{ display: 'flex', alignItems: 'center', columnGap: length(2), marginTop: length(1) }}
            >
              <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1 }}>
                <Divider />
              </div>
              <span style={{ opacity: 0.9, fontSize: '0.9em' }}>read items</span>
              <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1 }}>
                <Divider />
              </div>
            </div>
            {readItems.map(item => (
              <CardItem
                key={item.id}
                item={item}
                setStatus={setStatus}
                setItems={setItems}
                selectedItem={selectedItem}
                setSelectedItem={setSelectedItem}
                contentRef={contentRef}
                feedsById={feedsById}
                style={{ opacity: 0.85 }}
              />
            ))}
          </>
        )}
        {(loading || items?.has_more || needInitReadItems) && (
          <div
            style={{ marginTop: length(4), marginBottom: length(3) }}
            ref={node => {
              if (node) {
                sentryNodeRef.current = node
                observer.current?.observe(node)
              } else if (sentryNodeRef.current) {
                observer.current?.unobserve(sentryNodeRef.current)
              }
            }}
          >
            <Spinner size={SpinnerSize.SMALL} />
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
  style,
}: {
  item: Item
  setItems: Dispatch<SetStateAction<Items | undefined>>
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selectedItem?: Item
  setSelectedItem: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById: Map<number, Feed>
  style?: CSSProperties
}) {
  const prevStatus = usePrevious(item.status)
  const isSelected = item.id === selectedItem?.id
  const iconProps = {
    style: { display: 'flex', width: '100%' },
    className: isSelected ? undefined : Classes.INTENT_PRIMARY,
  } satisfies SVGIconProps
  return (
    <Card
      selected={isSelected}
      style={style}
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
              ...(item.status === 'read' ? { width: 0 } : { width: '10px', marginRight: length(1) }),
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
          {item.title.length > 100 ? `${item.title.slice(0, 100)}...` : item.title || 'untitled'}
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
