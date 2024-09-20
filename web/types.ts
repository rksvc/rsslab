export type Stats = {
  unread: number
  starred: number
}

export type Status = {
  stats: ({ feed_id: number } & Stats)[]
  running: number
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
  description?: string
  link?: string
  feed_link: string
  last_refreshed?: string
  has_icon: boolean
}

export type FolderWithFeeds = Folder & {
  feeds: Feed[]
}

export type Image = {
  loaded?: boolean
}

export type Item = {
  id: number
  guid: string
  feed_id: number
  title: string
  link: string
  content: string
  date: string
  status: string
  image?: string
  podcast_url?: string
}

export type Items = {
  list: Item[]
  has_more: boolean
}

export type Settings = {
  refresh_rate: number
  rsshub_path: string
}
