import {
  Button,
  Card,
  CardList,
  Classes,
  Divider,
  Icon,
  InputGroup,
  Intent,
  Spinner,
  SpinnerSize,
} from '@blueprintjs/core'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import {
  type Dispatch,
  type MutableRefObject,
  type RefObject,
  type SetStateAction,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { Check, Search } from 'react-feather'
import useInfiniteScroll from 'react-infinite-scroll-hook'
import { useDebouncedCallback } from 'use-debounce'
import type { Feed, Image, Item, Items, Status } from './types'
import { cn, iconProps, param, xfetch } from './utils'

dayjs.extend(relativeTime)

export default function ItemList({
  filter,
  setStatus,
  errors,
  selected,

  items,
  setItems,
  selectedItemId,
  setSelectedItemId,

  setSelectedItemDetails,
  contentRef,

  refreshStats,
  feedsById,
}: {
  filter: string
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selected: string
  errors?: Map<number, string>

  items?: (Item & Image)[]
  setItems: Dispatch<SetStateAction<(Item & Image)[] | undefined>>
  selectedItemId?: number
  setSelectedItemId: Dispatch<SetStateAction<number | undefined>>

  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>

  refreshStats: (loop?: boolean) => Promise<void>
  feedsById: Map<number, Feed>
}) {
  const [search, setSearch] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const itemListRef = useRef<HTMLDivElement>(null)
  const loaded = useRef<boolean[]>()
  const [sentryRef] = useInfiniteScroll({
    disabled: false,
    loading: loading,
    hasNextPage: hasMore,
    rootMargin: '0px 0px 400px 0px',
    onLoadMore: async () => {
      if (!items) return
      setLoading(true)
      try {
        const result = await xfetch<Items>(
          `api/items${param({ ...query(), after: items.at(-1)?.id })}`,
        )
        setItems([...items, ...result.list])
        setHasMore(result.has_more)
      } finally {
        setLoading(false)
      }
    },
  })

  const [type, s] = selected.split(':')
  const id = Number.parseInt(s)
  const isFeedSelected = type === 'feed'
  const query = useCallback(() => {
    const query: Record<string, string | boolean> = {}
    if (selected) {
      const [type, id] = selected.split(':')
      query[`${type}_id`] = id
    }
    if (filter !== 'Feeds') query.status = filter.toLowerCase()
    if (filter === 'Unread') query.oldest_first = true
    const search = inputRef.current?.value
    if (search) query.search = search
    return query
  }, [selected, filter])
  const onSearch = useDebouncedCallback(async () => {
    const result = await xfetch<Items>(`api/items${param(query())}`)
    setItems(result.list)
    setHasMore(result.has_more)
    loaded.current = Array.from({ length: result.list.length })
  }, 200)

  useEffect(() => {
    ;(async () => {
      const result = await xfetch<Items>(`api/items${param(query())}`)
      setItems(result.list)
      setSelectedItemId(undefined)
      setSelectedItemDetails(undefined)
      setHasMore(result.has_more)
      loaded.current = Array.from({ length: result.list.length })
      itemListRef.current?.scrollTo(0, 0)
    })()
  }, [query, setItems, setSelectedItemId, setSelectedItemDetails])

  return (
    <div className="flex flex-col min-h-screen max-h-screen">
      <div className="flex flex-row items-center min-h-10 max-h-10">
        <InputGroup
          inputRef={inputRef}
          className="mx-1"
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
          className="mr-1"
          icon={<Check {...iconProps} />}
          title="Mark All Read"
          disabled={filter === 'Starred'}
          minimal
          onClick={async () => {
            const query: Record<string, string> = {}
            if (selected) {
              const [type, id] = selected.split(':')
              query[`${type}_id`] = id
            }
            await xfetch(`api/items${param(query)}`, { method: 'PUT' })
            setItems(items =>
              items?.map(item => ({
                ...item,
                status: item.status === 'starred' ? 'starred' : 'read',
              })),
            )
            await refreshStats(false)
          }}
        />
      </div>
      <Divider className="m-0" />
      <CardList className="grow" ref={itemListRef} bordered={false} compact>
        {items?.map((item, i) => (
          <CardItem
            key={item.id}
            item={item}
            i={i}
            loaded={loaded}
            setStatus={setStatus}
            setItems={setItems}
            selectedItemId={selectedItemId}
            setSelectedItemId={setSelectedItemId}
            setSelectedItemDetails={setSelectedItemDetails}
            contentRef={contentRef}
            feedsById={feedsById}
          />
        ))}
        {(loading || hasMore) && (
          <div className="mt-4 mb-3" ref={sentryRef}>
            <Spinner size={SpinnerSize.SMALL} />
          </div>
        )}
      </CardList>
      {isFeedSelected && errors?.get(id) && (
        <>
          <Divider className="m-0" />
          <div className="p-3 break-words text-red-600">{errors?.get(id)}</div>
        </>
      )}
    </div>
  )
}

function CardItem({
  item,
  i,
  loaded,
  setStatus,
  setItems,
  selectedItemId,
  setSelectedItemId,
  setSelectedItemDetails,
  contentRef,
  feedsById,
}: {
  item: Item & Image
  i: number
  loaded: MutableRefObject<boolean[] | undefined>
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  setItems: Dispatch<SetStateAction<(Item & Image)[] | undefined>>
  selectedItemId?: number
  setSelectedItemId: Dispatch<SetStateAction<number | undefined>>
  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>
  feedsById: Map<number, Feed>
}) {
  const previousStatus = usePrevious(item.status)
  const onLoad = () => {
    if (loaded.current && !loaded.current[i]) {
      loaded.current[i] = true
      setItems(items =>
        items?.map(i => (i.id === item.id ? { ...item, loaded: true } : i)),
      )
    }
  }

  const selected = item.id === selectedItemId
  return (
    <Card
      className="w-full"
      selected={selected}
      interactive
      onClick={async () => {
        if (selected) return
        setSelectedItemId(item.id)
        setSelectedItemDetails(await xfetch<Item>(`api/items/${item.id}`))
        contentRef.current?.scrollTo(0, 0)
        if (item.status === 'unread') {
          await xfetch(`api/items/${item.id}`, {
            method: 'PUT',
            body: JSON.stringify({ status: 'read' }),
          })
          setStatus(
            status =>
              status && {
                ...status,
                stats: new Map([
                  ...status.stats,
                  [
                    item.feed_id,
                    {
                      starred: status.stats.get(item.feed_id)?.starred ?? 0,
                      unread: (status.stats.get(item.feed_id)?.unread ?? 0) - 1,
                    },
                  ],
                ]),
              },
          )
          setItems(items =>
            items?.map(i => (i.id === item.id ? { ...i, status: 'read' } : i)),
          )
          setSelectedItemDetails(item => item && { ...item, status: 'read' })
        }
      }}
    >
      <div className="flex flex-row w-full">
        {item.image && (
          <div
            className={cn(
              'flex h-full mr-2 my-2 min-w-[80px] max-w-[80px]',
              !item.loaded && 'bp5-skeleton',
            )}
          >
            <img
              ref={image => image?.complete && onLoad()}
              className="w-full aspect-square object-cover rounded-lg"
              src={item.image}
              onLoad={onLoad}
            />
          </div>
        )}
        <div className="flex flex-col grow min-w-0">
          <div className="flex flex-row items-center opacity-70">
            <Icon
              svgProps={{
                className: cn(
                  'flex items-center transition-all',
                  item.status === 'read' ? 'w-0' : 'mr-1',
                ),
              }}
              icon={
                (item.status === 'read' ? previousStatus : item.status) === 'unread'
                  ? 'record'
                  : 'star'
              }
              size={10}
              intent={selected ? Intent.NONE : Intent.PRIMARY}
            />
            <small className="truncate grow">{feedsById.get(item.feed_id)?.title}</small>
            <small className="whitespace-nowrap ml-2">
              <time dateTime={item.date} title={new Date(item.date).toLocaleString()}>
                {dayjs(item.date).fromNow(true)}
              </time>
            </small>
          </div>
          <span className="mb-0.5 break-words">
            {item.title.length > 100
              ? `${item.title.slice(0, 100)}...`
              : item.title || 'untitled'}
          </span>
        </div>
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
