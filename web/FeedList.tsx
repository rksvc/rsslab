import {
  Button,
  ButtonGroup,
  ButtonVariant,
  Classes,
  Colors,
  Divider,
  FileInput,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  NumericInput,
  Popover,
  Spinner,
  Tooltip,
  Tree,
  type TreeNodeInfo,
} from '@blueprintjs/core'
import { type CSSProperties, type RefObject, useMemo, useRef, useState } from 'react'
import {
  AlertCircle,
  Circle,
  Download,
  Edit,
  Menu as MenuIcon,
  Moon,
  MoreHorizontal,
  Plus,
  RotateCw,
  Star,
  Sun,
  Upload,
} from 'react-feather'
import { useMyContext } from './Context.tsx'
import FeedIcon from './FeedIcon.tsx'
import { NewFeedDialog } from './NewFeed.tsx'
import RelativeTime from './RelativeTime.tsx'
import TextEditor from './TextEditor.tsx'
import type { Feed, FeedState, Folder, Selected } from './types.ts'
import { fromNow, length, menuModifiers, xfetch } from './utils.ts'

const statusBarStyle = {
  display: 'flex',
  alignItems: 'center',
  padding: length(1),
  overflowWrap: 'break-word',
} satisfies CSSProperties

export default function FeedList() {
  const {
    setFolders,
    status,
    settings,
    setSettings,

    filter,
    setFilter,
    selected,
    setSelected,
    refreshed,

    refreshFeeds,
    refreshStats,
    feedsOutsideFolders,
    foldersWithFeeds,
  } = useMyContext()
  const [newFeedDialogOpen, setNewFeedDialogOpen] = useState(false)
  const refreshRateRef = useRef<HTMLInputElement>(null)
  const opmlFormRef = useRef<HTMLFormElement>(null)
  const menuCloserRef = useRef<HTMLDivElement>(null)

  const secondaryLabel = (state?: FeedState) =>
    filter === 'Unread' ? (
      state?.unread.toString()
    ) : filter === 'Starred' ? (
      state?.starred.toString()
    ) : state?.error ? (
      <AlertCircle style={{ display: 'flex' }} />
    ) : undefined
  const feed = (feed: Feed) =>
    ({
      id: `feed:${feed.id}`,
      isSelected: selected?.feed_id === feed.id,
      secondaryLabel: secondaryLabel(status?.state.get(feed.id)),
      nodeData: { feed_id: feed.id },
      icon: <FeedIcon feed={feed}  />,
      label: (
        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }} title={feed.title}>
          {feed.title || 'untitled'}
        </span>
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
  const [totalUnread, totalStarred, errorCount] = useMemo(
    () => [
      status?.state
        .values()
        .reduce((acc, state) => acc + state.unread, 0)
        .toString(),
      status?.state
        .values()
        .reduce((acc, state) => acc + state.starred, 0)
        .toString(),
      status?.state.values().reduce((acc, state) => acc + (state.error ? 1 : 0), 0),
    ],
    [status],
  )
  // biome-ignore lint/correctness/useExhaustiveDependencies(feedsOutsideFolders): controlled by `refreshed`
  // biome-ignore lint/correctness/useExhaustiveDependencies(foldersWithFeeds): controlled by `refreshed`
  // biome-ignore lint/correctness/useExhaustiveDependencies(status?.state.get): controlled by `refreshed`
  // biome-ignore lint/correctness/useExhaustiveDependencies(refreshed): controller
  const [hiddenFolders, hiddenFeeds] = useMemo(() => {
    if (filter === 'Feeds' || !feedsOutsideFolders || !foldersWithFeeds) return []

    const folders = new Set<number>()
    const feeds = new Set<number>()
    const hideFeed = (id: number) =>
      selected?.feed_id !== id &&
      !(filter === 'Unread' ? status?.state.get(id)?.unread : status?.state.get(id)?.starred)
    for (const folder of foldersWithFeeds) {
      let hideFolder = true
      for (const feed of folder.feeds)
        if (hideFeed(feed.id)) feeds.add(feed.id)
        else hideFolder = false
      if (hideFolder && selected?.folder_id !== folder.id) folders.add(folder.id)
    }
    for (const feed of feedsOutsideFolders) if (hideFeed(feed.id)) feeds.add(feed.id)

    return [folders, feeds]
  }, [filter, selected, refreshed])

  return (
    <div id="feed-list">
      <div className="topbar" style={{ justifyContent: 'space-between' }}>
        <Button
          icon={
            settings.dark_theme ? (
              <Sun
                fill={Colors.ORANGE5}
                stroke={Colors.ORANGE4}
                filter={`drop-shadow(0 0 1px ${Colors.ORANGE5})`}
              />
            ) : (
              <Moon
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
          variant={ButtonVariant.MINIMAL}
        />
        <ButtonGroup variant={ButtonVariant.OUTLINED}>
          {(
            [
              { value: 'Unread', title: 'Unread', icon: <Circle /> },
              { value: 'Feeds', title: 'All', icon: <MenuIcon /> },
              { value: 'Starred', title: 'Starred', icon: <Star /> },
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
          transitionDuration={0}
          modifiers={menuModifiers}
          content={
            <Menu>
              <MenuItem text="New Feed" icon={<Plus />} onClick={() => setNewFeedDialogOpen(true)} />
              <TextEditor
                menuText="New Folder"
                menuIcon={<Plus />}
                placeholder="Folder title"
                onConfirm={async title => {
                  if (!title) throw new Error('Folder title is required')
                  const folder = await xfetch<Folder>('api/folders', {
                    method: 'POST',
                    body: JSON.stringify({ title }),
                  })
                  await refreshFeeds()
                  setSelected({ folder_id: folder.id })
                }}
              />
              <MenuDivider
                className="last-refreshed"
                title={
                  status?.last_refreshed ? (
                    <small>
                      Last refreshed:{' '}
                      <RelativeTime date={status.last_refreshed} format={date => fromNow(new Date(date))} />
                    </small>
                  ) : undefined
                }
              />
              <MenuItem
                text="Refresh Feeds"
                icon={<RotateCw />}
                disabled={!!status?.running}
                onClick={async () => {
                  await xfetch('api/feeds/refresh', { method: 'POST' })
                  await refreshStats()
                }}
              />
              <RefreshRateEditor
                defaultValue={settings?.refresh_rate}
                inputRef={refreshRateRef}
                onConfirm={async () => {
                  if (!refreshRateRef.current) return
                  const refreshRate = +refreshRateRef.current.value
                  await xfetch('api/settings', {
                    method: 'PUT',
                    body: JSON.stringify({ refresh_rate: refreshRate }),
                  })
                  setSettings(settings => settings && { ...settings, refresh_rate: refreshRate })
                }}
              />
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
                  <MenuItem text="Import OPML File" icon={<Download />} shouldDismissPopover={false} />
                </label>
                <div className={Classes.POPOVER_DISMISS} ref={menuCloserRef} hidden />
              </form>
              <MenuItem text="Export OPML File" href="api/opml/export" icon={<Upload />} />
            </Menu>
          }
        >
          <Button icon={<MoreHorizontal />} variant={ButtonVariant.MINIMAL} />
        </Popover>
      </div>
      <Divider compact />
      <Tree<Selected>
        contents={[
          {
            id: '',
            label: `All ${filter}`,
            isSelected: selected === null,
            secondaryLabel:
              filter === 'Unread' ? totalUnread : filter === 'Starred' ? totalStarred : undefined,
            nodeData: null,
          },
          ...(feedsOutsideFolders ?? []).filter(({ id }) => !hiddenFeeds?.has(id)).map(f => feed(f)),
          ...(foldersWithFeeds ?? [])
            .filter(({ id }) => !hiddenFolders?.has(id))
            .map(
              folder =>
                ({
                  id: `folder:${folder.id}`,
                  label: (
                    <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }} title={folder.title}>
                      {folder.title || 'untitled'}
                    </span>
                  ),
                  isExpanded: folder.is_expanded,
                  isSelected: selected?.folder_id === folder.id,
                  childNodes: folder.feeds.filter(({ id }) => !hiddenFeeds?.has(id)).map(f => feed(f)),
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
          <Divider compact />
          <div style={statusBarStyle}>
            <Spinner style={{ marginLeft: length(3), marginRight: length(2) }} size={15} />
            Refreshing ({status.running} left)
          </div>
        </>
      ) : errorCount ? (
        <>
          <Divider compact />
          <div style={statusBarStyle}>
            <AlertCircle style={{ marginLeft: length(3), marginRight: length(2) }} />
            {errorCount} feed{errorCount === 1 ? ' has an error' : 's have errors'}
          </div>
        </>
      ) : undefined}
      <NewFeedDialog isOpen={newFeedDialogOpen} setIsOpen={setNewFeedDialogOpen} />
    </div>
  )
}

function RefreshRateEditor({
  inputRef,
  defaultValue,
  onConfirm,
}: {
  defaultValue?: number
  inputRef: RefObject<HTMLInputElement>
  onConfirm: () => Promise<void>
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
      modifiers={{ offset: { options: { offset: [0, 3] } } }}
      shouldReturnFocusOnClose
      content={
        <>
          <Tooltip
            content={<small>minutes</small>}
            intent={Intent.PRIMARY}
            placement="top"
            modifiers={{ offset: { options: { offset: [0, 5] } } }}
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
              onKeyDown={async evt => {
                if (evt.key === 'Enter') {
                  evt.preventDefault()
                  await confirm()
                }
              }}
            />
          </Tooltip>
          <Button loading={loading} intent={Intent.PRIMARY} text="OK" onClick={confirm} fill />
          <div className={Classes.POPOVER_DISMISS} ref={closerRef} hidden />
        </>
      }
      onOpening={node => node.querySelector<HTMLInputElement>(`.${Classes.INPUT}`)?.focus()}
    >
      <MenuItem text="Change Refresh Rate" icon={<Edit />} shouldDismissPopover={false} />
    </Popover>
  )
}
