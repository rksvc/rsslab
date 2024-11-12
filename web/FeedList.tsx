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
import type {
  Feed,
  FeedState,
  Folder,
  FolderWithFeeds,
  Selected,
  Settings,
  Status,
} from './types'
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

type Param = {
  value: string
  setValue: Dispatch<SetStateAction<string>>
  key: string
  desc: string | JSX.Element
  parse?: (input: string) => any
  placeholder?: string
}

export default function FeedList({
  filter,
  setFilter,
  folders,
  setFolders,
  setFeeds,
  status,
  selected,
  setSelected,
  settings,
  setSettings,

  refreshFeeds,
  refreshStats,
  errorCount,
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
  selected: Selected
  setSelected: Dispatch<SetStateAction<Selected>>
  settings?: Settings
  setSettings: Dispatch<SetStateAction<Settings | undefined>>

  refreshFeeds: () => Promise<void>
  refreshStats: (loop?: boolean) => Promise<void>
  errorCount?: number
  foldersWithFeeds?: FolderWithFeeds[]
  feedsWithoutFolders?: Feed[]
  feedsById: Map<number, Feed>
}) {
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
  const [transType, setTransType] = useState('html')
  const [transUrl, setTransUrl] = useState('')
  const [transHtmlTitle, setTransHtmlTitle] = useState('')
  const [transHtmlItems, setTransHtmlItems] = useState('')
  const [transHtmlItemTitle, setTransHtmlItemTitle] = useState('')
  const [transHtmlItemUrl, setTransHtmlItemUrl] = useState('')
  const [transHtmlItemUrlAttr, setTransHtmlItemUrlAttr] = useState('')
  const [transHtmlItemContent, setTransHtmlItemContent] = useState('')
  const [transHtmlItemDate, setTransHtmlItemDate] = useState('')
  const [transJsonHomePageUrl, setTransJsonHomePageUrl] = useState('')
  const [transJsonTitle, setTransJsonTitle] = useState('')
  const [transJsonHeaders, setTransJsonHeaders] = useState('')
  const [transJsonItems, setTransJsonItems] = useState('')
  const [transJsonItemTitle, setTransJsonItemTitle] = useState('')
  const [transJsonItemUrl, setTransJsonItemUrl] = useState('')
  const [transJsonItemUrlPrefix, setTransJsonItemUrlPrefix] = useState('')
  const [transJsonItemContent, setTransJsonItemContent] = useState('')
  const [transJsonItemDatePublished, setTransJsonItemDatePublished] = useState('')
  const transHtmlParams: Param[] = [
    {
      value: transHtmlTitle,
      setValue: setTransHtmlTitle,
      key: 'title',
      desc: 'Title of RSS',
      placeholder: 'extracted from <title>',
    },
    {
      value: transHtmlItems,
      setValue: setTransHtmlItems,
      key: 'items',
      desc: 'CSS selector targetting items',
      placeholder: 'html',
    },
    {
      value: transHtmlItemTitle,
      setValue: setTransHtmlItemTitle,
      key: 'item_title',
      desc: 'CSS selector targetting title of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemUrl,
      setValue: setTransHtmlItemUrl,
      key: 'item_url',
      desc: 'CSS selector targetting URL of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemUrlAttr,
      setValue: setTransHtmlItemUrlAttr,
      key: 'item_url_attr',
      desc: (
        <span>
          Attribute of <Code>item_url</Code> element as URL
        </span>
      ),
      placeholder: 'href',
    },
    {
      value: transHtmlItemContent,
      setValue: setTransHtmlItemContent,
      key: 'item_content',
      desc: 'CSS selector targetting content of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemDate,
      setValue: setTransHtmlItemDate,
      key: 'item_date_published',
      desc: 'CSS selector targetting publication date of item',
      placeholder: 'same as item element',
    },
  ]
  const transJsonParams: Param[] = [
    {
      value: transJsonHomePageUrl,
      setValue: setTransJsonHomePageUrl,
      key: 'home_page_url',
      desc: 'Home page URL of RSS',
    },
    {
      value: transJsonTitle,
      setValue: setTransJsonTitle,
      key: 'title',
      desc: 'Title of RSS',
    },
    {
      value: transJsonHeaders,
      setValue: setTransJsonHeaders,
      key: 'headers',
      desc: 'HTTP request headers in JSON form',
      parse: (input: string) => {
        try {
          return JSON.parse(input)
        } catch {
          return null
        }
      },
    },
    {
      value: transJsonItems,
      setValue: setTransJsonItems,
      key: 'items',
      desc: 'JSON path to items',
      placeholder: 'entire JSON response',
    },
    {
      value: transJsonItemTitle,
      setValue: setTransJsonItemTitle,
      key: 'item_title',
      desc: 'JSON path to title of item',
    },
    {
      value: transJsonItemUrl,
      setValue: setTransJsonItemUrl,
      key: 'item_url',
      desc: 'JSON path to URL of item',
    },
    {
      value: transJsonItemUrlPrefix,
      setValue: setTransJsonItemUrlPrefix,
      key: 'item_url_prefix',
      desc: 'Optional prefix for URL',
    },
    {
      value: transJsonItemContent,
      setValue: setTransJsonItemContent,
      key: 'item_content',
      desc: 'JSON path to content of item',
    },
    {
      value: transJsonItemDatePublished,
      setValue: setTransJsonItemDatePublished,
      key: 'item_date_published',
      desc: 'JSON path to publication date of item',
    },
  ]
  const transParams = JSON.stringify({
    url: transUrl,
    ...Object.fromEntries(
      (transType === 'html' ? transHtmlParams : transJsonParams)
        .map(({ key, value, parse }) => [key, parse ? parse(value) : value])
        .filter(([_, value]) => value),
    ),
  })

  const menuRef = useRef<HTMLButtonElement>(null)
  const [newFeedLink, setNewFeedLink] = useState('')
  const selectedFolderRef = useRef<HTMLSelectElement>(null)
  const newFolderTitleRef = useRef<HTMLTextAreaElement>(null)
  const refreshRateRef = useRef<HTMLInputElement>(null)
  const opmlFormRef = useRef<HTMLFormElement>(null)
  const feedTitleRef = useRef<HTMLTextAreaElement>(null)
  const feedLinkRef = useRef<HTMLTextAreaElement>(null)
  const folderTitleRef = useRef<HTMLTextAreaElement>(null)
  const [isTypingTransParams, setIsTypingTransParams] = useState(false)
  const autoNewFeedLink = isTypingTransParams
    ? `${transType}:${transParams}`
    : newFeedLink

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
  const secondaryLabel = (state?: FeedState, lastRefreshed?: string) =>
    filter === 'Unread' ? (
      `${state?.unread ?? ''}`
    ) : filter === 'Starred' ? (
      `${state?.starred ?? ''}`
    ) : state?.error ? (
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
  const feed = (feed: Feed) =>
    ({
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
      isSelected: selected?.feed_id === feed.id,
      secondaryLabel: secondaryLabel(status?.state.get(feed.id), feed.last_refreshed),
      nodeData: { feed_id: feed.id },
    }) satisfies TreeNodeInfo<Selected>
  const setExpanded = (isExpanded: boolean) => async (node: TreeNodeInfo<Selected>) => {
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
              (acc, feed) => acc + (status?.state.get(feed.id)?.starred ?? 0),
              0,
            ),
            unread: folder.feeds.reduce(
              (acc, feed) => acc + (status?.state.get(feed.id)?.unread ?? 0),
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
      (status?.state
        .values()
        .reduce(
          (acc, state) => acc + (filter === 'Unread' ? state.unread : state.starred),
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
                selected?.feed_id === feed.id ||
                (filter === 'Unread'
                  ? (status?.state.get(feed.id)?.unread ?? 0)
                  : (status?.state.get(feed.id)?.starred ?? 0)) > 0,
            ),
          }))
          .filter(folder => folder.feeds.length > 0 || selected?.folder_id === folder.id)
  const visibleFeeds =
    filter === 'Feeds'
      ? feedsWithoutFolders
      : feedsWithoutFolders?.filter(
          feed =>
            selected?.feed_id === feed.id ||
            (filter === 'Unread'
              ? (status?.state.get(feed.id)?.unread ?? 0)
              : (status?.state.get(feed.id)?.starred ?? 0)) > 0,
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
      <Tree<Selected>
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
            isSelected: selected?.folder_id === folder.id,
            childNodes: folder.feeds.map(f => feed(f)),
            secondaryLabel: secondaryLabel(folderStats.get(folder.id)),
            nodeData: { folder_id: folder.id },
          })),
        ]}
        onNodeExpand={setExpanded(true)}
        onNodeCollapse={setExpanded(false)}
        onNodeClick={node => setSelected(node.nodeData)}
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
      ) : errorCount ? (
        <>
          <Divider />
          <div style={statusBarStyle}>
            <AlertCircle
              style={{ marginLeft: length(3), marginRight: length(2) }}
              {...iconProps}
            />
            {errorCount} feeds have errors
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
          if (!autoNewFeedLink) throw new Error('Feed link is required')
          if (!selectedFolderRef.current) return
          setCreatingNewFeed(true)
          try {
            const feed = await xfetch<Feed>('api/feeds', {
              method: 'POST',
              body: JSON.stringify({
                url: autoNewFeedLink,
                folder_id: selectedFolderRef.current.value
                  ? Number.parseInt(selectedFolderRef.current.value)
                  : null,
              }),
            })
            await Promise.all([refreshFeeds(), refreshStats(false)])
            setSelected({ feed_id: feed.id })
            setNewFeedLink('')
          } finally {
            setCreatingNewFeed(false)
          }
        }}
      >
        <div style={{ display: 'flex' }}>
          <TextArea
            placeholder="https://example.com/feed"
            value={autoNewFeedLink}
            onChange={evt => {
              const feedLink = evt.target.value
              setNewFeedLink(feedLink)
              setIsTypingTransParams(false)
              const i = feedLink.indexOf(':')
              if (i === -1) return
              const scheme = feedLink.slice(0, i)
              switch (scheme) {
                case 'html':
                case 'json': {
                  const list = scheme === 'html' ? transHtmlParams : transJsonParams
                  try {
                    const params: Record<string, any> = JSON.parse(feedLink.slice(i + 1))
                    for (const key in params) {
                      const param = list.find(param => param.key === key)
                      if (param) {
                        const value = params[key]
                        param.setValue(
                          typeof value === 'string' ? value : JSON.stringify(value),
                        )
                      }
                    }
                    setTransUrl(params.url ?? '')
                    setTransType(scheme)
                  } catch {}
                }
              }
            }}
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
            defaultValue={
              selected
                ? (selected.folder_id ?? feedsById.get(selected.feed_id)?.folder_id ?? '')
                : ''
            }
            ref={selectedFolderRef}
          />
        </div>
        <Collapse isOpen={showGenerator} keepChildrenMounted>
          <Divider style={{ marginTop: length(5), marginBottom: length(5) }} />
          <div style={{ textAlign: 'center' }}>
            <RadioGroup
              onChange={evt => setTransType(evt.currentTarget.value)}
              selectedValue={transType}
              options={[
                { value: 'html', label: 'HTML Transformer' },
                { value: 'json', label: 'JSON Transformer' },
              ]}
              inline
            />
          </div>
          {[
            {
              value: transUrl,
              setValue: setTransUrl,
              key: 'url',
              desc: undefined,
              placeholder: 'https://example.com',
            },
            ...(transType === 'html' ? transHtmlParams : transJsonParams),
          ].map(({ value, setValue, key, desc, placeholder }) => (
            <FormGroup
              key={`${transType}_${key}`}
              label={<Code>{key}</Code>}
              labelFor={`${transType}_${key}`}
              labelInfo={<span style={{ fontSize: '0.9em' }}>{desc}</span>}
              fill
            >
              <InputGroup
                value={value}
                id={`${transType}_${key}`}
                placeholder={placeholder}
                onValueChange={value => {
                  setValue(value)
                  setIsTypingTransParams(true)
                }}
                round
              />
            </FormGroup>
          ))}
          <AnchorButton
            text="Preview"
            href={`api/transform/${transType}/${encodeURIComponent(transParams)}`}
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
            setSelected({ folder_id: folder.id })
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
            deleteFeed.folder_id === null
              ? undefined
              : { folder_id: deleteFeed.folder_id },
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
          setSelected(undefined)
        }}
        intent={Intent.DANGER}
      >
        Are you sure you want to delete {deleteFolder?.title || 'untitled'}?
      </Dialog>
    </div>
  )
}
