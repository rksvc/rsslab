import {
  Button,
  ButtonGroup,
  ContextMenu,
  type ContextMenuChildrenProps,
  Divider,
  FileInput,
  HTMLSelect,
  InputGroup,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  NumericInput,
  Popover,
  Spinner,
  Tree,
  type TreeNodeInfo,
} from '@blueprintjs/core'
import {
  type Dispatch,
  type SetStateAction,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  AlertCircle,
  Circle,
  Download,
  Edit,
  Folder as FolderIcon,
  FolderMinus,
  Link,
  Menu as MenuIcon,
  MoreHorizontal,
  Move,
  Plus,
  RotateCw,
  Rss,
  Star,
  Trash,
  Upload,
  Wind,
} from 'react-feather'
import { Confirm } from './Confirm'
import type { Feed, Folder, FolderWithFeeds, Settings, Stats } from './types'
import { cn, iconProps, menuIconProps, popoverProps, xfetch } from './utils'

export default function FeedList({
  filter,
  setFilter,
  folders,
  setFolders,
  setFeeds,
  stats,
  errors,
  selected,
  setSelected,
  loadingFeeds,
  settings,
  setSettings,

  refreshFeeds,
  refreshStats,
  foldersWithFeeds,
  feedsWithoutFolders,
  feedsById,
}: {
  filter: string
  setFilter: Dispatch<SetStateAction<string>>
  folders?: Folder[]
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>
  setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
  stats?: Map<number, Stats>
  errors?: Map<number, string>
  selected: string
  setSelected: Dispatch<SetStateAction<string>>
  loadingFeeds: number
  settings?: Settings
  setSettings: Dispatch<SetStateAction<Settings | undefined>>

  refreshFeeds: () => Promise<void>
  refreshStats: (loop?: boolean) => Promise<void>
  foldersWithFeeds?: FolderWithFeeds[]
  feedsWithoutFolders?: Feed[]
  feedsById: Map<number, Feed>
}) {
  const [selectedFolder, setSelectedFolder] = useState(
    getSelectedFolder(selected, feedsById),
  )
  useEffect(
    () => setSelectedFolder(getSelectedFolder(selected, feedsById)),
    [selected, feedsById],
  )

  const [creatingNewFeed, setCreatingNewFeed] = useState(false)
  const [creatingNewFolder, setCreatingNewFolder] = useState(false)
  const [newFeedDialogOpen, setNewFeedDialogOpen] = useState(false)
  const [newFolderDialogOpen, setNewFolderDialogOpen] = useState(false)
  const [changeRefreshRateDialogOpen, setChangeRefreshRateDialogOpen] = useState(false)
  const [renameFeed, setRenameFeed] = useState<Feed>()
  const [changeLink, setChangeLink] = useState<Feed>()
  const [deleteFeed, setDeleteFeed] = useState<Feed>()
  const [renameFolder, setRenameFolder] = useState<Folder>()
  const [deleteFolder, setDeleteFolder] = useState<Folder>()
  const menuRef = useRef<HTMLButtonElement>(null)
  const newFeedLinkRef = useRef<HTMLInputElement>(null)
  const newFolderTitleRef = useRef<HTMLInputElement>(null)
  const refreshRateRef = useRef<HTMLInputElement>(null)
  const opmlFormRef = useRef<HTMLFormElement>(null)
  const feedTitleRef = useRef<HTMLInputElement>(null)
  const feedLinkRef = useRef<HTMLInputElement>(null)
  const folderTitleRef = useRef<HTMLInputElement>(null)

  const updateFeedAttr = async <T extends 'title' | 'feed_link' | 'folder_id'>(
    id: number,
    attrName: T,
    value: Feed[T],
  ) => {
    await xfetch(`api/feeds/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ [attrName]: value ?? -1 }),
    })
    setFeeds(feeds =>
      feeds?.map(feed => (feed.id === id ? { ...feed, [attrName]: value } : feed)),
    )
  }
  const secondaryLabel = (stats?: Stats, error?: boolean, lastRefreshed?: string) =>
    filter === 'Unread' ? (
      `${stats?.unread ?? ''}`
    ) : filter === 'Starred' ? (
      `${stats?.starred ?? ''}`
    ) : error ? (
      <span
        className="flex"
        title={
          lastRefreshed && `Last Refreshed: ${new Date(lastRefreshed).toLocaleString()}`
        }
      >
        <AlertCircle {...iconProps} />
      </span>
    ) : (
      ''
    )
  const feed = (feed: Feed) => ({
    id: `feed:${feed.id}`,
    label: (
      <ContextMenu
        content={
          <Menu>
            {feed.link && (
              <MenuItem
                text="Website"
                icon={<Link {...menuIconProps} />}
                target="_blank"
                href={feed.link}
              />
            )}
            <MenuItem
              text="Feed Link"
              icon={<Rss {...menuIconProps} />}
              target="_blank"
              href={(() => {
                const feedLink = feed.feed_link
                return feedLink?.startsWith('rsshub:')
                  ? `/${feedLink.slice('rsshub:'.length)}`
                  : feedLink
              })()}
            />
            <MenuDivider />
            <MenuItem
              text="Rename"
              icon={<Edit {...menuIconProps} />}
              onClick={() => setRenameFeed(feed)}
            />
            <MenuItem
              text="Change Link"
              icon={<Edit {...menuIconProps} />}
              onClick={() => setChangeLink(feed)}
            />
            <MenuItem
              text="Refresh"
              icon={<RotateCw {...menuIconProps} />}
              disabled={loadingFeeds > 0}
              onClick={async () => {
                await xfetch(`api/feeds/${feed.id}/refresh`, { method: 'POST' })
                await refreshStats()
              }}
            />
            <MenuItem
              text="Move to..."
              icon={<Move {...menuIconProps} />}
              disabled={!folders?.length}
            >
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
              {feed.folder_id != null && (
                <MenuItem
                  text="--"
                  icon={<FolderMinus {...menuIconProps} />}
                  onClick={() => updateFeedAttr(feed.id, 'folder_id', null)}
                />
              )}
            </MenuItem>
            <MenuItem
              text="Delete"
              icon={<Trash {...menuIconProps} />}
              intent={Intent.DANGER}
              onClick={() => setDeleteFeed(feed)}
            />
          </Menu>
        }
      >
        {(ctxMenuProps: ContextMenuChildrenProps) => (
          <div
            className={cn(
              ctxMenuProps.className,
              ctxMenuProps.contentProps.isOpen && 'context-menu-open',
            )}
            onContextMenu={ctxMenuProps.onContextMenu}
            ref={ctxMenuProps.ref}
          >
            {ctxMenuProps.popover}
            <span title={feed.title}>{feed.title}</span>
          </div>
        )}
      </ContextMenu>
    ),
    icon: feed.has_icon ? (
      <img className="w-4 mr-[7px]" src={`api/feeds/${feed.id}/icon`} />
    ) : (
      <Rss className="mr-[6px]" {...iconProps} />
    ),
    isSelected: selected === `feed:${feed.id}`,
    secondaryLabel: secondaryLabel(
      stats?.get(feed.id),
      !!errors?.get(feed.id),
      feed.last_refreshed,
    ),
  })
  const setExpanded = (isExpanded: boolean) => async (node: TreeNodeInfo) => {
    const id = Number.parseInt(`${node.id}`.split(':')[1])
    setFolders(folders =>
      folders?.map(folder =>
        folder.id === id ? { ...folder, is_expanded: isExpanded } : folder,
      ),
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
            starred: folder.feeds.reduce(
              (acc, feed) => acc + (stats?.get(feed.id)?.starred ?? 0),
              0,
            ),
            unread: folder.feeds.reduce(
              (acc, feed) => acc + (stats?.get(feed.id)?.unread ?? 0),
              0,
            ),
          },
        ]),
      ),
    [foldersWithFeeds, stats],
  )
  const total = useMemo(
    () =>
      filter !== 'Feeds' &&
      `${[...(stats?.values() ?? [])].reduce(
        (acc, stats) => acc + (filter === 'Unread' ? stats.unread : stats.starred),
        0,
      )}`,
    [stats, filter],
  )
  const visibleFolders =
    filter === 'Feeds'
      ? foldersWithFeeds
      : foldersWithFeeds
          ?.map(folder => ({
            ...folder,
            feeds: folder.feeds.filter(
              feed =>
                selected === `feed:${feed.id}` ||
                (filter === 'Unread'
                  ? (stats?.get(feed.id)?.unread ?? 0)
                  : (stats?.get(feed.id)?.starred ?? 0)) > 0,
            ),
          }))
          .filter(folder => folder.feeds.length > 0 || selected === `folder:${folder.id}`)
  const visibleFeeds =
    filter === 'Feeds'
      ? feedsWithoutFolders
      : feedsWithoutFolders?.filter(
          feed =>
            selected === `feed:${feed.id}` ||
            (filter === 'Unread'
              ? (stats?.get(feed.id)?.unread ?? 0)
              : (stats?.get(feed.id)?.starred ?? 0)) > 0,
        )

  return (
    <div className="flex flex-col min-h-screen max-h-screen">
      <div className="flex flex-row justify-between items-center">
        <Wind className="ml-3 mr-3" {...iconProps} />
        <ButtonGroup className="min-h-10 max-h-10" outlined>
          {[
            { value: 'Unread', title: 'Unread', icon: <Circle {...iconProps} /> },
            { value: 'Feeds', title: 'All', icon: <MenuIcon {...iconProps} /> },
            { value: 'Starred', title: 'Starred', icon: <Star {...iconProps} /> },
          ].map(option => (
            <Button
              key={option.value}
              className="my-1"
              intent={Intent.PRIMARY}
              icon={option.icon}
              title={option.title}
              active={option.value === filter}
              onClick={() => setFilter(option.value)}
            />
          ))}
        </ButtonGroup>
        <Popover
          placement="bottom"
          {...popoverProps}
          content={
            <Menu>
              <MenuItem
                text="New Feed"
                disabled={creatingNewFeed}
                icon={<Plus {...menuIconProps} />}
                onClick={() => setNewFeedDialogOpen(true)}
              />
              <MenuItem
                text="New Folder"
                disabled={creatingNewFolder}
                icon={<Plus {...menuIconProps} />}
                onClick={() => setNewFolderDialogOpen(true)}
              />
              <MenuDivider />
              <MenuItem
                text="Refresh Feeds"
                icon={<RotateCw {...menuIconProps} />}
                disabled={loadingFeeds > 0}
                onClick={async () => {
                  await xfetch('api/feeds/refresh', { method: 'POST' })
                  await refreshStats()
                }}
              />
              <MenuDivider className="select-none" />
              <MenuItem
                text="Change Refresh Rate"
                icon={<Edit {...menuIconProps} />}
                onClick={() => setChangeRefreshRateDialogOpen(true)}
              />
              <MenuDivider className="select-none" />
              <form ref={opmlFormRef}>
                <FileInput
                  className="hidden"
                  inputProps={{ name: 'opml', id: 'opml-import' }}
                  onInputChange={async () => {
                    if (!opmlFormRef.current) return
                    await xfetch('api/opml/import', {
                      method: 'POST',
                      body: new FormData(opmlFormRef.current),
                    })
                    menuRef.current?.parentElement?.click()
                    refreshFeeds()
                    refreshStats()
                  }}
                />
                <label htmlFor="opml-import">
                  <MenuItem
                    text="Import OPML File"
                    icon={<Download {...menuIconProps} />}
                    onClick={event => event.stopPropagation()}
                  />
                </label>
              </form>
              <MenuItem
                text="Export OPML File"
                href="api/opml/export"
                icon={<Upload {...menuIconProps} />}
              />
            </Menu>
          }
        >
          <Button
            ref={menuRef}
            className="mr-1"
            icon={<MoreHorizontal {...iconProps} />}
            minimal
          />
        </Popover>
      </div>
      <Divider className="m-0" />
      <Tree
        className="overflow-auto grow"
        contents={[
          {
            id: 0,
            label: `All ${filter}`,
            isSelected: !selected,
            secondaryLabel: total,
          },
          ...(visibleFeeds ?? []).map(f => feed(f)),
          ...(visibleFolders ?? []).map(folder => ({
            id: `folder:${folder.id}`,
            label: (
              <>
                <ContextMenu
                  content={
                    <Menu>
                      <MenuItem
                        text="Rename"
                        icon={<Edit {...menuIconProps} />}
                        onClick={() => setRenameFolder(folder)}
                      />
                      <MenuItem
                        text="Refresh"
                        icon={<RotateCw {...menuIconProps} />}
                        disabled={loadingFeeds > 0}
                        onClick={async () => {
                          await xfetch(`api/folders/${folder.id}/refresh`, {
                            method: 'POST',
                          })
                          await refreshStats()
                        }}
                      />
                      <MenuItem
                        text="Delete"
                        icon={<Trash {...menuIconProps} />}
                        intent={Intent.DANGER}
                        onClick={() => setDeleteFolder(folder)}
                      />
                    </Menu>
                  }
                >
                  {(ctxMenuProps: ContextMenuChildrenProps) => (
                    <div
                      className={cn(
                        ctxMenuProps.className,
                        ctxMenuProps.contentProps.isOpen && 'context-menu-open',
                      )}
                      onContextMenu={ctxMenuProps.onContextMenu}
                      ref={ctxMenuProps.ref}
                    >
                      {ctxMenuProps.popover}
                      <span title={folder.title}>{folder.title}</span>
                    </div>
                  )}
                </ContextMenu>
              </>
            ),
            isExpanded: folder.is_expanded,
            isSelected: selected === `folder:${folder.id}`,
            childNodes: folder.feeds.map(f => feed(f)),
            secondaryLabel: secondaryLabel(folderStats.get(folder.id)),
          })),
        ]}
        onNodeExpand={setExpanded(true)}
        onNodeCollapse={setExpanded(false)}
        onNodeClick={node => setSelected(typeof node.id === 'number' ? '' : node.id)}
      />
      <Divider className="m-0" />
      <div className="flex items-center p-1 break-words">
        {loadingFeeds > 0 ? (
          <>
            <Spinner className="ml-3 mr-2" size={15} />
            Refreshing ({loadingFeeds} left)
          </>
        ) : errors?.size ? (
          <>
            <AlertCircle className="ml-3 mr-2" {...iconProps} />
            {errors.size} feeds have errors
          </>
        ) : (
          <></>
        )}
      </div>
      <Confirm
        isOpen={newFeedDialogOpen}
        close={() => setNewFeedDialogOpen(false)}
        title="New Feed"
        callback={async () => {
          const url = newFeedLinkRef.current?.value
          if (!url) return
          setCreatingNewFeed(true)
          try {
            const feed = await xfetch<Feed>('api/feeds', {
              method: 'POST',
              body: JSON.stringify({
                url,
                folder_id: selectedFolder ? Number.parseInt(selectedFolder) : null,
              }),
            })
            await Promise.all([refreshFeeds(), refreshStats(false)])
            setSelected(`feed:${feed.id}`)
          } finally {
            setCreatingNewFeed(false)
          }
        }}
      >
        <div className="flex flex-row">
          <InputGroup
            placeholder="https://example.com/feed"
            inputRef={newFeedLinkRef}
            spellCheck={false}
            fill
          />
          <HTMLSelect
            className="ml-2"
            iconName="caret-down"
            options={[
              { value: '', label: '--' },
              ...(folders ?? []).map(folder => ({
                value: folder.id.toString(),
                label: folder.title,
              })),
            ]}
            value={selectedFolder}
            onChange={evt => setSelectedFolder(evt.currentTarget.value)}
          />
        </div>
      </Confirm>
      <Confirm
        isOpen={newFolderDialogOpen}
        close={() => setNewFolderDialogOpen(false)}
        title="New Folder"
        callback={async () => {
          const title = newFolderTitleRef.current?.value
          if (!title) return
          setCreatingNewFolder(true)
          try {
            const folder = await xfetch<Folder>('api/folders', {
              method: 'POST',
              body: JSON.stringify({ title }),
            })
            setFolders(
              folders =>
                folders &&
                [...folders, folder].toSorted((a, b) =>
                  a.title.toLocaleLowerCase().localeCompare(b.title.toLocaleLowerCase()),
                ),
            )
            setSelected(`folder:${folder.id}`)
          } finally {
            setCreatingNewFolder(false)
          }
        }}
      >
        <InputGroup className="ml-1" inputRef={newFolderTitleRef} />
      </Confirm>
      <Confirm
        isOpen={changeRefreshRateDialogOpen}
        close={() => setChangeRefreshRateDialogOpen(false)}
        title="Change Auto Refresh Rate (min)"
        callback={async () => {
          if (!refreshRateRef.current) return
          const refreshRate = Number.parseInt(refreshRateRef.current.value)
          setSettings(settings => settings && { ...settings, refresh_rate: refreshRate })
          await xfetch('api/settings', {
            method: 'PUT',
            body: JSON.stringify({ refresh_rate: refreshRate }),
          })
        }}
      >
        <NumericInput
          defaultValue={settings?.refresh_rate}
          inputRef={refreshRateRef}
          min={0}
          stepSize={30}
          minorStepSize={1}
          majorStepSize={60}
          fill
        />
      </Confirm>
      <Confirm
        isOpen={renameFeed}
        close={() => setRenameFeed(undefined)}
        title="Rename Feed"
        callback={async () => {
          if (!renameFeed) return
          const title = feedTitleRef.current?.value
          if (title && title !== renameFeed.title)
            await updateFeedAttr(renameFeed.id, 'title', title)
        }}
      >
        <InputGroup defaultValue={renameFeed?.title} inputRef={feedTitleRef} />
      </Confirm>
      <Confirm
        isOpen={changeLink}
        close={() => setChangeLink(undefined)}
        title="Change Feed Link"
        callback={async () => {
          if (!changeLink) return
          const feedLink = feedLinkRef.current?.value
          if (feedLink && feedLink !== changeLink.feed_link)
            await updateFeedAttr(changeLink.id, 'feed_link', feedLink)
        }}
      >
        <InputGroup
          defaultValue={changeLink?.feed_link}
          inputRef={feedLinkRef}
          spellCheck={false}
        />
      </Confirm>
      <Confirm
        isOpen={deleteFeed}
        close={() => setDeleteFeed(undefined)}
        title="Delete Feed"
        callback={async () => {
          if (!deleteFeed) return
          await xfetch(`api/feeds/${deleteFeed.id}`, { method: 'DELETE' })
          await Promise.all([refreshFeeds(), refreshStats(false)])
          setSelected(
            deleteFeed.folder_id === null ? '' : `folder:${deleteFeed.folder_id}`,
          )
        }}
        intent={Intent.DANGER}
      >
        Are you sure you want to delete {deleteFeed?.title ?? 'untitled'}?
      </Confirm>
      <Confirm
        isOpen={renameFolder}
        close={() => setRenameFolder(undefined)}
        title="Rename Folder"
        callback={async () => {
          if (!renameFolder) return
          const title = folderTitleRef.current?.value
          if (title && title !== renameFolder.title) {
            await xfetch(`api/folders/${renameFolder.id}`, {
              method: 'PUT',
              body: JSON.stringify({ title }),
            })
            setFolders(folders =>
              folders?.map(folder =>
                folder.id === renameFolder.id ? { ...folder, title } : folder,
              ),
            )
          }
        }}
      >
        <InputGroup defaultValue={renameFolder?.title} inputRef={folderTitleRef} />
      </Confirm>
      <Confirm
        isOpen={deleteFolder}
        close={() => setDeleteFolder(undefined)}
        title="Delete Folder"
        callback={async () => {
          if (!deleteFolder) return
          await xfetch(`api/folders/${deleteFolder.id}`, { method: 'DELETE' })
          await Promise.all([refreshFeeds(), refreshStats(false)])
          setSelected('')
        }}
        intent={Intent.DANGER}
      >
        Are you sure you want to delete {deleteFolder?.title || 'untitled'}?
      </Confirm>
    </div>
  )
}

function getSelectedFolder(selected: string, feedsById: Map<number, Feed>): string {
  const [type, id] = selected.split(':')
  return type === 'feed'
    ? (feedsById.get(Number.parseInt(id))?.folder_id?.toString() ?? '')
    : type === 'folder'
      ? id
      : ''
}
