import {
  Card,
  Collapse,
  Divider,
  FocusStyleManager,
  Intent,
  OverlayToaster,
  Position,
} from '@blueprintjs/core'
import { useEffect, useMemo, useRef, useState } from 'react'
import { AlertCircle } from 'react-feather'
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
  State,
  Status,
} from './types.ts'
import { iconProps, xfetch } from './utils.ts'

FocusStyleManager.onlyShowFocusOnTabs()
Collapse.defaultProps.transitionDuration = 0
const darkTheme = (document.querySelector<HTMLMetaElement>('meta[name=dark-theme]')?.content.length ?? 0) > 0

export default function App() {
  const [filter, setFilter] = useState<Filter>('Unread')
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status>()
  const [selected, setSelected] = useState<Selected>()
  const [settings, setSettings] = useState<Settings>({ dark_theme: darkTheme })
  const [refreshed, setRefreshed] = useState<Record<never, never>>({})

  const [items, setItems] = useState<Items>()
  const [itemsOutdated, setItemsOutdated] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Item>()

  const contentRef = useRef<HTMLDivElement>(null)

  const caughtErrors = useRef(new Set<any>())
  const toaster = useRef<OverlayToaster>(null)
  useEffect(() => {
    window.addEventListener('unhandledrejection', evt => {
      if (!caughtErrors.current.has(evt.reason)) {
        caughtErrors.current.add(evt.reason)
        const message = evt.reason instanceof Error ? evt.reason.message : String(evt.reason)
        toaster.current?.show({
          icon: <AlertCircle {...iconProps} />,
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
      State & { state: (FeedState & { id: number })[] }
    >('api/status')
    setStatus({
      running,
      last_refreshed,
      state: new Map(state.map(({ id, ...state }) => [id, state])),
    })
    setRefreshed({})
    setItemsOutdated(true)
    if (running) setTimeout(() => refreshStats(), 500)
  }
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshFeeds):
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshStats):
  useEffect(() => {
    ;(async () => {
      await Promise.all([refreshFeeds(), refreshStats()])
      setItemsOutdated(false)
    })()
  }, [])
  useEffect(() => {
    if (settings.dark_theme) document.body.classList.add('bp5-dark')
    else document.body.classList.remove('bp5-dark')
  }, [settings])

  const [foldersById, foldersWithFeeds, feedsWithoutFolders, feedsById] = useMemo(() => {
    const foldersById = new Map<number, FolderWithFeeds>()
    for (const folder of folders ?? []) foldersById.set(folder.id, { ...folder, feeds: [] })
    const feedsById = new Map<number, Feed>()
    const feedsWithoutFolders: Feed[] = []
    for (const feed of feeds ?? []) {
      if (feed.folder_id === null) feedsWithoutFolders.push(feed)
      else foldersById.get(feed.folder_id)?.feeds.push(feed)
      feedsById.set(feed.id, feed)
    }
    return [foldersById, [...foldersById.values()], feedsWithoutFolders, feedsById]
  }, [feeds, folders])
  const errorCount = useMemo(
    () => status?.state.values().reduce((acc, state) => acc + (state.error ? 1 : 0), 0),
    [status],
  )

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
    refreshed,
    setRefreshed,

    items,
    setItems,
    itemsOutdated,
    setItemsOutdated,
    selectedItem,
    setSelectedItem,

    contentRef,

    refreshFeeds,
    refreshStats,
    errorCount,
    foldersById,
    foldersWithFeeds,
    feedsWithoutFolders,
    feedsById,
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
      <FeedList {...props} style={{ minWidth: '300px', maxWidth: '300px' }} />
      <Divider />
      <ItemList {...props} style={{ minWidth: '300px', maxWidth: '300px' }} />
      <Divider />
      {selectedItem?.content != null && (
        <ItemShow
          {...props}
          style={{ flexGrow: 1, minWidth: '300px' }}
          selectedItem={{ ...selectedItem, content: selectedItem.content }}
        />
      )}
      <OverlayToaster canEscapeKeyClear={false} position={Position.BOTTOM_RIGHT} ref={toaster} />
    </Card>
  )
}
