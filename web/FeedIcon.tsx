import { Rss } from 'lucide-react'
import { useEffect } from 'react'

import { useMyContext } from './Context.tsx'
import type { Feed } from './types.ts'
import { iconSize, xfetch } from './utils.ts'

export default function FeedIcon({ feed }: { feed: Feed }) {
  const { setFeeds } = useMyContext()
  const src = `api/feeds/${feed.id}/icon`

  useEffect(() => {
    void (async () => {
      if (feed.has_icon == null) {
        const hasIcon = await xfetch<boolean>(`api/feeds/${feed.id}/has_icon`)
        setFeeds(feeds => feeds?.map(f => (f.id === feed.id ? { ...f, has_icon: hasIcon } : f)))
      }
    })()
  }, [feed, setFeeds])

  return feed.has_icon ? (
    <img
      style={{ marginRight: 8, width: iconSize, aspectRatio: '1/1' }}
      alt="feed icon"
      src={src}
    />
  ) : (
    <Rss style={{ marginRight: 8, display: 'flex' }} size={iconSize} />
  )
}
