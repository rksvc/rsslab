import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Divider, FocusStyleManager } from '@blueprintjs/core';
import FeedList from './FeedList';
import ItemList from './ItemList';
import ItemShow from './ItemShow';
import {
  Item,
  Image,
  FolderWithFeeds,
  Stats,
  Feed,
  Folder,
  Status,
  Settings,
} from './types';
import { xfetch } from './utils';

FocusStyleManager.onlyShowFocusOnTabs();

export default function App() {
  const [filter, setFilter] = useState('Unread');
  const [folders, setFolders] = useState<Folder[]>();
  const [feeds, setFeeds] = useState<Feed[]>();
  const [stats, setStats] = useState<Map<number, Stats>>();
  const [errors, setErrors] = useState<Map<number, string>>();
  const [selectedFeed, setSelectedFeed] = useState<string>('');
  const [loadingFeeds, setLoadingFeeds] = useState(0);
  const [settings, setSettings] = useState<Settings>();

  const [items, setItems] = useState<(Item & Image)[]>();
  const [selectedItemId, setSelectedItemId] = useState<number>();

  const [selectedItemDetails, setSelectedItemDetails] = useState<Item>();
  const contentRef = useRef<HTMLDivElement>(null);

  const refreshFeeds = useCallback(async () => {
    const [folders, feeds, settings] = await Promise.all([
      xfetch<Folder[]>('./api/folders'),
      xfetch<Feed[]>('./api/feeds'),
      xfetch<Settings>('./api/settings'),
    ]);
    setFolders(folders);
    setFeeds(feeds);
    setSettings(settings);
  }, []);
  const refreshStats = useCallback(async (loop: boolean = true) => {
    const [errors, status] = await Promise.all([
      xfetch<Record<number, string>>('./api/feeds/errors'),
      xfetch<Status>('./api/status'),
    ]);
    setErrors(
      new Map(Object.entries(errors).map(([id, error]) => [parseInt(id), error])),
    );
    setStats(new Map(status.stats.map(stats => [stats.feed_id, stats])));
    if (loop) {
      setLoadingFeeds(status.running);
      if (status.running) setTimeout(() => refreshStats(), 500);
    }
  }, []);
  useEffect(() => {
    refreshFeeds();
    refreshStats();
  }, [refreshFeeds, refreshStats]);

  const [foldersWithFeeds, feedsWithoutFolders, foldersById, feedsById] = useMemo(() => {
    const foldersById = new Map<number, FolderWithFeeds>();
    for (const folder of folders ?? [])
      foldersById.set(folder.id, { ...folder, feeds: [] });
    const feedsById = new Map<number, Feed>();
    const feedsWithoutFolders: Feed[] = [];
    for (const feed of feeds ?? []) {
      if (feed.folder_id != null) foldersById.get(feed.folder_id)?.feeds.push(feed);
      else feedsWithoutFolders.push(feed);
      feedsById.set(feed.id, feed);
    }
    return [[...foldersById.values()], feedsWithoutFolders, foldersById, feedsById];
  }, [feeds, folders]);

  const props = {
    filter,
    setFilter,
    folders,
    setFolders,
    setFeeds,
    stats,
    setStats,
    errors,
    selectedFeed,
    setSelectedFeed,
    loadingFeeds,
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
    foldersById,
    feedsById,
  };

  return (
    <div className="flex flex-row text-base min-h-screen max-h-screen">
      <div style={{ minWidth: '300px', maxWidth: '300px' }}>
        <FeedList {...props} />
      </div>
      <Divider className="m-0" />
      <div style={{ minWidth: '300px', maxWidth: '300px' }}>
        <ItemList {...props} />
      </div>
      <Divider className="m-0" />
      <div className="grow min-w-0">
        {selectedItemDetails && (
          <ItemShow {...props} selectedItemDetails={selectedItemDetails} />
        )}
      </div>
    </div>
  );
}
