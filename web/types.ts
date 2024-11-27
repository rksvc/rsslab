export type FeedState = {
  unread: number
  starred: number
  last_refreshed?: string
  error?: string
}

export type Status = {
  running: number
  last_refreshed: string | null
  state: Map<number, FeedState>
}

export type Folder = {
  id: number
  title: string
  is_expanded: boolean
}

export type Feed = {
  id: number
  folder_id: number | null
  title: string
  link?: string
  feed_link: string
  has_icon: boolean
}

export type FolderWithFeeds = Folder & {
  feeds: Feed[]
}

export type Item = {
  id: number
  guid: string
  feed_id: number
  title: string
  link: string
  content?: string
  date: string
  status: 'unread' | 'read' | 'starred'
  image?: string
  podcast_url?: string
}

export type Items = {
  list: Item[]
  has_more: boolean
}

export type Settings = {
  refresh_rate?: number
  dark_theme?: boolean
}

export type Filter = 'Unread' | 'Feeds' | 'Starred'

// https://stackoverflow.com/a/53229567
type Without<T, U> = { [P in Exclude<keyof T, keyof U>]?: undefined }
type Xor<T, U> = T | U extends object ? (Without<T, U> & U) | (Without<U, T> & T) : T | U

export type Selected =
  | Xor<
      {
        feed_id: number
      },
      { folder_id: number }
    >
  | undefined

export type Transformer = 'html' | 'json'
