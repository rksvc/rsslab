import { Dispatch, SetStateAction, useEffect, useMemo, useRef, useState } from 'react';
import {
  type TreeNodeInfo,
  Tree,
  Divider,
  Intent,
  Button,
  Menu,
  Popover,
  InputGroup,
  MenuItem,
  HTMLSelect,
  MenuDivider,
  FileInput,
  NumericInput,
  ButtonGroup,
  Spinner,
} from '@blueprintjs/core';
import {
  AlertCircle,
  Circle,
  Download,
  Edit,
  MoreHorizontal,
  Plus,
  RotateCw,
  Upload,
  Wind,
  Menu as MenuIcon,
  Star,
  Rss,
} from 'react-feather';
import { Confirm } from './Confirm';
import type { Feed, Folder, FolderWithFeeds, Stats, Settings } from './types';
import { iconProps, menuIconProps, popoverProps, xfetch } from './utils';

export default function FeedList({
  filter,
  setFilter,
  folders,
  setFolders,
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
  filter: string;
  setFilter: Dispatch<SetStateAction<string>>;
  folders?: Folder[];
  setFolders: Dispatch<SetStateAction<Folder[] | undefined>>;
  stats?: Map<number, Stats>;
  errors?: Map<number, string>;
  selected: string;
  setSelected: Dispatch<SetStateAction<string>>;
  loadingFeeds: number;
  settings?: Settings;
  setSettings: Dispatch<SetStateAction<Settings | undefined>>;

  refreshFeeds: () => Promise<void>;
  refreshStats: (loop?: boolean) => Promise<void>;
  foldersWithFeeds?: FolderWithFeeds[];
  feedsWithoutFolders?: Feed[];
  feedsById: Map<number, Feed>;
}) {
  const [selectedFolder, setSelectedFolder] = useState(
    getSelectedFolder(selected, feedsById),
  );
  useEffect(
    () => setSelectedFolder(getSelectedFolder(selected, feedsById)),
    [selected, feedsById],
  );

  const [creatingNewFeed, setCreatingNewFeed] = useState(false);
  const [creatingNewFolder, setCreatingNewFolder] = useState(false);
  const [newFeedDialogOpen, setNewFeedDialogOpen] = useState(false);
  const [newFolderDialogOpen, setNewFolderDialogOpen] = useState(false);
  const [changeRefreshRateDialogOpen, setChangeRefreshRateDialogOpen] = useState(false);
  const menuRef = useRef<HTMLButtonElement>(null);
  const newFeedLinkRef = useRef<HTMLInputElement>(null);
  const newFolderTitleRef = useRef<HTMLInputElement>(null);
  const refreshRateRef = useRef<HTMLInputElement>(null);
  const opmlFormRef = useRef<HTMLFormElement>(null);

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
    );
  const feed = (feed: Feed) => ({
    id: `feed:${feed.id}`,
    label: <span title={feed.title}>{feed.title}</span>,
    icon: feed.has_icon ? (
      <img className="w-4 mr-[7px]" src={`./api/feeds/${feed.id}/icon`}></img>
    ) : (
      <Rss className="mr-[6px]" {...iconProps} />
    ),
    isSelected: selected === `feed:${feed.id}`,
    secondaryLabel: secondaryLabel(
      stats?.get(feed.id),
      !!errors?.get(feed.id),
      feed.last_refreshed,
    ),
  });
  const setExpanded = (isExpanded: boolean) => async (node: TreeNodeInfo) => {
    const id = Number.parseInt(`${node.id}`.split(':')[1]);
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
                selected === `feed:${feed.id}` ||
                (filter === 'Unread'
                  ? stats?.get(feed.id)?.unread ?? 0
                  : stats?.get(feed.id)?.starred ?? 0) > 0,
            ),
          }))
          .filter(
            folder => folder.feeds.length > 0 || selected === `folder:${folder.id}`,
          );
  const visibleFeeds =
    filter === 'Feeds'
      ? feedsWithoutFolders
      : feedsWithoutFolders?.filter(
          feed =>
            selected === `feed:${feed.id}` ||
            (filter === 'Unread'
              ? stats?.get(feed.id)?.unread ?? 0
              : stats?.get(feed.id)?.starred ?? 0) > 0,
        );

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
            isSelected: !selected,
            secondaryLabel: total,
          },
          ...(visibleFeeds ?? []).map(f => feed(f)),
          ...(visibleFolders ?? []).map(folder => ({
            id: `folder:${folder.id}`,
            label: <span title={folder.title}>{folder.title}</span>,
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
      {loadingFeeds > 0 && (
        <>
          <Divider className="m-0" />
          <div className="flex items-center p-1 break-words">
            <Spinner className="ml-3 mr-2" size={15} />
            {`Refreshing (${loadingFeeds} left)`}
          </div>
        </>
      )}
      <Confirm
        open={newFeedDialogOpen}
        setOpen={setNewFeedDialogOpen}
        title="New Feed"
        children={
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
        }
        callback={async () => {
          const url = newFeedLinkRef.current?.value;
          if (!url) return;
          setCreatingNewFeed(true);
          try {
            const feed = await xfetch<Feed>(`./api/feeds`, {
              method: 'POST',
              body: {
                url,
                folder_id: selectedFolder ? Number.parseInt(selectedFolder) : undefined,
              },
            });
            await Promise.all([refreshFeeds(), refreshStats(false)]);
            setSelected(`feed:${feed.id}`);
          } finally {
            setCreatingNewFeed(false);
          }
        }}
      />
      <Confirm
        open={newFolderDialogOpen}
        setOpen={setNewFolderDialogOpen}
        title="New Folder"
        children={<InputGroup className="ml-1" inputRef={newFolderTitleRef} />}
        callback={async () => {
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
            setSelected(`folder:${folder.id}`);
          } finally {
            setCreatingNewFolder(false);
          }
        }}
      />
      <Confirm
        open={changeRefreshRateDialogOpen}
        setOpen={setChangeRefreshRateDialogOpen}
        title="Change Auto Refresh Rate (min)"
        children={
          <NumericInput
            defaultValue={settings?.refresh_rate}
            inputRef={refreshRateRef}
            min={0}
            stepSize={30}
            minorStepSize={1}
            majorStepSize={60}
            fill
          />
        }
        callback={async () => {
          if (!refreshRateRef.current) return;
          const refreshRate = Number.parseInt(refreshRateRef.current.value);
          setSettings(settings => settings && { ...settings, refresh_rate: refreshRate });
          await xfetch('./api/settings', {
            method: 'PUT',
            body: { refresh_rate: refreshRate },
          });
        }}
      />
    </div>
  );
}

function getSelectedFolder(selected: string, feedsById: Map<number, Feed>): string {
  const [type, id] = selected.split(':');
  return type === 'feed'
    ? feedsById.get(Number.parseInt(id))?.folder_id?.toString() ?? ''
    : type === 'folder'
    ? id
    : '';
}
