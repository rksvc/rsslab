import {
  Button,
  ButtonVariant,
  Card,
  CardList,
  Classes,
  Divider,
  InputGroup,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  Popover,
  Spinner,
} from '@blueprintjs/core'
import { type Dispatch, type SetStateAction, useCallback, useEffect, useRef, useState } from 'react'
import {
  Check,
  ChevronLeft,
  Circle,
  Edit,
  ExternalLink,
  Folder as FolderIcon,
  Link,
  MoreHorizontal,
  Move,
  RotateCw,
  Rss,
  Search,
  Star,
  Trash,
} from 'react-feather'
import TextEditor from './TextEditor.tsx'
import type { Feed, Filter, Folder, FolderWithFeeds, Item, Items, Selected, Status } from './types.ts'
import { fromNow, length, menuModifiers, param, parseFeedLink, xfetch } from './utils.ts'

export default function ItemList({
  setFolders,
  setFeeds,
  items,
  setItems,
  status,
  setStatus,
  selectedItemIndex,
  setSelectedItemIndex,
  setSelectedItemContent,

  filter,
  selected,
  setSelected,
  itemsOutdated,
  setItemsOutdated,
  setRefreshed,

  refreshStats,
  selectItem,
  foldersById,
  feedsById,
  foldersWithFeeds,
}: {
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>
  setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
  items?: Items
  setItems: Dispatch<SetStateAction<Items | undefined>>
  status?: Status
  setStatus: Dispatch<SetStateAction<Status | undefined>>
  selectedItemIndex?: number
  setSelectedItemIndex: Dispatch<SetStateAction<number | undefined>>
  setSelectedItemContent: Dispatch<SetStateAction<string | undefined>>

  filter: Filter
  selected: Selected
  setSelected: Dispatch<SetStateAction<Selected>>
  itemsOutdated: boolean
  setItemsOutdated: Dispatch<SetStateAction<boolean>>
  setRefreshed: Dispatch<SetStateAction<Record<never, never>>>

  refreshStats: (loop?: boolean) => Promise<void>
  selectItem: (index: number) => Promise<void>
  foldersById?: Map<number, FolderWithFeeds>
  feedsById?: Map<number, Feed>
  foldersWithFeeds?: FolderWithFeeds[]
}) {
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
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
  const updateFeedAttr = async <T extends 'title' | 'feed_link' | 'folder_id'>(
    id: number,
    attrName: T,
    value: Feed[T],
  ) => {
    await xfetch(`api/feeds/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ [attrName]: value ?? -1 }),
    })
    setFeeds(feeds => feeds?.map(feed => (feed.id === id ? { ...feed, [attrName]: value } : feed)))
  }

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
    setSelectedItemIndex(undefined)
    setSelectedItemContent(undefined)
    setItemsOutdated(false)
    itemListRef.current?.scrollTo(0, 0)
  }, [query, setItems, setSelectedItemIndex, setSelectedItemContent, setItemsOutdated])
  useEffect(() => {
    refresh()
  }, [refresh])

  const feedError = selected?.feed_id != null && status?.state.get(selected.feed_id)?.error
  return (
    <div id="item-list">
      <div className="topbar" style={{ gap: length(1.5) }}>
        <Button
          id="show-feeds"
          style={{ display: 'none' }}
          icon={<ChevronLeft />}
          title={'Show Feeds'}
          variant={ButtonVariant.MINIMAL}
          onClick={() => setSelected(undefined)}
        />
        <InputGroup
          inputRef={searchRef}
          leftIcon={
            <Search style={{ pointerEvents: 'none', alignSelf: 'anchor-center' }} className={Classes.ICON} />
          }
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
          }}
        />
        <Popover
          transitionDuration={0}
          modifiers={menuModifiers}
          onOpening={() => setMenuOpen(true)}
          onClosed={() => setMenuOpen(false)}
          content={
            selected?.feed_id != null
              ? (() => {
                  const feed = feedsById?.get(selected.feed_id)
                  if (!feed) return undefined
                  return (
                    <Menu>
                      {feed.link && (
                        <MenuItem
                          text="Website"
                          intent={Intent.PRIMARY}
                          labelElement={<ExternalLink />}
                          icon={<Link />}
                          target="_blank"
                          href={feed.link}
                          rel="noopener noreferrer"
                          referrerPolicy="no-referrer"
                        />
                      )}
                      <MenuItem
                        text="Feed Link"
                        intent={Intent.PRIMARY}
                        labelElement={<ExternalLink />}
                        icon={<Rss />}
                        target="_blank"
                        href={(() => {
                          const [scheme, link] = parseFeedLink(feed.feed_link)
                          return scheme ? `api/transform/${scheme}/${encodeURIComponent(link)}` : link
                        })()}
                        rel="noopener noreferrer"
                        referrerPolicy="no-referrer"
                      />
                      <MenuDivider />
                      <TextEditor
                        menuText="Rename"
                        menuIcon={<Edit />}
                        defaultValue={feed.title}
                        onConfirm={async title => {
                          if (!title) throw new Error('Feed name is required')
                          await updateFeedAttr(feed.id, 'title', title)
                        }}
                      />
                      <TextEditor
                        menuText="Change Link"
                        menuIcon={<Edit />}
                        defaultValue={feed.feed_link}
                        textAreaStyle={{ wordBreak: 'break-all' }}
                        onConfirm={async feedLink => {
                          if (!feedLink) throw new Error('Feed link is required')
                          await updateFeedAttr(feed.id, 'feed_link', feedLink)
                        }}
                      />
                      <MenuItem
                        text="Refresh"
                        icon={<RotateCw />}
                        disabled={!!status?.running}
                        onClick={async () => {
                          await xfetch(`api/feeds/${feed.id}/refresh`, { method: 'POST' })
                          await refreshStats()
                        }}
                      />
                      <MenuItem text="Move to..." icon={<Move />} disabled={!foldersWithFeeds?.length}>
                        {[
                          { key: null, text: '--' },
                          ...(foldersWithFeeds ?? []).map(({ id, title }) => ({ key: id, text: title })),
                        ]
                          .filter(({ key }) => key !== feed.folder_id)
                          .map(({ key, text }) => (
                            <MenuItem
                              key={key}
                              text={text}
                              icon={<FolderIcon />}
                              onClick={async () => {
                                await updateFeedAttr(feed.id, 'folder_id', key)
                                setRefreshed({})
                              }}
                            />
                          ))}
                      </MenuItem>
                      <Deleter
                        isOpen={menuOpen}
                        onConfirm={async () => {
                          await xfetch(`api/feeds/${feed.id}`, { method: 'DELETE' })
                          setFeeds(feeds => feeds?.filter(f => f.id !== feed.id))
                          setStatus(
                            status =>
                              status && {
                                ...status,
                                state: new Map(status.state.entries().filter(([id]) => id !== feed.id)),
                              },
                          )
                          setSelected(feed.folder_id === null ? undefined : { folder_id: feed.folder_id })
                        }}
                      />
                    </Menu>
                  )
                })()
              : selected?.folder_id != null
                ? (() => {
                    const folder = foldersById?.get(selected.folder_id)
                    if (!folder) return undefined
                    return (
                      <Menu>
                        <TextEditor
                          menuText="Rename"
                          menuIcon={<Edit />}
                          defaultValue={folder.title}
                          onConfirm={async title => {
                            if (!title) throw new Error('Folder title is required')
                            await xfetch(`api/folders/${folder.id}`, {
                              method: 'PUT',
                              body: JSON.stringify({ title }),
                            })
                            setFolders(folders =>
                              folders?.map(f => (f.id === folder.id ? { ...f, title } : f)),
                            )
                          }}
                        />
                        <MenuItem
                          text="Refresh"
                          icon={<RotateCw />}
                          disabled={!!status?.running}
                          onClick={async () => {
                            await xfetch(`api/folders/${folder.id}/refresh`, {
                              method: 'POST',
                            })
                            await refreshStats()
                          }}
                        />
                        <Deleter
                          isOpen={menuOpen}
                          onConfirm={async () => {
                            await xfetch(`api/folders/${folder.id}`, { method: 'DELETE' })
                            const deletedFeeds = new Set(folder.feeds.map(feed => feed.id))
                            setFolders(folders => folders?.filter(f => f.id !== folder.id))
                            setFeeds(feeds => feeds?.filter(feed => !deletedFeeds.has(feed.id)))
                            setStatus(
                              status =>
                                status && {
                                  ...status,
                                  state: new Map(
                                    status.state.entries().filter(([id]) => !deletedFeeds.has(id)),
                                  ),
                                },
                            )
                            setSelected(null)
                          }}
                        />
                      </Menu>
                    )
                  })()
                : undefined
          }
        >
          <Button icon={<MoreHorizontal />} variant={ButtonVariant.MINIMAL} disabled={!selected} />
        </Popover>
      </div>
      <Divider compact />
      <CardList style={{ flexGrow: 1 }} ref={itemListRef} bordered={false} compact>
        {items?.list.map((item, i) => (
          <CardItem
            key={item.id}
            item={item}
            index={i}
            isSelected={selectedItemIndex != null && item.id === items.list[selectedItemIndex].id}
            feedsById={feedsById}
            selectItem={selectItem}
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
          <Divider compact />
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
  index,
  isSelected,
  feedsById,
  selectItem,
}: {
  item: Item
  index: number
  isSelected: boolean
  feedsById?: Map<number, Feed>
  selectItem: (index: number) => Promise<void>
}) {
  const prevStatus = usePrevious(item.status)
  return (
    <Card
      selected={isSelected}
      interactive
      onClick={async () => {
        if (!isSelected) await selectItem(index)
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

function Deleter({ isOpen, onConfirm }: { isOpen: boolean; onConfirm: () => Promise<void> }) {
  const [state, setState] = useState<boolean>()
  const closerRef = useRef<HTMLDivElement>(null)
  if (!isOpen && state === false) setState(undefined)

  return (
    <>
      <MenuItem
        text={`Delete${state === false ? ' (confirm)' : ''}`}
        active={state != null}
        disabled={state}
        icon={state ? <Spinner intent={Intent.DANGER} /> : <Trash />}
        intent={Intent.DANGER}
        shouldDismissPopover={false}
        onClick={async () => {
          if (state === false) {
            setState(true)
            try {
              await onConfirm()
              closerRef.current?.click()
            } finally {
              setState(undefined)
            }
          } else if (state === undefined) {
            setState(false)
          }
        }}
      />
      <div className={Classes.POPOVER_DISMISS} ref={closerRef} hidden />
    </>
  )
}

function usePrevious<T>(value: T) {
  const ref = useRef<T>()
  useEffect(() => {
    ref.current = value
  })
  return ref.current
}
