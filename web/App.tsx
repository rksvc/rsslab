import { Alert, Divider, FocusStyleManager, Pre } from '@blueprintjs/core'
import { useEffect, useMemo, useRef, useState } from 'react'
import FeedList from './FeedList'
import ItemList from './ItemList'
import ItemShow from './ItemShow'
import type {
  Feed,
  FeedState,
  Folder,
  FolderWithFeeds,
  Item,
  Items,
  Selected,
  Settings,
  State,
  Status,
} from './types'
import { xfetch } from './utils'

FocusStyleManager.onlyShowFocusOnTabs()

export default function App() {
  const [filter, setFilter] = useState('Unread')
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status>()
  const [selected, setSelected] = useState<Selected>()
  const [settings, setSettings] = useState<Settings>()

  const [items, setItems] = useState<Items>()
  const [itemsOutdated, setItemsOutdated] = useState(false)
  const [selectedItemId, setSelectedItemId] = useState<number>()

  const [selectedItemDetails, setSelectedItemDetails] = useState<Item>()
  const contentRef = useRef<HTMLDivElement>(null)

  const [alerts, setAlerts] = useState<string[]>([])
  const caughtErrors = useRef(new Set<any>())
  window.addEventListener('unhandledrejection', evt => {
    if (!caughtErrors.current.has(evt.reason)) {
      caughtErrors.current.add(evt.reason)
      setAlerts(alerts => [...alerts, String(evt.reason)])
    }
  })

  const refreshFeeds = async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('api/folders'),
      xfetch<Feed[]>('api/feeds'),
      xfetch<Settings>('api/settings'),
    ])
    setFolders(folders)
    setFeeds(feeds)
    setSettings(settings)
  }
  const refreshStats = async (loop = true) => {
    const { running, last_refreshed, state } = await xfetch<
      State & { state: (FeedState & { id: number })[] }
    >('api/status')
    setStatus({
      running,
      last_refreshed,
      state: new Map(state.map(({ id, ...state }) => [id, state])),
    })
    if (loop) {
      setItemsOutdated(true)
      if (running) setTimeout(() => refreshStats(), 500)
    }
  }
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshFeeds):
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshStats):
  useEffect(() => {
    ;(async () => {
      await Promise.all([refreshFeeds(), refreshStats()])
      setItemsOutdated(false)
    })()
  }, [])

  const errorCount = useMemo(
    () => status?.state.values().reduce((acc, state) => acc + (state.error ? 1 : 0), 0),
    [status],
  )
  const [foldersWithFeeds, feedsWithoutFolders, feedsById] = useMemo(() => {
    const foldersById = new Map<number, FolderWithFeeds>()
    for (const folder of folders ?? [])
      foldersById.set(folder.id, { ...folder, feeds: [] })
    const feedsById = new Map<number, Feed>()
    const feedsWithoutFolders: Feed[] = []
    for (const feed of feeds ?? []) {
      if (feed.folder_id === null) feedsWithoutFolders.push(feed)
      else foldersById.get(feed.folder_id)?.feeds.push(feed)
      feedsById.set(feed.id, feed)
    }
    return [[...foldersById.values()], feedsWithoutFolders, feedsById]
  }, [feeds, folders])

  const props = {
    filter,
    setFilter,
    folders,
    setFolders,
    setFeeds,
    status,
    setStatus,
    selected,
    setSelected,
    settings,
    setSettings,

    items,
    setItems,
    itemsOutdated,
    setItemsOutdated,
    selectedItemId,
    setSelectedItemId,

    setSelectedItemDetails,
    contentRef,

    refreshFeeds,
    refreshStats,
    errorCount,
    foldersWithFeeds,
    feedsWithoutFolders,
    feedsById,
  }

  return (
    <div style={{ display: 'flex', fontSize: '1rem', lineHeight: '1.5rem' }}>
      <div style={{ minWidth: '300px', maxWidth: '300px' }}>
        <FeedList {...props} />
      </div>
      <Divider />
      <div style={{ minWidth: '300px', maxWidth: '300px' }}>
        <ItemList {...props} />
      </div>
      <Divider />
      <div style={{ flexGrow: 1, minWidth: 0 }}>
        {selectedItemDetails && (
          <ItemShow {...props} selectedItemDetails={selectedItemDetails} />
        )}
      </div>
      {alerts.map((alert, i) => (
        <Alert
          // biome-ignore lint/suspicious/noArrayIndexKey:
          key={i}
          isOpen={!!alert}
          canEscapeKeyCancel
          onOpening={ref => ref.focus()}
          onClose={() => setAlerts(alerts => alerts.with(i, ''))}
        >
          {alert.includes('\n') ? <Pre>{alert}</Pre> : <p>{alert}</p>}
        </Alert>
      ))}
    </div>
  )
}
