import { useEffect } from 'react'
import { Rss } from 'react-feather'
import { useMyContext } from './Context.tsx'
import type { Feed } from './types.ts'
import { length, xfetch } from './utils.ts'

export default function FeedIcon({ feed }: { feed: Feed }) {
  const { setFeeds } = useMyContext()
  const src = `api/feeds/${feed.id}/icon`

  useEffect(() => {
    ;(async () => {
      if (feed.has_icon == null) {
        const hasIcon = await xfetch<boolean>(`api/feeds/${feed.id}/has_icon`)
        setFeeds(feeds => feeds?.map(f => (f.id === feed.id ? { ...f, has_icon: hasIcon } : f)))
      }
    })()
  }, [feed, setFeeds])

  return feed.has_icon ? (
    <img alt="feed icon" style={{ width: length(4), aspectRatio: '1/1', marginRight: '7px' }} src={src} />
  ) : (
    <span style={{ display: 'flex' }}>
      <Rss style={{ marginRight: '6px' }} />
    </span>
  )
}
