import {
  AnchorButton,
  Button,
  ButtonGroup,
  Code,
  Collapse,
  ContextMenu,
  type ContextMenuChildrenProps,
  Divider,
  FileInput,
  FormGroup,
  HTMLSelect,
  InputGroup,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  NumericInput,
  Popover,
  RadioGroup,
  Spinner,
  TextArea,
  Tree,
  type TreeNodeInfo,
} from '@blueprintjs/core'
import {
  type Dispatch,
  type KeyboardEvent,
  type SetStateAction,
  useCallback,
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
import { Dialog } from './Dialog'
import type { Feed, Folder, FolderWithFeeds, Settings, Stats, Status } from './types'
import {
  cn,
  iconProps,
  length,
  menuIconProps,
  panelStyle,
  popoverProps,
  statusBarStyle,
  xfetch,
} from './utils'

const textAreaProps = {
  style: { fontFamily: 'inherit' },
  autoResize: true,
  fill: true,
  onKeyDown: (evt: KeyboardEvent<HTMLTextAreaElement>) =>
    evt.key === 'Enter' && evt.preventDefault(),
} as const

export default function FeedList({
  filter,
  setFilter,
  folders,
  setFolders,
  setFeeds,
  status,
  errors,
  selected,
  setSelected,
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
  status?: Status
  errors?: Map<number, string>
  selected: string
  setSelected: Dispatch<SetStateAction<string>>
  settings?: Settings
  setSettings: Dispatch<SetStateAction<Settings | undefined>>

  refreshFeeds: () => Promise<void>
  refreshStats: (loop?: boolean) => Promise<void>
  foldersWithFeeds?: FolderWithFeeds[]
  feedsWithoutFolders?: Feed[]
  feedsById: Map<number, Feed>
}) {
  const [type, id] = selected.split(':')
  const defaultSelectedFolder =
    type === 'feed'
      ? (feedsById.get(Number.parseInt(id))?.folder_id ?? '')
      : type === 'folder'
        ? id
        : ''

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

  const [showGenerator, setShowGenerator] = useState(false)
  const [generatorType, setGeneratorType] = useState('html')
  const transUrl = useRef<HTMLInputElement>(null)
  const transHtmlTitle = useRef<HTMLInputElement>(null)
  const transHtmlItems = useRef<HTMLInputElement>(null)
  const transHtmlItemTitle = useRef<HTMLInputElement>(null)
  const transHtmlItemUrl = useRef<HTMLInputElement>(null)
  const transHtmlItemUrlAttr = useRef<HTMLInputElement>(null)
  const transHtmlItemContent = useRef<HTMLInputElement>(null)
  const transHtmlItemDate = useRef<HTMLInputElement>(null)
  const transJsonHomePageUrl = useRef<HTMLInputElement>(null)
  const transJsonTitle = useRef<HTMLInputElement>(null)
  const transJsonHeaders = useRef<HTMLInputElement>(null)
  const transJsonItems = useRef<HTMLInputElement>(null)
  const transJsonItemTitle = useRef<HTMLInputElement>(null)
  const transJsonItemUrl = useRef<HTMLInputElement>(null)
  const transJsonItemUrlPrefix = useRef<HTMLInputElement>(null)
  const transJsonItemContent = useRef<HTMLInputElement>(null)
  const transJsonItemDatePublished = useRef<HTMLInputElement>(null)
  const transHtmlParams = useRef([
    {
      ref: transHtmlTitle,
      key: 'title',
      desc: 'Title of RSS',
      placeholder: 'extracted from <title>',
    },
    {
      ref: transHtmlItems,
      key: 'items',
      desc: 'CSS selector targetting items',
      placeholder: 'html',
    },
    {
      ref: transHtmlItemTitle,
      key: 'item_title',
      desc: 'CSS selector targetting title of item',
      placeholder: 'same as item element',
    },
    {
      ref: transHtmlItemUrl,
      key: 'item_url',
      desc: 'CSS selector targetting URL of item',
      placeholder: 'same as item element',
    },
    {
      ref: transHtmlItemUrlAttr,
      key: 'item_url_attr',
      desc: (
        <span>
          Attribute of <Code>item_url</Code> element as URL
        </span>
      ),
      placeholder: 'href',
    },
    {
      ref: transHtmlItemContent,
      key: 'item_content',
      desc: 'CSS selector targetting content of item',
      placeholder: 'same as item element',
    },
    {
      ref: transHtmlItemDate,
      key: 'item_date_published',
      desc: 'CSS selector targetting publication date of item',
      placeholder: 'same as item element',
    },
  ])
  const transJsonParams = useRef([
    {
      ref: transJsonHomePageUrl,
      key: 'home_page_url',
      desc: 'Home page URL of RSS',
    },
    {
      ref: transJsonTitle,
      key: 'title',
      desc: 'Title of RSS',
    },
    {
      ref: transJsonHeaders,
      key: 'headers',
      desc: 'HTTP request headers in JSON form',
    },
    {
      ref: transJsonItems,
      key: 'items',
      desc: 'JSON path to items',
      placeholder: 'entire JSON response',
    },
    {
      ref: transJsonItemTitle,
      key: 'item_title',
      desc: 'JSON path to title of item',
    },
    {
      ref: transJsonItemUrl,
      key: 'item_url',
      desc: 'JSON path to URL of item',
    },
    {
      ref: transJsonItemUrlPrefix,
      key: 'item_url_prefix',
      desc: 'Optional prefix for URL',
    },
    {
      ref: transJsonItemContent,
      key: 'item_content',
      desc: 'JSON path to content of item',
    },
    {
      ref: transJsonItemDatePublished,
      key: 'item_date_published',
      desc: 'JSON path to publication date of item',
    },
  ])
  const transParams = useCallback(
    () =>
      JSON.stringify({
        url: transUrl.current?.value,
        ...Object.fromEntries(
          (generatorType === 'html' ? transHtmlParams : transJsonParams).current
            .filter(({ ref }) => ref.current?.value)
            .map(({ key, ref }) => [key, ref.current!.value]),
        ),
      }),
    [generatorType],
  )

  const menuRef = useRef<HTMLButtonElement>(null)
  const [newFeedLink, setNewFeedLink] = useState('')
  const selectedFolderRef = useRef<HTMLSelectElement>(null)
  const newFolderTitleRef = useRef<HTMLTextAreaElement>(null)
  const refreshRateRef = useRef<HTMLInputElement>(null)
  const opmlFormRef = useRef<HTMLFormElement>(null)
  const feedTitleRef = useRef<HTMLTextAreaElement>(null)
  const feedLinkRef = useRef<HTMLTextAreaElement>(null)
  const folderTitleRef = useRef<HTMLTextAreaElement>(null)

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
        style={{ display: 'flex' }}
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
                const feedLink = feed.feed_link
                const i = feedLink.indexOf(':')
                if (i === -1) return feedLink
                const scheme = feedLink.slice(0, i)
                switch (scheme) {
                  case 'html':
                  case 'json':
                    return `api/transform/${scheme}/${encodeURIComponent(feedLink.slice(i + 1))}`
                  default:
                    return feedLink
                }
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
              disabled={!!status?.running}
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
          <span
            className={cn(
              ctxMenuProps.className,
              ctxMenuProps.contentProps.isOpen && 'context-menu-open',
            )}
            style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}
            title={feed.title}
            onContextMenu={ctxMenuProps.onContextMenu}
            ref={ctxMenuProps.ref}
          >
            {ctxMenuProps.popover}
            {feed.title}
          </span>
        )}
      </ContextMenu>
    ),
    icon: feed.has_icon ? (
      <img
        style={{ width: length(4), marginRight: '7px' }}
        src={`api/feeds/${feed.id}/icon`}
      />
    ) : (
      <span style={{ display: 'flex' }}>
        <Rss style={{ marginRight: '6px' }} {...iconProps} />
      </span>
    ),
    isSelected: selected === `feed:${feed.id}`,
    secondaryLabel: secondaryLabel(
      status?.stats.get(feed.id),
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
              (acc, feed) => acc + (status?.stats.get(feed.id)?.starred ?? 0),
              0,
            ),
            unread: folder.feeds.reduce(
              (acc, feed) => acc + (status?.stats.get(feed.id)?.unread ?? 0),
              0,
            ),
          },
        ]),
      ),
    [foldersWithFeeds, status],
  )
  const total = useMemo(
    () =>
      filter !== 'Feeds' &&
      (status?.stats
        .values()
        .reduce(
          (acc, stats) => acc + (filter === 'Unread' ? stats.unread : stats.starred),
          0,
        )
        .toString() ??
        ''),
    [status, filter],
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
                  ? (status?.stats.get(feed.id)?.unread ?? 0)
                  : (status?.stats.get(feed.id)?.starred ?? 0)) > 0,
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
              ? (status?.stats.get(feed.id)?.unread ?? 0)
              : (status?.stats.get(feed.id)?.starred ?? 0)) > 0,
        )

  return (
    <div style={panelStyle}>
      <div
        style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
      >
        <Wind style={{ marginLeft: length(3), marginRight: length(3) }} {...iconProps} />
        <ButtonGroup style={{ marginTop: length(1), marginBottom: length(1) }} outlined>
          {[
            { value: 'Unread', title: 'Unread', icon: <Circle {...iconProps} /> },
            { value: 'Feeds', title: 'All', icon: <MenuIcon {...iconProps} /> },
            { value: 'Starred', title: 'Starred', icon: <Star {...iconProps} /> },
          ].map(option => (
            <Button
              key={option.value}
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
                title={
                  status?.last_refreshed
                    ? `Last Refreshed: ${new Date(status.last_refreshed).toLocaleString()}`
                    : undefined
                }
                icon={<RotateCw {...menuIconProps} />}
                disabled={!!status?.running}
                onClick={async () => {
                  await xfetch('api/feeds/refresh', { method: 'POST' })
                  await refreshStats()
                }}
              />
              <MenuItem
                text="Change Refresh Rate"
                icon={<Edit {...menuIconProps} />}
                onClick={() => setChangeRefreshRateDialogOpen(true)}
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
                    menuRef.current?.parentElement?.click()
                    await Promise.all([refreshFeeds(), refreshStats()])
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
            style={{ marginRight: length(1) }}
            icon={<MoreHorizontal {...iconProps} />}
            minimal
          />
        </Popover>
      </div>
      <Divider />
      <Tree
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
                        disabled={!!status?.running}
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
                    <span
                      className={cn(
                        ctxMenuProps.className,
                        ctxMenuProps.contentProps.isOpen && 'context-menu-open',
                      )}
                      style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}
                      title={folder.title}
                      onContextMenu={ctxMenuProps.onContextMenu}
                      ref={ctxMenuProps.ref}
                    >
                      {ctxMenuProps.popover}
                      {folder.title}
                    </span>
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
      {status?.running ? (
        <>
          <Divider />
          <div style={statusBarStyle}>
            <Spinner
              style={{ marginLeft: length(3), marginRight: length(2) }}
              size={15}
            />
            Refreshing ({status.running} left)
          </div>
        </>
      ) : errors?.size ? (
        <>
          <Divider />
          <div style={statusBarStyle}>
            <AlertCircle
              style={{ marginLeft: length(3), marginRight: length(2) }}
              {...iconProps}
            />
            {errors.size} feeds have errors
          </div>
        </>
      ) : (
        <></>
      )}
      <Dialog
        isOpen={newFeedDialogOpen}
        close={() => setNewFeedDialogOpen(false)}
        title="New Feed"
        extraAction={
          <Button
            text={showGenerator ? 'Hide generator' : 'Show generator'}
            onClick={() => setShowGenerator(showGenerator => !showGenerator)}
          />
        }
        callback={async () => {
          if (!newFeedLink) throw new Error('Feed link is required')
          if (!selectedFolderRef.current) return
          setCreatingNewFeed(true)
          try {
            const feed = await xfetch<Feed>('api/feeds', {
              method: 'POST',
              body: JSON.stringify({
                url: newFeedLink,
                folder_id: selectedFolderRef.current.value
                  ? Number.parseInt(selectedFolderRef.current.value)
                  : null,
              }),
            })
            await Promise.all([refreshFeeds(), refreshStats(false)])
            setSelected(`feed:${feed.id}`)
            setNewFeedLink('')
          } finally {
            setCreatingNewFeed(false)
          }
        }}
      >
        <div style={{ display: 'flex' }}>
          <TextArea
            placeholder="https://example.com/feed"
            value={newFeedLink}
            onChange={evt => setNewFeedLink(evt.target.value)}
            spellCheck={false}
            {...textAreaProps}
          />
          <HTMLSelect
            style={{ marginLeft: length(2) }}
            iconName="caret-down"
            options={[
              { value: '', label: '--' },
              ...(folders ?? []).map(folder => ({
                value: folder.id,
                label: folder.title,
              })),
            ]}
            defaultValue={defaultSelectedFolder}
            ref={selectedFolderRef}
          />
        </div>
        <Collapse isOpen={showGenerator} keepChildrenMounted>
          <Divider style={{ marginTop: length(5), marginBottom: length(5) }} />
          <div style={{ textAlign: 'center' }}>
            <RadioGroup
              onChange={evt => setGeneratorType(evt.currentTarget.value)}
              selectedValue={generatorType}
              options={[
                { value: 'html', label: 'HTML Transformer' },
                { value: 'json', label: 'JSON Transformer' },
              ]}
              inline
            />
          </div>
          {[
            {
              ref: transUrl,
              key: 'url',
              desc: undefined,
              placeholder: 'https://example.com',
            },
            ...(generatorType === 'html' ? transHtmlParams : transJsonParams).current,
          ].map(({ ref, key, desc, placeholder }) => (
            <FormGroup
              key={`${generatorType}_${key}`}
              label={<Code>{key}</Code>}
              labelFor={`${generatorType}_${key}`}
              labelInfo={<span style={{ fontSize: '0.9em' }}>{desc}</span>}
              fill
            >
              <InputGroup
                id={`${generatorType}_${key}`}
                placeholder={placeholder}
                inputRef={ref}
                onValueChange={() => setNewFeedLink(`${generatorType}:${transParams()}`)}
                round
              />
            </FormGroup>
          ))}
          <AnchorButton
            text="Preview"
            href={`api/transform/${generatorType}/${encodeURIComponent(transParams())}`}
            target="_blank"
            intent={Intent.PRIMARY}
            rightIcon={<ExternalLink {...iconProps} />}
            outlined
            fill
          />
        </Collapse>
      </Dialog>
      <Dialog
        isOpen={newFolderDialogOpen}
        close={() => setNewFolderDialogOpen(false)}
        title="New Folder"
        callback={async () => {
          const title = newFolderTitleRef.current?.value
          if (!title) throw new Error('Folder title is required')
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
        <TextArea inputRef={newFolderTitleRef} {...textAreaProps} />
      </Dialog>
      <Dialog
        isOpen={changeRefreshRateDialogOpen}
        close={() => setChangeRefreshRateDialogOpen(false)}
        title="Change Auto Refresh Rate (min)"
        callback={async () => {
          if (!refreshRateRef.current) return
          const refreshRate = Number.parseInt(refreshRateRef.current.value)
          await xfetch('api/settings', {
            method: 'PUT',
            body: JSON.stringify({ refresh_rate: refreshRate }),
          })
          setSettings(settings => settings && { ...settings, refresh_rate: refreshRate })
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
      </Dialog>
      <Dialog
        isOpen={renameFeed}
        close={() => setRenameFeed(undefined)}
        title="Rename Feed"
        callback={async () => {
          if (!renameFeed) return
          const title = feedTitleRef.current?.value
          if (!title) throw new Error('Feed name is required')
          if (title !== renameFeed.title)
            await updateFeedAttr(renameFeed.id, 'title', title)
        }}
      >
        <TextArea
          defaultValue={renameFeed?.title}
          inputRef={feedTitleRef}
          {...textAreaProps}
        />
      </Dialog>
      <Dialog
        isOpen={changeLink}
        close={() => setChangeLink(undefined)}
        title="Change Feed Link"
        callback={async () => {
          if (!changeLink) return
          const feedLink = feedLinkRef.current?.value
          if (!feedLink) throw new Error('Feed link is required')
          if (feedLink !== changeLink.feed_link)
            await updateFeedAttr(changeLink.id, 'feed_link', feedLink)
        }}
      >
        <TextArea
          defaultValue={changeLink?.feed_link}
          inputRef={feedLinkRef}
          spellCheck={false}
          {...textAreaProps}
        />
      </Dialog>
      <Dialog
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
      </Dialog>
      <Dialog
        isOpen={renameFolder}
        close={() => setRenameFolder(undefined)}
        title="Rename Folder"
        callback={async () => {
          if (!renameFolder) return
          const title = folderTitleRef.current?.value
          if (!title) throw new Error('Folder title is required')
          if (title !== renameFolder.title) {
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
        <TextArea
          defaultValue={renameFolder?.title}
          inputRef={folderTitleRef}
          {...textAreaProps}
        />
      </Dialog>
      <Dialog
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
      </Dialog>
    </div>
  )
}
