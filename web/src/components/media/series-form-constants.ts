import type { SeriesMonitorOnAdd, SeriesSearchOnAdd } from '@/types'

export const MONITOR_LABELS: Record<SeriesMonitorOnAdd, string> = {
  all: 'All Episodes',
  future: 'Future Episodes Only',
  first_season: 'First Season Only',
  latest_season: 'Latest Season Only',
  none: 'None',
}

export const SEARCH_ON_ADD_LABELS: Record<SeriesSearchOnAdd, string> = {
  no: "Don't Search",
  first_episode: 'First Episode Only',
  first_season: 'First Season Only',
  latest_season: 'Latest Season Only',
  all: 'All Monitored Episodes',
}
