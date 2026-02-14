export type CalendarEvent = {
  id: number
  title: string
  mediaType: 'movie' | 'episode'
  eventType: 'digital' | 'physical' | 'airDate' // digital = streaming/VOD release, physical = Bluray
  date: string // YYYY-MM-DD
  status: 'missing' | 'available' | 'downloading'
  monitored: boolean

  // Movie-specific
  tmdbId?: number
  year?: number

  // Episode-specific
  seriesId?: number
  seriesTitle?: string
  seasonNumber?: number
  episodeNumber?: number
  network?: string

  // Streaming services with early release (Apple TV+)
  earlyAccess?: boolean
}

export type CalendarRequest = {
  start: string // YYYY-MM-DD
  end: string // YYYY-MM-DD
}

export type CalendarView = 'month' | 'week' | 'agenda'
