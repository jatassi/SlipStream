// TMDB image base URL (used for search results before download)
const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p'

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

// Local artwork API base URL
const ARTWORK_API_BASE = '/api/v1/metadata/artwork'

// Build local artwork URL
export function getLocalArtworkUrl(
  type: 'movie' | 'series',
  tmdbId: number,
  artworkType: 'poster' | 'backdrop' | 'logo' | 'studio_logo',
): string {
  return `${ARTWORK_API_BASE}/${type}/${tmdbId}/${artworkType}`
}

