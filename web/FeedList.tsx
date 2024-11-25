import {
  Button,
  ButtonGroup,
  Classes,
  Colors,
  ContextMenu,
  Divider,
  FileInput,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  NumericInput,
  Popover,
  Spinner,
  TextArea,
  Tooltip,
  Tree,
  type TreeNodeInfo,
} from '@blueprintjs/core'
import {
  type CSSProperties,
  type Dispatch,
  type ReactNode,
  type RefObject,
  type SetStateAction,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  AlertCircle,
  Circle,
  Download,
  Edit,
  ExternalLink,
  Folder as FolderIcon,
  FolderMinus,
  Link,
  Menu as MenuIcon,
  Moon,
  MoreHorizontal,
  Move,
  Plus,
  RotateCw,
  Rss,
  Star,
  Sun,
  Trash,
  Upload,
} from 'react-feather'
import { NewFeedDialog } from './NewFeed.tsx'
import type { Feed, FeedState, Filter, Folder, FolderWithFeeds, Selected, Settings, Status } from './types.ts'
import {
  cn,
  compareTitle,
  fromNow,
  iconProps,
  length,
  menuIconProps,
  panelStyle,
  parseFeedLink,
  statusBarStyle,
  xfetch,
} from './utils.ts'

export default function FeedList({
  style,

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

  refreshFeeds,
  refreshStats,
  errorCount,
  foldersById,
  foldersWithFeeds,
  feedsWithoutFolders,
  feedsById,
}: {
  style?: CSSProperties

  filter: Filter
  setFilter: Dispatch<SetStateAction<Filter>>
  folders?: Folder[]
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>
  setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
  status?: Status
  setStatus: Dispatch<React.SetStateAction<Status | undefined>>
  selected: Selected
  setSelected: Dispatch<SetStateAction<Selected>>
  settings: Settings
  setSettings: Dispatch<SetStateAction<Settings>>
  refreshed: Record<never, never>
  setRefreshed: Dispatch<SetStateAction<Record<never, never>>>

  refreshFeeds: () => Promise<void>
  refreshStats: (loop?: boolean) => Promise<void>
  errorCount?: number
  foldersById: Map<number, FolderWithFeeds>
  foldersWithFeeds?: FolderWithFeeds[]
  feedsWithoutFolders?: Feed[]
  feedsById: Map<number, Feed>
}) {
  const [newFeedDialogOpen, setNewFeedDialogOpen] = useState(false)
  const refreshRateRef = useRef<HTMLInputElement>(null)
  const opmlFormRef = useRef<HTMLFormElement>(null)
  const menuCloserRef = useRef<HTMLDivElement>(null)

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
    if (attrName === 'folder_id') setRefreshed({})
  }
  const secondaryLabel = (state?: FeedState) =>
    filter === 'Unread' ? (
      state?.unread.toString()
    ) : filter === 'Starred' ? (
      state?.starred.toString()
    ) : state?.error ? (
      <span
        style={{ display: 'flex' }}
        title={state.last_refreshed && `Last Refreshed: ${new Date(state.last_refreshed).toLocaleString()}`}
      >
        <AlertCircle {...iconProps} />
      </span>
    ) : undefined
  const feed = (feed: Feed) =>
    ({
      id: `feed:${feed.id}`,
      isSelected: selected?.feed_id === feed.id,
      secondaryLabel: secondaryLabel(status?.state.get(feed.id)),
      nodeData: { feed_id: feed.id },
      icon: feed.has_icon ? (
        <img style={{ width: length(4), marginRight: '7px' }} src={`api/feeds/${feed.id}/icon`} />
      ) : (
        <span style={{ display: 'flex' }}>
          <Rss style={{ marginRight: '6px' }} {...iconProps} />
        </span>
      ),
      label: (
        <ContextMenu
          content={({ isOpen }) => (
            <Menu>
              {feed.link && (
                <MenuItem
                  text="Website"
                  intent={Intent.PRIMARY}
                  labelElement={<ExternalLink {...menuIconProps} />}
                  icon={<Link {...menuIconProps} />}
                  target="_blank"
                  href={feed.link}
                />
              )}
              <MenuItem
                text="Feed Link"
                intent={Intent.PRIMARY}
                labelElement={<ExternalLink {...menuIconProps} />}
                icon={<Rss {...menuIconProps} />}
                target="_blank"
                href={(() => {
                  const [scheme, link] = parseFeedLink(feed.feed_link)
                  return scheme ? `api/transform/${scheme}/${encodeURIComponent(link)}` : link
                })()}
              />
              <MenuDivider />
              <TextEditor
                defaultValue={feed.title}
                onConfirm={async title => {
                  if (!title) throw new Error('Feed name is required')
                  await updateFeedAttr(feed.id, 'title', title)
                }}
              >
                <MenuItem text="Rename" icon={<Edit {...menuIconProps} />} shouldDismissPopover={false} />
              </TextEditor>
              <TextEditor
                defaultValue={feed.feed_link}
                textAreaStyle={{ wordBreak: 'break-all' }}
                onConfirm={async feedLink => {
                  if (!feedLink) throw new Error('Feed link is required')
                  await updateFeedAttr(feed.id, 'feed_link', feedLink)
                }}
              >
                <MenuItem
                  text="Change Link"
                  icon={<Edit {...menuIconProps} />}
                  shouldDismissPopover={false}
                />
              </TextEditor>
              <MenuItem
                text="Refresh"
                icon={<RotateCw {...menuIconProps} />}
                disabled={!!status?.running}
                onClick={async () => {
                  await xfetch(`api/feeds/${feed.id}/refresh`, { method: 'POST' })
                  await refreshStats()
                }}
              />
              <MenuItem text="Move to..." icon={<Move {...menuIconProps} />} disabled={!folders?.length}>
                {feed.folder_id != null && (
                  <MenuItem
                    text="--"
                    icon={<FolderMinus {...menuIconProps} />}
                    onClick={() => updateFeedAttr(feed.id, 'folder_id', null)}
                  />
                )}
                {folders
                  ?.filter(folder => folder.id !== feed.folder_id)
                  .map(folder => (
                    <MenuItem
                      key={folder.id}
                      text={folder.title}
                      icon={<FolderIcon {...menuIconProps} />}
                      onClick={() => updateFeedAttr(feed.id, 'folder_id', folder.id)}
                    />
                  ))}
              </MenuItem>
              <Deleter
                isOpen={isOpen}
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
          )}
          onContextMenu={() =>
            setSelected(selected => (selected?.feed_id === feed.id ? selected : { feed_id: feed.id }))
          }
        >
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }} title={feed.title}>
            {feed.title || 'untitled'}
          </span>
        </ContextMenu>
      ),
    }) satisfies TreeNodeInfo<Selected>
  const setExpanded = (isExpanded: boolean) => async (node: TreeNodeInfo<Selected>) => {
    const id = node.nodeData?.folder_id
    if (id == null) return
    setFolders(folders =>
      folders?.map(folder => (folder.id === id ? { ...folder, is_expanded: isExpanded } : folder)),
    )
    await xfetch(`api/folders/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ is_expanded: isExpanded }),
    })
  }

  const folderStats = useMemo(
    () =>
      new Map(
        foldersWithFeeds?.map(folder => [
          folder.id,
          {
            starred: folder.feeds.reduce((acc, feed) => acc + (status?.state.get(feed.id)?.starred ?? 0), 0),
            unread: folder.feeds.reduce((acc, feed) => acc + (status?.state.get(feed.id)?.unread ?? 0), 0),
          },
        ]),
      ),
    [foldersWithFeeds, status],
  )
  const totalUnread = useMemo(
    () =>
      status?.state
        .values()
        .reduce((acc, state) => acc + state.unread, 0)
        .toString(),
    [status],
  )
  const totalStarred = useMemo(
    () =>
      status?.state
        .values()
        .reduce((acc, state) => acc + state.starred, 0)
        .toString(),
    [status],
  )
  // biome-ignore lint/correctness/useExhaustiveDependencies(foldersWithFeeds):
  // biome-ignore lint/correctness/useExhaustiveDependencies(feedsWithoutFolders):
  // biome-ignore lint/correctness/useExhaustiveDependencies(status?.state.get):
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshed):
  const [hiddenFolderIds, hiddenFeedIds] = useMemo(() => {
    const folders = new Set<number>()
    const feeds = new Set<number>()
    if (filter === 'Feeds') return [folders, feeds]

    for (const folder of foldersWithFeeds ?? []) {
      let hideFolder = true
      for (const feed of folder.feeds)
        if (
          selected?.feed_id !== feed.id &&
          !(filter === 'Unread' ? status?.state.get(feed.id)?.unread : status?.state.get(feed.id)?.starred)
        )
          feeds.add(feed.id)
        else hideFolder = false
      if (hideFolder && selected?.folder_id !== folder.id) folders.add(folder.id)
    }
    for (const feed of feedsWithoutFolders ?? [])
      if (
        selected?.feed_id !== feed.id &&
        !(filter === 'Unread' ? status?.state.get(feed.id)?.unread : status?.state.get(feed.id)?.starred)
      )
        feeds.add(feed.id)

    return [folders, feeds]
  }, [filter, selected, refreshed])

  return (
    <div style={{ ...style, ...panelStyle }}>
      <div className="topbar" style={{ justifyContent: 'space-between' }}>
        <Button
          icon={
            settings.dark_theme ? (
              <Sun
                {...iconProps}
                fill={Colors.ORANGE5}
                stroke={Colors.ORANGE4}
                filter={`drop-shadow(0 0 1px ${Colors.ORANGE5})`}
              />
            ) : (
              <Moon
                {...iconProps}
                stroke={Colors.DARK_GRAY3}
                strokeWidth={1.5}
                filter={`drop-shadow(0 0 0.5px ${Colors.DARK_GRAY3})`}
              />
            )
          }
          onClick={async () => {
            await xfetch('api/settings', {
              method: 'PUT',
              body: JSON.stringify({ dark_theme: !settings.dark_theme }),
            })
            setSettings(settings => ({ ...settings, dark_theme: !settings.dark_theme }))
          }}
          minimal
        />
        <ButtonGroup outlined>
          {(
            [
              { value: 'Unread', title: 'Unread', icon: <Circle {...iconProps} /> },
              { value: 'Feeds', title: 'All', icon: <MenuIcon {...iconProps} /> },
              { value: 'Starred', title: 'Starred', icon: <Star {...iconProps} /> },
            ] as const
          ).map(({ value, title, icon }) => (
            <Button
              key={value}
              className="filter"
              intent={Intent.PRIMARY}
              icon={icon}
              title={title}
              active={value === filter}
              onClick={() => setFilter(value)}
            />
          ))}
        </ButtonGroup>
        <Popover
          placement="bottom"
          transitionDuration={0}
          content={
            <Menu>
              <MenuItem
                text="New Feed"
                icon={<Plus {...menuIconProps} />}
                onClick={() => setNewFeedDialogOpen(true)}
              />
              <TextEditor
                placeholder="Folder title"
                onConfirm={async title => {
                  if (!title) throw new Error('Folder title is required')
                  const folder = await xfetch<Folder>('api/folders', {
                    method: 'POST',
                    body: JSON.stringify({ title }),
                  })
                  setFolders(folders => folders && [...folders, folder].toSorted(compareTitle))
                  setSelected({ folder_id: folder.id })
                }}
              >
                <MenuItem text="New Folder" icon={<Plus {...menuIconProps} />} shouldDismissPopover={false} />
              </TextEditor>
              <MenuDivider />
              <Tooltip
                content={
                  status?.last_refreshed ? (
                    <small>Last Refreshed: {fromNow(new Date(status.last_refreshed), true)}</small>
                  ) : undefined
                }
                intent={Intent.PRIMARY}
                placement="right"
                modifiers={{ offset: { enabled: true, options: { offset: [0, 6] } } }}
                compact
              >
                <MenuItem
                  text="Refresh Feeds"
                  icon={<RotateCw {...menuIconProps} />}
                  disabled={!!status?.running}
                  onClick={async () => {
                    await xfetch('api/feeds/refresh', { method: 'POST' })
                    await refreshStats()
                  }}
                />
              </Tooltip>
              <RefreshRateEditor
                defaultValue={settings?.refresh_rate}
                inputRef={refreshRateRef}
                onConfirm={async () => {
                  if (!refreshRateRef.current) return
                  const refreshRate = Number.parseInt(refreshRateRef.current.value)
                  await xfetch('api/settings', {
                    method: 'PUT',
                    body: JSON.stringify({ refresh_rate: refreshRate }),
                  })
                  setSettings(settings => settings && { ...settings, refresh_rate: refreshRate })
                }}
              >
                <MenuItem
                  text="Change Refresh Rate"
                  icon={<Edit {...menuIconProps} />}
                  shouldDismissPopover={false}
                />
              </RefreshRateEditor>
              <MenuDivider />
              <form ref={opmlFormRef}>
                <FileInput
                  style={{ display: 'none' }}
                  inputProps={{ name: 'opml', id: 'opml-import' }}
                  onInputChange={async () => {
                    if (!opmlFormRef.current) return
                    await xfetch('api/opml/import', {
                      method: 'POST',
                      body: new FormData(opmlFormRef.current),
                    })
                    menuCloserRef.current?.click()
                    await Promise.all([refreshFeeds(), refreshStats()])
                  }}
                />
                <label htmlFor="opml-import">
                  <MenuItem
                    text="Import OPML File"
                    icon={<Download {...menuIconProps} />}
                    onClick={evt => evt.stopPropagation()}
                  />
                </label>
                <div className={Classes.POPOVER_DISMISS} style={{ display: 'none' }} ref={menuCloserRef} />
              </form>
              <MenuItem text="Export OPML File" href="api/opml/export" icon={<Upload {...menuIconProps} />} />
            </Menu>
          }
        >
          <Button icon={<MoreHorizontal {...iconProps} />} minimal />
        </Popover>
      </div>
      <Divider />
      <Tree<Selected>
        contents={[
          {
            id: '',
            label: `All ${filter}`,
            isSelected: !selected,
            secondaryLabel:
              filter === 'Unread' ? totalUnread : filter === 'Starred' ? totalStarred : undefined,
          },
          ...(feedsWithoutFolders ?? []).filter(f => !hiddenFeedIds.has(f.id)).map(f => feed(f)),
          ...(foldersWithFeeds ?? [])
            .filter(f => !hiddenFolderIds.has(f.id))
            .map(
              folder =>
                ({
                  id: `folder:${folder.id}`,
                  label: (
                    <>
                      <ContextMenu
                        content={({ isOpen }) => (
                          <Menu>
                            <TextEditor
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
                            >
                              <MenuItem
                                text="Rename"
                                icon={<Edit {...menuIconProps} />}
                                shouldDismissPopover={false}
                              />
                            </TextEditor>
                            <MenuItem
                              text="Refresh"
                              icon={<RotateCw {...menuIconProps} />}
                              disabled={!!status?.running}
                              onClick={async () => {
                                await xfetch(`api/folders/${folder.id}/refresh`, {
                                  method: 'POST',
                                })
                                await refreshStats()
                              }}
                            />
                            <Deleter
                              isOpen={isOpen}
                              onConfirm={async () => {
                                await xfetch(`api/folders/${folder.id}`, { method: 'DELETE' })
                                const deletedFeeds = new Set(
                                  foldersById.get(folder.id)?.feeds.map(feed => feed.id),
                                )
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
                                setSelected(undefined)
                              }}
                            />
                          </Menu>
                        )}
                        onContextMenu={() =>
                          setSelected(selected =>
                            selected?.folder_id === folder.id ? selected : { folder_id: folder.id },
                          )
                        }
                      >
                        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }} title={folder.title}>
                          {folder.title || 'untitled'}
                        </span>
                      </ContextMenu>
                    </>
                  ),
                  isExpanded: folder.is_expanded,
                  isSelected: selected?.folder_id === folder.id,
                  childNodes: folder.feeds.filter(f => !hiddenFeedIds.has(f.id)).map(f => feed(f)),
                  secondaryLabel: secondaryLabel(folderStats.get(folder.id)),
                  nodeData: { folder_id: folder.id },
                }) satisfies TreeNodeInfo,
            ),
        ]}
        onNodeExpand={setExpanded(true)}
        onNodeCollapse={setExpanded(false)}
        onNodeClick={node =>
          JSON.stringify(selected) !== JSON.stringify(node.nodeData) && setSelected(node.nodeData)
        }
      />
      {status?.running ? (
        <>
          <Divider />
          <div style={statusBarStyle}>
            <Spinner style={{ marginLeft: length(3), marginRight: length(2) }} size={15} />
            Refreshing ({status.running} left)
          </div>
        </>
      ) : errorCount ? (
        <>
          <Divider />
          <div style={statusBarStyle}>
            <AlertCircle style={{ marginLeft: length(3), marginRight: length(2) }} {...iconProps} />
            {errorCount} feeds have errors
          </div>
        </>
      ) : undefined}
      <NewFeedDialog
        isOpen={newFeedDialogOpen}
        setIsOpen={setNewFeedDialogOpen}
        defaultFolderId={selected && (selected.folder_id ?? feedsById.get(selected.feed_id)?.folder_id)}
        folders={folders}
        setFeeds={setFeeds}
        setStatus={setStatus}
        setSelected={setSelected}
      />
    </div>
  )
}

function TextEditor({
  defaultValue,
  placeholder,
  textAreaStyle,
  onConfirm,
  children,
}: {
  defaultValue?: string
  placeholder?: string
  textAreaStyle?: CSSProperties
  onConfirm: (value: string) => Promise<void>
  children: ReactNode
}) {
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const closerRef = useRef<HTMLDivElement>(null)
  const confirm = async () => {
    if (inputRef.current == null) return
    setLoading(true)
    try {
      await onConfirm(inputRef.current.value)
      closerRef.current?.click()
    } finally {
      setLoading(false)
    }
  }

  return (
    <Popover
      usePortal={false}
      placement="right"
      transitionDuration={0}
      modifiers={{
        flip: { enabled: true },
        offset: { enabled: true, options: { offset: [0, 4] } },
      }}
      shouldReturnFocusOnClose
      content={
        <>
          <TextArea
            defaultValue={defaultValue}
            placeholder={placeholder}
            inputRef={inputRef}
            cols={30}
            spellCheck="false"
            disabled={loading}
            autoResize
            style={{
              borderBottomLeftRadius: 0,
              borderBottomRightRadius: 0,
              ...textAreaStyle,
            }}
            onKeyDown={async evt => {
              if (evt.key === 'Enter') {
                evt.preventDefault()
                await confirm()
              }
            }}
          />
          <Button
            loading={loading}
            style={{ width: '100%', borderTopLeftRadius: 0, borderTopRightRadius: 0 }}
            intent={Intent.PRIMARY}
            text="OK"
            onClick={confirm}
          />
          <div className={Classes.POPOVER_DISMISS} style={{ display: 'none' }} ref={closerRef} />
        </>
      }
      onOpening={node => {
        const elem = node.querySelector<HTMLInputElement>('.bp5-input')
        if (elem) {
          elem.focus()
          elem.setSelectionRange(elem.value.length, elem.value.length)
        }
      }}
    >
      {children}
    </Popover>
  )
}

function RefreshRateEditor({
  inputRef,
  defaultValue,
  onConfirm,
  children,
}: {
  defaultValue?: number
  inputRef: RefObject<HTMLInputElement>
  onConfirm: () => Promise<void>
  children: ReactNode
}) {
  const [loading, setLoading] = useState(false)
  const closerRef = useRef<HTMLDivElement>(null)
  const confirm = async () => {
    setLoading(true)
    try {
      await onConfirm()
      closerRef.current?.click()
    } finally {
      setLoading(false)
    }
  }

  return (
    <Popover
      usePortal={false}
      placement="right"
      transitionDuration={0}
      modifiers={{
        flip: { enabled: true },
        offset: { enabled: true, options: { offset: [0, 3] } },
      }}
      shouldReturnFocusOnClose
      content={
        <>
          <Tooltip
            content={<small>minutes</small>}
            intent={Intent.PRIMARY}
            placement="top"
            modifiers={{ offset: { enabled: true, options: { offset: [0, 5] } } }}
            compact
          >
            <NumericInput
              defaultValue={defaultValue}
              inputRef={inputRef}
              buttonPosition="none"
              min={0}
              stepSize={30}
              minorStepSize={1}
              majorStepSize={60}
              disabled={loading}
              style={{ width: '120px' }}
            />
          </Tooltip>
          <Button
            loading={loading}
            style={{ width: '100%', borderTopLeftRadius: 0, borderTopRightRadius: 0 }}
            intent={Intent.PRIMARY}
            text="OK"
            onClick={confirm}
          />
          <div className={Classes.POPOVER_DISMISS} style={{ display: 'none' }} ref={closerRef} />
        </>
      }
      onOpening={node => node.querySelector<HTMLInputElement>('.bp5-input')?.focus()}
    >
      {children}
    </Popover>
  )
}

function Deleter({ isOpen, onConfirm }: { isOpen: boolean; onConfirm: () => Promise<void> }) {
  const [state, setState] = useState<boolean>()
  if (!isOpen && state === false) setState(undefined)

  return (
    <MenuItem
      text={`Delete${state === false ? ' (confirm)' : ''}`}
      className={cn(state != null && Classes.ACTIVE)}
      disabled={state}
      icon={state ? <Spinner {...menuIconProps} intent={Intent.DANGER} /> : <Trash {...menuIconProps} />}
      intent={Intent.DANGER}
      shouldDismissPopover={false}
      onClick={async () => {
        if (state === false) {
          setState(true)
          try {
            await onConfirm()
          } finally {
            setState(undefined)
          }
        } else if (state === undefined) {
          setState(false)
        }
      }}
    />
  )
}
