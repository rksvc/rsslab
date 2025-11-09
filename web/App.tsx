import { Card, Divider, FocusStyleManager, Intent, OverlayToaster, Position } from '@blueprintjs/core'
import { type CSSProperties, useEffect, useMemo, useRef, useState } from 'react'
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
import { xfetch } from './utils.ts'

FocusStyleManager.onlyShowFocusOnTabs()
const darkTheme = (document.querySelector<HTMLMetaElement>('meta[name=dark-theme]')?.content.length ?? 0) > 0

const panelStyle = {
  display: 'flex',
  flexDirection: 'column',
  height: '100%',
} satisfies CSSProperties

export default function App() {
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status>()
  const [settings, setSettings] = useState<Settings>({ dark_theme: darkTheme })
  const [items, setItems] = useState<Items>()
  const [selectedItem, setSelectedItem] = useState<Item>()

  const [filter, setFilter] = useState<Filter>('Unread')
  const [selected, setSelected] = useState<Selected>()
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

  const refreshFeeds = async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('api/folders'),
      xfetch<Feed[]>('api/feeds'),
      xfetch<Settings>('api/settings'),
    ])
    setFolders(folders)
    setFeeds(feeds)
    setRefreshed({})
    setSettings(settings)
  }
  const refreshStats = async () => {
    const { running, last_refreshed, state } = await xfetch<
      Omit<Status, 'state'> & { state: Record<number, FeedState> }
    >('api/status')
    setStatus({
      running,
      last_refreshed,
      state: new Map(Object.entries(state).map(([id, state]) => [+id, state])),
    })
    setRefreshed({})
    setItemsOutdated(true)
    if (running) setTimeout(() => refreshStats(), 500)
  }
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
    setStatus,
    settings,
    setSettings,
    items,
    setItems,
    selectedItem,
    setSelectedItem,

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
    feedsById,
    foldersById,
    feedsWithoutFolders,
    foldersWithFeeds,
  }

  return (
    <Card
      style={{
        padding: 0,
        width: '100vw',
        display: 'flex',
        fontSize: '1rem',
        lineHeight: '1.5rem',
        boxShadow: 'none',
        borderRadius: 0,
      }}
    >
      <FeedList {...props} style={{ ...panelStyle, minWidth: '300px', maxWidth: '300px' }} />
      <Divider compact />
      <ItemList {...props} style={{ ...panelStyle, minWidth: '300px', maxWidth: '300px' }} />
      <Divider compact />
      {selectedItem?.content != null && (
        <ItemShow
          {...props}
          style={{ ...panelStyle, flexGrow: 1, minWidth: '300px' }}
          selectedItem={{ ...selectedItem, content: selectedItem.content }}
        />
      )}
      <OverlayToaster canEscapeKeyClear={false} position={Position.BOTTOM_RIGHT} ref={toaster} />
    </Card>
  )
}
