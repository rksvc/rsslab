import { Divider, FocusStyleManager } from '@blueprintjs/core'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import FeedList from './FeedList'
import ItemList from './ItemList'
import ItemShow from './ItemShow'
import type {
  Feed,
  Folder,
  FolderWithFeeds,
  Image,
  Item,
  Settings,
  State,
  Stats,
  Status,
} from './types'
import { xfetch } from './utils'

FocusStyleManager.onlyShowFocusOnTabs()

export default function App() {
  const [filter, setFilter] = useState('Unread')
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status>()
  const [errors, setErrors] = useState<Map<number, string>>()
  const [selected, setSelected] = useState('')
  const [settings, setSettings] = useState<Settings>()

  const [items, setItems] = useState<(Item & Image)[]>()
  const [selectedItemId, setSelectedItemId] = useState<number>()

  const [selectedItemDetails, setSelectedItemDetails] = useState<Item>()
  const contentRef = useRef<HTMLDivElement>(null)

  const refreshFeeds = useCallback(async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('api/folders'),
      xfetch<Feed[]>('api/feeds'),
      xfetch<Settings>('api/settings'),
    ])
    setFolders(folders)
    setFeeds(feeds)
    setSettings(settings)
  }, [])
  const refreshStats = useCallback(async (loop = true) => {
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
    if (loop && running) setTimeout(() => refreshStats(), 500)
  }, [])
  useEffect(() => {
    refreshFeeds()
    refreshStats()
  }, [refreshFeeds, refreshStats])

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
    selectedItemId,
    setSelectedItemId,

    setSelectedItemDetails,
    contentRef,

    refreshFeeds,
    refreshStats,
    foldersWithFeeds,
    feedsWithoutFolders,
    feedsById,
  }

  return (
    <div className="flex flex-row text-base min-h-screen max-h-screen">
      <div className="min-w-[300px] max-w-[300px]">
        <FeedList {...props} />
      </div>
      <Divider className="m-0" />
      <div className="min-w-[300px] max-w-[300px]">
        <ItemList {...props} />
      </div>
      <Divider className="m-0" />
      <div className="grow min-w-0">
        {selectedItemDetails && (
          <ItemShow {...props} selectedItemDetails={selectedItemDetails} />
        )}
      </div>
    </div>
  )
}
