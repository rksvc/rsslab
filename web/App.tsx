import { Alert, Divider, FocusStyleManager } from '@blueprintjs/core'
import { useEffect, useMemo, useRef, useState } from 'react'
import FeedList from './FeedList'
import ItemList from './ItemList'
import ItemShow from './ItemShow'
import type {
  Feed,
  Folder,
  FolderWithFeeds,
  Item,
  Settings,
  State,
  Stats,
  Status,
} from './types'
import type { Xfetch } from './utils'

FocusStyleManager.onlyShowFocusOnTabs()

export default function App() {
  const [filter, setFilter] = useState('Unread')
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status>()
  const [errors, setErrors] = useState<Map<number, string>>()
  const [selected, setSelected] = useState('')
  const [settings, setSettings] = useState<Settings>()

  const [items, setItems] = useState<Item[]>()
  const [itemsOutdated, setItemsOutdated] = useState(false)
  const [selectedItemId, setSelectedItemId] = useState<number>()

  const [selectedItemDetails, setSelectedItemDetails] = useState<Item>()
  const contentRef = useRef<HTMLDivElement>(null)

  const [alerts, setAlerts] = useState<string[]>([])

  const xfetch: Xfetch = async <T,>(
    url: string,
    options?: RequestInit,
  ): Promise<T | unknown> => {
    if (typeof options?.body === 'string')
      options.headers = { 'Content-Type': 'application/json' }
    try {
      const response = await fetch(url, options)
      const text = await response.text()
      if (response.ok) return text && text !== 'OK' && JSON.parse(text)
      throw new Error(text || `${response.status} ${response.statusText}`)
    } catch (error) {
      setAlerts(alerts => [...alerts, String(error)])
      throw error
    }
  }

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
    const [errors, { running, last_refreshed, stats }] = await Promise.all([
      xfetch<Record<string, string>>('api/feeds/errors'),
      xfetch<State & { stats: Record<string, Stats> }>('api/status'),
    ])
    setErrors(
      new Map(Object.entries(errors).map(([id, error]) => [Number.parseInt(id), error])),
    )
    setStatus({
      running,
      last_refreshed,
      stats: new Map(
        Object.entries(stats).map(([id, stats]) => [Number.parseInt(id), stats]),
      ),
    })
    setItemsOutdated(true)
    if (loop && running) setTimeout(() => refreshStats(), 500)
  }
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshFeeds):
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshStats):
  useEffect(() => {
    ;(async () => {
      await Promise.all([refreshFeeds(), refreshStats()])
      setItemsOutdated(false)
    })()
  }, [])

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
    errors,
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
    foldersWithFeeds,
    feedsWithoutFolders,
    feedsById,

    xfetch,
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
          onClose={() => setAlerts(alerts => alerts.with(i, ''))}
        >
          {alert.includes('\n') ? (
            <pre style={{ marginTop: 0, fontFamily: 'inherit' }}>{alert}</pre>
          ) : (
            <p>{alert}</p>
          )}
        </Alert>
      ))}
    </div>
  )
}
