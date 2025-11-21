import { Card, Divider, FocusStyleManager, Intent, OverlayToaster, Position } from '@blueprintjs/core'
import { enableMapSet } from 'immer'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useImmer } from 'use-immer'
import FeedList from './FeedList.tsx'
import ItemList from './ItemList.tsx'
import ItemShow from './ItemShow.tsx'
import type {
  Feed,
  FeedState,
  Filter,
  Folder,
  FolderWithFeeds,
  Item,
  Items,
  Selected,
  Settings,
  Status,
} from './types.ts'
import { cn, xfetch } from './utils.ts'

enableMapSet()
FocusStyleManager.onlyShowFocusOnTabs()
const darkTheme = (document.querySelector<HTMLMetaElement>('meta[name=dark-theme]')?.content.length ?? 0) > 0

export default function App() {
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, updateStatus] = useImmer<Status | undefined>(undefined)
  const [settings, setSettings] = useState<Settings>({ dark_theme: darkTheme })
  const [items, updateItems] = useImmer<Items | undefined>(undefined)
  const [selectedItemIndex, setSelectedItemIndex] = useState<number>()
  const [selectedItemContent, setSelectedItemContent] = useState<string>()

  const [filter, setFilter] = useState<Filter>('Unread')
  const [selected, setSelected] = useState<Selected>(null)
  const [refreshed, setRefreshed] = useState<Record<never, never>>({})
  const [itemsOutdated, setItemsOutdated] = useState(false)
  const contentRef = useRef<HTMLDivElement>(null)

  const caughtErrors = useRef(new Set<any>())
  const toaster = useRef<OverlayToaster>(null)
  useEffect(() => {
    window.addEventListener('unhandledrejection', evt => {
      if (!caughtErrors.current.has(evt.reason)) {
        caughtErrors.current.add(evt.reason)
        const message = evt.reason instanceof Error ? evt.reason.message : String(evt.reason)
        toaster.current?.show({
          intent: Intent.DANGER,
          message: message.split('\n', 2)[0],
          timeout: 0,
        })
      }
    })
  }, [])

  const refreshFeeds = useCallback(async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('api/folders'),
      xfetch<Feed[]>('api/feeds'),
      xfetch<Settings>('api/settings'),
    ])
    setFolders(folders)
    setFeeds(feeds)
    setRefreshed({})
    setSettings(settings)
  }, [])
  const refreshStats = useCallback(async () => {
    const { running, last_refreshed, state } = await xfetch<
      Omit<Status, 'state'> & { state: Record<number, FeedState> }
    >('api/status')
    updateStatus({
      running,
      last_refreshed,
      state: new Map(Object.entries(state).map(([id, state]) => [+id, state])),
    })
    setRefreshed({})
    setItemsOutdated(true)
    if (running) setTimeout(() => refreshStats(), 500)
  }, [updateStatus])
  const selectItem = useCallback(
    async (index: number) => {
      const item = items?.list[index]
      if (!item) return
      setSelectedItemIndex(index)
      setSelectedItemContent((await xfetch<Item>(`api/items/${item.id}`)).content)
      contentRef.current?.scrollTo(0, 0)
      if (item.status === 'unread') {
        await xfetch(`api/items/${item.id}`, {
          method: 'PUT',
          body: JSON.stringify({ status: 'read' }),
        })
        updateStatus(status => {
          const state = status?.state.get(item.feed_id)
          if (state) --state.unread
        })
        updateItems(items => {
          for (const i of items?.list ?? [])
            if (i.id === item.id) {
              i.status = 'read'
              break
            }
        })
      }
    },
    [items?.list, updateStatus, updateItems],
  )
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshFeeds): run only at startup
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshStats): run only at startup
  useEffect(() => {
    ;(async () => {
      await Promise.all([refreshFeeds(), refreshStats()])
      setItemsOutdated(false)
    })()
  }, [])
  useEffect(() => {
    if (settings.dark_theme) document.body.classList.add('bp6-dark')
    else document.body.classList.remove('bp6-dark')
  }, [settings])

  const [feedsById, foldersById, feedsWithoutFolders, foldersWithFeeds] = useMemo(() => {
    if (!feeds || !folders) return []
    const foldersById = new Map<number, FolderWithFeeds>()
    for (const folder of folders) foldersById.set(folder.id, { ...folder, feeds: [] })
    const feedsById = new Map<number, Feed>()
    const feedsWithoutFolders: Feed[] = []
    for (const feed of feeds) {
      if (feed.folder_id === null) feedsWithoutFolders.push(feed)
      else foldersById.get(feed.folder_id)?.feeds.push(feed)
      feedsById.set(feed.id, feed)
    }
    return [feedsById, foldersById, feedsWithoutFolders, [...foldersById.values()]]
  }, [feeds, folders])

  const props = {
    setFolders,
    setFeeds,
    status,
    updateStatus,
    settings,
    setSettings,
    items,
    updateItems,
    selectedItemIndex,
    setSelectedItemIndex,
    setSelectedItemContent,

    filter,
    setFilter,
    selected,
    setSelected,
    refreshed,
    setRefreshed,
    itemsOutdated,
    setItemsOutdated,
    contentRef,

    refreshFeeds,
    refreshStats,
    selectItem,
    feedsById,
    foldersById,
    feedsWithoutFolders,
    foldersWithFeeds,
  }

  return (
    <Card
      className={cn(selected !== undefined && 'feed-selected', selectedItemIndex != null && 'item-selected')}
      style={{
        padding: 0,
        height: '100vh',
        display: 'flex',
        fontSize: '1rem',
        lineHeight: '1.5rem',
        boxShadow: 'none',
        borderRadius: 0,
      }}
    >
      <FeedList {...props} />
      <Divider id="list-divider" compact />
      <ItemList {...props} />
      <Divider id="item-divider" compact />
      {items != null && selectedItemIndex != null && selectedItemContent != null && (
        <ItemShow
          {...props}
          items={items}
          selectedItemIndex={selectedItemIndex}
          item={{ ...items.list[selectedItemIndex], content: selectedItemContent }}
        />
      )}
      <OverlayToaster canEscapeKeyClear={false} position={Position.BOTTOM_RIGHT} ref={toaster} />
    </Card>
  )
}
