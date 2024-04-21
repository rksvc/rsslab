import { Dispatch, SetStateAction, useMemo, useRef, useState } from 'react';
import {
  type TreeNodeInfo,
  Tree,
  SegmentedControl,
  Divider,
  Intent,
  Icon,
  Button,
  Menu,
  Popover,
  InputGroup,
  MenuItem,
  HTMLSelect,
  MenuDivider,
  Slider,
  FileInput,
} from '@blueprintjs/core';
import {
  AlertCircle,
  Download,
  MoreHorizontal,
  Plus,
  RotateCw,
  Upload,
  Wind,
} from 'react-feather';
import { type Feed, type Folder, type FolderWithFeeds, Stats, Settings } from './types';
import { cn, iconProps, menuIconProps, popoverProps, xfetch } from './utils';
import classes from './styles.module.css';

export default function FeedList({
  filter,
  setFilter,
  folders,
  setFolders,
  stats,
  errors,
  selectedFeed,
  setSelectedFeed,
  loadingFeeds,
  settings,
  setSettings,

  refreshFeeds,
  refreshStats,
  foldersWithFeeds,
  feedsWithoutFolders,
}: {
  filter: string;
  setFilter: Dispatch<SetStateAction<string>>;
  folders?: Folder[];
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>;
  stats?: Map<number, Stats>;
  errors?: Map<number, string>;
  selectedFeed: string;
  setSelectedFeed: Dispatch<SetStateAction<string>>;
  loadingFeeds: number;
  settings?: Settings;
  setSettings: Dispatch<SetStateAction<Settings | undefined>>;

  refreshFeeds: () => Promise<void>;
  refreshStats: () => Promise<void>;
  foldersWithFeeds?: FolderWithFeeds[];
  feedsWithoutFolders?: Feed[];
}) {
  const [creatingNewFeed, setCreatingNewFeed] = useState(false);
  const [creatingNewFolder, setCreatingNewFolder] = useState(false);
  const menuRef = useRef<HTMLButtonElement>(null);
  const newFeedLinkRef = useRef<HTMLInputElement>(null);
  const folderSelectRef = useRef<HTMLSelectElement>(null);
  const newFolderTitleRef = useRef<HTMLInputElement>(null);
  const opmlFormRef = useRef<HTMLFormElement>(null);

  const newFeed = async () => {
    const url = newFeedLinkRef.current?.value;
    if (!url || !folderSelectRef.current) return;
    setCreatingNewFeed(true);
    try {
      const feed = await xfetch<Feed>(`./api/feeds`, {
        method: 'POST',
        body: {
          url,
          folder_id: parseInt(folderSelectRef.current.value) || null,
        },
      });
      await refreshFeeds();
      setSelectedFeed(`feed:${feed.id}`);
    } finally {
      setCreatingNewFeed(false);
    }
  };
  const newFolder = async () => {
    const title = newFolderTitleRef.current?.value;
    if (!title) return;
    setCreatingNewFolder(true);
    try {
      const folder = await xfetch<Folder>(`./api/folders`, {
        method: 'POST',
        body: { title },
      });
      setFolders(
        folders =>
          folders &&
          [...folders, folder].toSorted((a, b) =>
            a.title.toLocaleLowerCase().localeCompare(b.title.toLocaleLowerCase()),
          ),
      );
      setSelectedFeed(`folder:${folder.id}`);
    } finally {
      setCreatingNewFolder(false);
    }
  };
  const secondaryLabel = (stats?: Stats, error?: boolean) =>
    filter === 'Unread' ? (
      `${stats?.unread ?? ''}`
    ) : filter === 'Starred' ? (
      `${stats?.starred ?? ''}`
    ) : error ? (
      <Icon icon={<AlertCircle {...iconProps} />} />
    ) : (
      ''
    );
  const feed = (feed: Feed) => ({
    id: `feed:${feed.id}`,
    label: feed.title,
    icon: feed.has_icon ? (
      <img
        className="w-4"
        style={{ marginRight: '7px' }}
        src={`./api/feeds/${feed.id}/icon`}
      ></img>
    ) : (
      ('feed' as const)
    ),
    isSelected: selectedFeed === `feed:${feed.id}`,
    secondaryLabel: secondaryLabel(stats?.get(feed.id), !!errors?.get(feed.id)),
  });
  const setExpanded = (isExpanded: boolean) => async (node: TreeNodeInfo) => {
    const id = parseInt(`${node.id}`.split(':')[1]);
    setFolders(folders =>
      folders?.map(folder =>
        folder.id === id ? { ...folder, is_expanded: isExpanded } : folder,
      ),
    );
    await xfetch(`./api/folders/${id}`, {
      method: 'PUT',
      body: { is_expanded: isExpanded },
    });
  };

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
  );
  const total = useMemo(
    () =>
      filter !== 'Feeds' &&
      `${[...(stats?.values() ?? [])].reduce(
        (acc, stats) => acc + (filter === 'Unread' ? stats.unread : stats.starred),
        0,
      )}`,
    [stats, filter],
  );
  const visibleFolders =
    filter === 'Feeds'
      ? foldersWithFeeds
      : foldersWithFeeds
          ?.map(folder => ({
            ...folder,
            feeds: folder.feeds.filter(
              feed =>
                selectedFeed === `feed:${feed.id}` ||
                (filter === 'Unread'
                  ? stats?.get(feed.id)?.unread ?? 0
                  : stats?.get(feed.id)?.starred ?? 0) > 0,
            ),
          }))
          .filter(
            folder => folder.feeds.length || selectedFeed === `folder:${folder.id}`,
          );
  const visibleFeeds =
    filter === 'Feeds'
      ? feedsWithoutFolders
      : feedsWithoutFolders?.filter(
          feed =>
            selectedFeed === `feed:${feed.id}` ||
            (filter === 'Unread'
              ? stats?.get(feed.id)?.unread ?? 0
              : stats?.get(feed.id)?.starred ?? 0) > 0,
        );

  return (
    <div className="flex flex-col min-h-screen max-h-screen">
      <div className="flex flex-row justify-between items-center">
        <Wind className="ml-3 mr-3" {...iconProps} />
        <SegmentedControl
          className="min-h-10 max-h-10 bg-white"
          intent={Intent.PRIMARY}
          value={filter}
          onValueChange={setFilter}
          options={[
            { label: 'Unread', value: 'Unread' },
            { label: 'All', value: 'Feeds' },
            { label: 'Starred', value: 'Starred' },
          ]}
        />
        <Popover
          placement="bottom"
          {...popoverProps}
          content={
            <Menu>
              <MenuItem
                text="New Feed"
                disabled={creatingNewFeed}
                icon={<Plus {...menuIconProps} />}
                spellCheck={false}
                onClick={newFeed}
                labelElement={
                  <div className="flex flex-col">
                    <InputGroup
                      className="mb-1 ml-1"
                      placeholder="https://example.com/feed"
                      onClick={event => event.stopPropagation()}
                      inputRef={newFeedLinkRef}
                      small
                      onKeyDown={event => {
                        event.key === ' ' && event.stopPropagation();
                        event.key === 'Enter' && newFeed();
                      }}
                    />
                    <HTMLSelect
                      ref={folderSelectRef}
                      className="ml-1"
                      onClick={event => event.stopPropagation()}
                      iconName="caret-down"
                      options={[
                        { value: '', label: '--' },
                        ...(folders ?? []).map(folder => ({
                          value: folder.id,
                          label: folder.title,
                        })),
                      ]}
                    />
                  </div>
                }
              />
              <MenuItem
                text="New Folder"
                disabled={creatingNewFolder}
                icon={<Plus {...menuIconProps} />}
                onClick={newFolder}
                labelElement={
                  <InputGroup
                    className="ml-1"
                    onClick={event => event.stopPropagation()}
                    inputRef={newFolderTitleRef}
                    small
                    onKeyDown={event => {
                      event.key === ' ' && event.stopPropagation();
                      event.key === 'Enter' && newFolder();
                    }}
                  />
                }
              />
              <MenuDivider />
              {errors && errors.size > 0 && (
                <MenuItem
                  text={`Resolve Errors (${errors.size})`}
                  icon={<RotateCw {...menuIconProps} />}
                  disabled={loadingFeeds > 0}
                  onClick={async () => {
                    await xfetch('./api/feeds/errors/refresh', { method: 'POST' });
                    await refreshStats();
                  }}
                />
              )}
              <MenuItem
                text="Refresh Feeds"
                icon={<RotateCw {...menuIconProps} />}
                disabled={loadingFeeds > 0}
                onClick={async () => {
                  await xfetch('./api/feeds/refresh', { method: 'POST' });
                  await refreshStats();
                }}
              />
              <MenuDivider title="Auto Refresh (min)" className="select-none" />
              <Slider
                className={cn('mt-3', 'ml-auto', 'mr-auto', classes.slider)}
                min={0}
                max={240}
                stepSize={30}
                labelStepSize={30}
                value={settings?.refresh_rate}
                onChange={value =>
                  setSettings(
                    settings => settings && { ...settings, refresh_rate: value },
                  )
                }
                onRelease={() =>
                  xfetch('./api/settings', {
                    method: 'PUT',
                    body: { refresh_rate: settings?.refresh_rate },
                  })
                }
              />
              <MenuDivider title="Subscriptions" className="select-none" />
              <form ref={opmlFormRef}>
                <FileInput
                  className="hidden"
                  inputProps={{ name: 'opml', id: 'opml-import' }}
                  onInputChange={async () => {
                    if (!opmlFormRef.current) return;
                    await xfetch('./api/opml/import', {
                      method: 'POST',
                      body: new FormData(opmlFormRef.current),
                    });
                    menuRef.current?.parentElement?.click();
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
                href="./api/opml/export"
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
            isSelected: !selectedFeed,
            secondaryLabel: total,
          },
          ...(visibleFeeds ?? []).map(feed),
          ...(visibleFolders ?? []).map(folder => ({
            id: `folder:${folder.id}`,
            label: folder.title,
            isExpanded: folder.is_expanded,
            isSelected: selectedFeed === `folder:${folder.id}`,
            icon: 'folder-close' as const,
            childNodes: folder.feeds.map(feed),
            secondaryLabel: secondaryLabel(folderStats.get(folder.id)),
          })),
        ]}
        onNodeExpand={setExpanded(true)}
        onNodeCollapse={setExpanded(false)}
        onNodeClick={node => setSelectedFeed(typeof node.id === 'number' ? '' : node.id)}
      />
      {loadingFeeds > 0 && (
        <>
          <Divider className="m-0" />
          <div className="flex items-center p-1 break-words">
            <Icon icon={<RotateCw className="animate-spin ml-3 mr-2" {...iconProps} />} />
            {`Refreshing (${loadingFeeds} left)`}
          </div>
        </>
      )}
    </div>
  );
}
