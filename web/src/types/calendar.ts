export type CalendarEvent = {
  id: number
  title: string
  mediaType: string
  moduleType?: string
  eventType: 'digital' | 'physical' | 'airDate' // digital = streaming/VOD release, physical = Bluray
  date: string // YYYY-MM-DD
  status: 'missing' | 'available' | 'downloading'
  monitored: boolean

  // Module-specific fields (tmdbId, year, seriesId, seriesTitle,
  // seasonNumber, episodeNumber, network, earlyAccess, etc.)
  extra?: Record<string, unknown>
}

export type CalendarRequest = {
  start: string // YYYY-MM-DD
  end: string // YYYY-MM-DD
}

export type CalendarView = 'month' | 'week' | 'agenda'
