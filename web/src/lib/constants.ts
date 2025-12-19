// TMDB image base URL
export const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p'

export const POSTER_SIZES = {
  w92: `${TMDB_IMAGE_BASE}/w92`,
  w154: `${TMDB_IMAGE_BASE}/w154`,
  w185: `${TMDB_IMAGE_BASE}/w185`,
  w342: `${TMDB_IMAGE_BASE}/w342`,
  w500: `${TMDB_IMAGE_BASE}/w500`,
  w780: `${TMDB_IMAGE_BASE}/w780`,
  original: `${TMDB_IMAGE_BASE}/original`,
}

export const BACKDROP_SIZES = {
  w300: `${TMDB_IMAGE_BASE}/w300`,
  w780: `${TMDB_IMAGE_BASE}/w780`,
  w1280: `${TMDB_IMAGE_BASE}/w1280`,
  original: `${TMDB_IMAGE_BASE}/original`,
}

// Status colors
export const STATUS_COLORS = {
  available: 'text-green-500',
  downloading: 'text-yellow-500',
  missing: 'text-red-500',
  continuing: 'text-green-500',
  ended: 'text-muted-foreground',
  upcoming: 'text-blue-500',
} as const

// Quality resolution labels
export const RESOLUTION_LABELS: Record<number, string> = {
  480: 'SD',
  720: 'HD',
  1080: 'Full HD',
  2160: '4K',
}
