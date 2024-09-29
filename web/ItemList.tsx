import {
  Button,
  Card,
  CardList,
  Classes,
  Divider,
  Icon,
  InputGroup,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  Popover,
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
import {
  Check,
  Edit,
  Folder as FolderIcon,
  FolderMinus,
  Link,
  MoreHorizontal,
  Move,
  RotateCw,
  Rss,
  Search,
  Trash,
} from 'react-feather'
import useInfiniteScroll from 'react-infinite-scroll-hook'
import { useDebouncedCallback } from 'use-debounce'
import { Confirm } from './Confirm'
import type { Feed, Folder, Image, Item, Items, Stats, Status } from './types'
import { cn, iconProps, menuIconProps, param, popoverProps, xfetch } from './utils'

dayjs.extend(relativeTime)

export default function ItemList({
  filter,
  folders,
  setFolders,
  setFeeds,
  setStats,
  errors,
  selected,
  setSelected,
  loadingFeeds,

  items,
  setItems,
  selectedItemId,
  setSelectedItemId,

  setSelectedItemDetails,
  contentRef,

  refreshFeeds,
  refreshStats,
  foldersById,
  feedsById,
}: {
  filter: string
  folders?: Folder[]
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>
  setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
  setStats: Dispatch<SetStateAction<Map<number, Stats> | undefined>>
  setSelected: Dispatch<SetStateAction<string>>
  selected: string
  errors?: Map<number, string>
  loadingFeeds: number

  items?: (Item & Image)[]
  setItems: Dispatch<SetStateAction<(Item & Image)[] | undefined>>
  selectedItemId?: number
  setSelectedItemId: Dispatch<SetStateAction<number | undefined>>

  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>
  contentRef: RefObject<HTMLDivElement>

  refreshFeeds: () => Promise<void>
  refreshStats: (loop?: boolean) => Promise<void>
  foldersById: Map<number, Folder>
  feedsById: Map<number, Feed>
}) {
  const [search, setSearch] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const [renameFeedDialogOpen, setRenameFeedDialogOpen] = useState(false)
  const [changeLinkDialogOpen, setChangeLinkDialogOpen] = useState(false)
  const [deleteFeedDialogOpen, setDeleteFeedDialogOpen] = useState(false)
  const [renameFolderDialogOpen, setRenameFolderDialogOpen] = useState(false)
  const [deleteFolderDialogOpen, setDeleteFolderDialogOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const feedTitleRef = useRef<HTMLInputElement>(null)
  const feedLinkRef = useRef<HTMLInputElement>(null)
  const folderTitleRef = useRef<HTMLInputElement>(null)
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
          `./api/items${param({ ...query(), after: items.at(-1)?.id })}`,
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
  const updateFeedAttr = async <T extends 'title' | 'feed_link' | 'folder_id'>(
    attrName: T,
    value: Feed[T],
  ) => {
    await xfetch(`./api/feeds/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ [attrName]: value ?? -1 }),
    })
    setFeeds(feeds =>
      feeds?.map(feed => (feed.id === id ? { ...feed, [attrName]: value } : feed)),
    )
  }
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
    const result = await xfetch<Items>(`./api/items${param(query())}`)
    setItems(result.list)
    setHasMore(result.has_more)
    loaded.current = Array.from({ length: result.list.length })
  }, 500)

  useEffect(() => {
    ;(async () => {
      const result = await xfetch<Items>(`./api/items${param(query())}`)
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
            await xfetch(`./api/items${param(query)}`, { method: 'PUT' })
            setItems(items =>
              items?.map(item => ({
                ...item,
                status: item.status === 'starred' ? 'starred' : 'read',
              })),
            )
            const status = await xfetch<Status>('./api/status')
            setStats(new Map(status.stats.map(stats => [stats.feed_id, stats])))
          }}
        />
        <Popover
          placement="bottom"
          {...popoverProps}
          content={
            <Menu>
              {isFeedSelected ? (
                <>
                  {feedsById.get(id)?.link && (
                    <MenuItem
                      text="Website"
                      icon={<Link {...menuIconProps} />}
                      target="_blank"
                      href={feedsById.get(id)?.link}
                    />
                  )}
                  <MenuItem
                    text="Feed Link"
                    icon={<Rss {...menuIconProps} />}
                    target="_blank"
                    href={(() => {
                      const feedLink = feedsById.get(id)?.feed_link
                      return feedLink?.startsWith('rsshub:')
                        ? `/${feedLink.slice('rsshub:'.length)}`
                        : feedLink
                    })()}
                  />
                  <MenuDivider />
                  <MenuItem
                    text="Rename"
                    icon={<Edit {...menuIconProps} />}
                    onClick={() => setRenameFeedDialogOpen(true)}
                  />
                  <MenuItem
                    text="Change Link"
                    icon={<Edit {...menuIconProps} />}
                    onClick={() => setChangeLinkDialogOpen(true)}
                  />
                  <MenuItem
                    text="Refresh"
                    icon={<RotateCw {...menuIconProps} />}
                    disabled={loadingFeeds > 0}
                    onClick={async () => {
                      await xfetch(`./api/feeds/${id}/refresh`, { method: 'POST' })
                      await refreshStats()
                    }}
                  />
                  <MenuItem
                    text="Move to..."
                    icon={<Move {...menuIconProps} />}
                    disabled={!folders?.length}
                  >
                    {folders
                      ?.filter(folder => folder.id !== feedsById.get(id)?.folder_id)
                      .map(folder => (
                        <MenuItem
                          key={folder.id}
                          text={folder.title}
                          icon={<FolderIcon {...menuIconProps} />}
                          onClick={() => updateFeedAttr('folder_id', folder.id)}
                        />
                      ))}
                    {feedsById.get(id)?.folder_id != null && (
                      <MenuItem
                        text="--"
                        icon={<FolderMinus {...menuIconProps} />}
                        onClick={() => updateFeedAttr('folder_id', null)}
                      />
                    )}
                  </MenuItem>
                  <MenuItem
                    text="Delete"
                    icon={<Trash {...menuIconProps} />}
                    intent={Intent.DANGER}
                    onClick={async () => setDeleteFeedDialogOpen(true)}
                  />
                </>
              ) : (
                <>
                  <MenuItem
                    text="Rename"
                    icon={<Edit {...menuIconProps} />}
                    onClick={() => setRenameFolderDialogOpen(true)}
                  />
                  <MenuItem
                    text="Refresh"
                    icon={<RotateCw {...menuIconProps} />}
                    disabled={loadingFeeds > 0}
                    onClick={async () => {
                      await xfetch(`./api/folders/${id}/refresh`, { method: 'POST' })
                      await refreshStats()
                    }}
                  />
                  <MenuItem
                    text="Delete"
                    icon={<Trash {...menuIconProps} />}
                    intent={Intent.DANGER}
                    onClick={async () => setDeleteFolderDialogOpen(true)}
                  />
                </>
              )}
            </Menu>
          }
        >
          <Button
            className="mr-1"
            icon={<MoreHorizontal {...iconProps} />}
            title={
              type === 'feed'
                ? 'Feed Settings'
                : type === 'folder'
                  ? 'Folder Settings'
                  : ''
            }
            disabled={!selected}
            minimal
          />
        </Popover>
      </div>
      <Divider className="m-0" />
      <CardList className="grow" ref={itemListRef} bordered={false} compact>
        {items?.map((item, i) => (
          <CardItem
            key={item.id}
            item={item}
            i={i}
            loaded={loaded}
            setStats={setStats}
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
      <Confirm
        open={renameFeedDialogOpen}
        setOpen={setRenameFeedDialogOpen}
        title="Rename Feed"
        callback={async () => {
          const title = feedTitleRef.current?.value
          if (title && title !== feedsById.get(id)?.title)
            await updateFeedAttr('title', title)
        }}
      >
        <InputGroup defaultValue={feedsById.get(id)?.title} inputRef={feedTitleRef} />
      </Confirm>
      <Confirm
        open={changeLinkDialogOpen}
        setOpen={setChangeLinkDialogOpen}
        title="Change Feed Link"
        callback={async () => {
          const feedLink = feedLinkRef.current?.value
          if (feedLink && feedLink !== feedsById.get(id)?.feed_link)
            await updateFeedAttr('feed_link', feedLink)
        }}
      >
        <InputGroup
          defaultValue={feedsById.get(id)?.feed_link}
          inputRef={feedLinkRef}
          spellCheck={false}
        />
      </Confirm>
      <Confirm
        open={deleteFeedDialogOpen}
        setOpen={setDeleteFeedDialogOpen}
        title="Delete Feed"
        callback={async () => {
          await xfetch(`./api/feeds/${id}`, { method: 'DELETE' })
          const folderId = feedsById.get(id)?.folder_id
          await Promise.all([refreshFeeds(), refreshStats(false)])
          setSelected(folderId === null ? '' : `folder:${folderId}`)
        }}
        intent={Intent.DANGER}
      >
        Are you sure you want to delete {feedsById.get(id)?.title || 'untitled'}?
      </Confirm>
      <Confirm
        open={renameFolderDialogOpen}
        setOpen={setRenameFolderDialogOpen}
        title="Rename Folder"
        callback={async () => {
          const title = folderTitleRef.current?.value
          if (title && title !== foldersById.get(id)?.title) {
            await xfetch(`./api/folders/${id}`, {
              method: 'PUT',
              body: JSON.stringify({ title }),
            })
            setFolders(folders =>
              folders?.map(folder => (folder.id === id ? { ...folder, title } : folder)),
            )
          }
        }}
      >
        <InputGroup defaultValue={foldersById.get(id)?.title} inputRef={folderTitleRef} />
      </Confirm>
      <Confirm
        open={deleteFolderDialogOpen}
        setOpen={setDeleteFolderDialogOpen}
        title="Delete Folder"
        callback={async () => {
          await xfetch(`./api/folders/${id}`, { method: 'DELETE' })
          await Promise.all([refreshFeeds(), refreshStats(false)])
          setSelected('')
        }}
        intent={Intent.DANGER}
      >
        Are you sure you want to delete {foldersById.get(id)?.title || 'untitled'}?
      </Confirm>
    </div>
  )
}

function CardItem({
  item,
  i,
  loaded,
  setStats,
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
  setStats: Dispatch<SetStateAction<Map<number, Stats> | undefined>>
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
        setSelectedItemDetails(await xfetch<Item>(`./api/items/${item.id}`))
        contentRef.current?.scrollTo(0, 0)
        if (item.status === 'unread') {
          await xfetch(`./api/items/${item.id}`, {
            method: 'PUT',
            body: JSON.stringify({ status: 'read' }),
          })
          setStats(
            stats =>
              stats &&
              new Map(
                [...stats].map(([feedId, stats]) => [
                  feedId,
                  feedId === item.feed_id
                    ? {
                        ...stats,
                        unread: stats.unread - 1,
                      }
                    : stats,
                ]),
              ),
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
