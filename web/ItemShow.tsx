import { Dispatch, RefObject, SetStateAction } from 'react';
import {
  AnchorButton,
  Button,
  ButtonGroup,
  Classes,
  Colors,
  Divider,
  H2,
} from '@blueprintjs/core';
import { Star, Circle, ExternalLink } from 'react-feather';
import { Item, Image, Stats, Feed } from './types';
import { cn, iconProps, xfetch } from './utils';
import classes from './styles.module.css';

export default function ItemShow({
  setStats,
  setItems,
  selectedItemDetails,
  setSelectedItemDetails,
  contentRef,
  feedsById,
}: {
  setStats: Dispatch<SetStateAction<Map<number, Stats> | undefined>>;
  setItems: Dispatch<SetStateAction<(Item & Image)[] | undefined>>;
  selectedItemDetails: Item;
  setSelectedItemDetails: Dispatch<SetStateAction<Item | undefined>>;
  contentRef: RefObject<HTMLDivElement>;
  feedsById: Map<number, Feed>;
}) {
  const toggleStatus = (status: string) => async () => {
    status = status === selectedItemDetails.status ? 'read' : status;
    await xfetch(`./api/items/${selectedItemDetails.id}`, {
      method: 'PUT',
      body: { status },
    });
    const diff = (s: string) =>
      status === s ? +1 : selectedItemDetails.status === s ? -1 : 0;
    setStats(
      stats =>
        stats &&
        new Map(
          [...stats].map(([feedId, stats]) => [
            feedId,
            feedId === selectedItemDetails.feed_id
              ? {
                  starred: stats.starred + diff('starred'),
                  unread: stats.unread + diff('unread'),
                }
              : stats,
          ]),
        ),
    );
    setItems(items =>
      items?.map(i => (i.id === selectedItemDetails.id ? { ...i, status } : i)),
    );
    setSelectedItemDetails({ ...selectedItemDetails, status });
  };

  return (
    <div className="flex flex-col min-h-screen max-h-screen">
      <div className="flex flex-row min-h-10 max-h-10">
        <ButtonGroup className="ml-1 my-1 items-center" minimal>
          <Button
            icon={
              <Star
                {...iconProps}
                fill={
                  selectedItemDetails.status === 'starred'
                    ? Colors.DARK_GRAY1
                    : Colors.WHITE
                }
              />
            }
            onClick={toggleStatus('starred')}
            title="Mark Starred"
          />
          <Button
            icon={
              <Circle
                {...iconProps}
                fill={
                  selectedItemDetails.status === 'unread'
                    ? Colors.DARK_GRAY1
                    : Colors.WHITE
                }
              />
            }
            onClick={toggleStatus('unread')}
            title="Mark Unread"
          />
          <AnchorButton
            icon={<ExternalLink {...iconProps} color={Colors.BLUE3} />}
            href={selectedItemDetails.link}
            target="_blank"
            title="Open Link"
          />
        </ButtonGroup>
      </div>
      <Divider className="m-0" />
      <div className="overflow-auto p-5 break-words" ref={contentRef}>
        <H2 className="font-bold">{selectedItemDetails.title || 'untitled'}</H2>
        <div className="opacity-95">
          {feedsById.get(selectedItemDetails.feed_id)?.title}
        </div>
        <div className="opacity-95">
          {new Date(selectedItemDetails.date).toLocaleString()}
        </div>
        <Divider className="mx-0 my-3" />
        <div
          className={cn(Classes.RUNNING_TEXT, classes.content, 'text-base')}
          dangerouslySetInnerHTML={{ __html: selectedItemDetails.content }}
        ></div>
      </div>
    </div>
  );
}
