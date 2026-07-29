import { Classes } from '@blueprintjs/core'
import {
  createContext,
  type Dispatch,
  type ReactNode,
  type RefObject,
  type SetStateAction,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react'

import {
  Theme,
  type Feed,
  type FeedState,
  type Filter,
  type Folder,
  type FolderWithFeeds,
  type Item,
  type Items,
  type ItemWithContent,
  type Selected,
  type Settings,
  type Status,
} from './types.ts'
import { xfetch } from './utils.ts'

const Context = createContext<
  | {
      setFolders: Dispatch<SetStateAction<Folder[] | undefined>>
      feeds?: Feed[]
      setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
      status?: Status
      setStatus: Dispatch<SetStateAction<Status | undefined>>
      settings: Settings
      setSettings: Dispatch<SetStateAction<Settings>>
      items?: Items
      setItems: Dispatch<SetStateAction<Items | undefined>>
      selectedItemId?: number
      setSelectedItemId: Dispatch<SetStateAction<number | undefined>>
      selectedItem?: ItemWithContent
      setSelectedItem: Dispatch<SetStateAction<ItemWithContent | undefined>>

      filter: Filter
      setFilter: Dispatch<SetStateAction<Filter>>
      selected: Selected
      setSelected: Dispatch<SetStateAction<Selected>>
      feedListRefreshed: Record<never, never>
      setFeedListRefreshed: Dispatch<SetStateAction<Record<never, never>>>
      itemsOutdated: boolean
      setItemsOutdated: Dispatch<SetStateAction<boolean>>
      mobile: boolean
      contentRef: RefObject<HTMLDivElement | null>

      refreshFeeds: () => Promise<void>
      refreshStats: (refreshFeedList?: boolean) => Promise<void>
      selectItem: (item: Item) => Promise<void>
      feedsById?: Map<number, Feed>
      foldersById?: Map<number, FolderWithFeeds>
      feedsOutsideFolders?: Feed[]
      foldersWithFeeds?: FolderWithFeeds[]
    }
  | undefined
>(undefined)

export function useMyContext() {
  const value = useContext(Context)
  if (value == null) throw new Error('useMyContext must be used within ContextProvider')
  return value
}

const mobileQuery = window.matchMedia('(max-width: 991.98px)')
const darkQuery = window.matchMedia('(prefers-color-scheme: dark)')
const theme = +(document.querySelector<HTMLMetaElement>('meta[name=theme]')?.content ?? '')

export default function ContextProvider({ children }: { children: ReactNode }) {
  const [folders, setFolders] = useState<Folder[]>()
  const [feeds, setFeeds] = useState<Feed[]>()
  const [status, setStatus] = useState<Status | undefined>(undefined)
  const [settings, setSettings] = useState<Settings>({ theme })
  const [items, setItems] = useState<Items | undefined>(undefined)
  const [selectedItemId, setSelectedItemId] = useState<number>()
  const [selectedItem, setSelectedItem] = useState<ItemWithContent>()

  const [filter, setFilter] = useState<Filter>('Unread')
  const [selected, setSelected] = useState<Selected>(null)
  const [feedListRefreshed, setFeedListRefreshed] = useState<Record<never, never>>({})
  const [itemsOutdated, setItemsOutdated] = useState(false)
  const [mobile, setMobile] = useState(mobileQuery.matches)
  const [preferDark, setPreferDark] = useState(darkQuery.matches)
  const contentRef = useRef<HTMLDivElement>(null)

  const refreshFeeds = async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('api/folders'),
      xfetch<Feed[]>('api/feeds'),
      xfetch<Settings>('api/settings'),
    ])
    setFolders(folders)
    setFeeds(feeds.map(f => ({ ...f, has_icon: f.has_icon || null })))
    setFeedListRefreshed({})
    setSettings(settings)
  }

  const refreshStats = async (refreshFeedList = true) => {
    const { running, last_refreshed, state } = await xfetch<
      Omit<Status, 'state'> & { state: Record<number, FeedState> }
    >('api/status')
    setStatus({
      running,
      last_refreshed,
      state: new Map(Object.entries(state).map(([id, state]) => [+id, state])),
    })
    if (refreshFeedList) setFeedListRefreshed({})
    setItemsOutdated(true)
    if (running) setTimeout(() => refreshStats(), 500)
  }

  const selectItem = async (item: Item) => {
    setSelectedItemId(item.id)
    setSelectedItem(await xfetch<ItemWithContent>(`api/items/${item.id}`))
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
  }

  useEffect(() => {
    void (async () => {
      await Promise.all([refreshFeeds(), refreshStats()])
      setItemsOutdated(false)
    })()
    mobileQuery.addEventListener('change', evt => setMobile(evt.matches))
    darkQuery.addEventListener('change', evt => setPreferDark(evt.matches))
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, []) // run only at startup
  useEffect(() => {
    if (settings.theme === Theme.Dark || (settings.theme === Theme.Auto && preferDark))
      document.body.classList.add(Classes.DARK)
    else document.body.classList.remove(Classes.DARK)
  }, [settings, preferDark])

  const [feedsById, foldersById, feedsOutsideFolders, foldersWithFeeds] = (() => {
    if (!feeds || !folders) return []
    const foldersById = new Map<number, FolderWithFeeds>()
    for (const folder of folders) foldersById.set(folder.id, { ...folder, feeds: [] })
    const feedsById = new Map<number, Feed>()
    const feedsOutsideFolders: Feed[] = []
    for (const feed of feeds) {
      if (feed.folder_id === null) feedsOutsideFolders.push(feed)
      else foldersById.get(feed.folder_id)?.feeds.push(feed)
      feedsById.set(feed.id, feed)
    }
    return [feedsById, foldersById, feedsOutsideFolders, [...foldersById.values()]]
  })()

  return (
    <Context
      value={{
        setFolders,
        feeds,
        setFeeds,
        status,
        setStatus,
        settings,
        setSettings,
        items,
        setItems,
        selectedItemId,
        setSelectedItemId,
        selectedItem,
        setSelectedItem,

        filter,
        setFilter,
        selected,
        setSelected,
        feedListRefreshed,
        setFeedListRefreshed,
        itemsOutdated,
        setItemsOutdated,
        mobile,
        contentRef,

        refreshFeeds,
        refreshStats,
        selectItem,
        feedsById,
        foldersById,
        feedsOutsideFolders,
        foldersWithFeeds,
      }}
    >
      {children}
    </Context>
  )
}
