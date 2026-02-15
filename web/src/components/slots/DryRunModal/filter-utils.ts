import type {
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
} from './types'

export type FilterType = 'all' | 'assigned' | 'conflicts' | 'nomatch'

type Episode = TVShowMigrationPreview['seasons'][0]['episodes'][0]
type Season = TVShowMigrationPreview['seasons'][0]
type FileEntry = { proposedSlotId: number | null; needsReview: boolean; conflict?: string }

function isCleanlyAssigned(file: FileEntry) {
  return file.proposedSlotId !== null && !file.needsReview && !file.conflict
}

function hasConflict(file: FileEntry) {
  return !!file.conflict
}

function isNoMatch(file: FileEntry) {
  return file.needsReview && !file.conflict
}

// --- Movie filtering ---

export function filterMovies(preview: MigrationPreview, filter: FilterType): MovieMigrationPreview[] {
  if (filter === 'all') { return preview.movies }
  if (filter === 'assigned') { return preview.movies.filter((m) => m.files.every((f) => isCleanlyAssigned(f))) }
  if (filter === 'conflicts') { return preview.movies.filter((m) => m.files.some((f) => hasConflict(f))) }

  return preview.movies
    .map((movie) => {
      const filteredFiles = movie.files.filter((f) => isNoMatch(f))
      if (filteredFiles.length === 0) { return null }
      return { ...movie, files: filteredFiles }
    })
    .filter((m): m is MovieMigrationPreview => m !== null)
}

// --- TV show filtering ---

function rebuildSeason(season: Season, filteredEpisodes: Episode[]): Season | null {
  if (filteredEpisodes.length === 0) { return null }
  return {
    ...season,
    episodes: filteredEpisodes,
    totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
  }
}

function rebuildShow(show: TVShowMigrationPreview, filteredSeasons: Season[]): TVShowMigrationPreview | null {
  if (filteredSeasons.length === 0) { return null }
  return {
    ...show,
    seasons: filteredSeasons,
    totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
  }
}

function filterShowByEpisodePredicate(
  show: TVShowMigrationPreview,
  predicate: (ep: Episode) => boolean,
): TVShowMigrationPreview | null {
  const seasons = show.seasons
    .map((season) => rebuildSeason(season, season.episodes.filter((ep) => predicate(ep))))
    .filter((s): s is Season => s !== null)
  return rebuildShow(show, seasons)
}

function filterEpisodeNoMatch(ep: Episode): Episode | null {
  const files = ep.files.filter((f) => isNoMatch(f))
  if (files.length === 0) { return null }
  return { ...ep, files }
}

function filterShowNoMatch(show: TVShowMigrationPreview): TVShowMigrationPreview | null {
  const seasons = show.seasons
    .map((season) => {
      const episodes = season.episodes
        .map((ep) => filterEpisodeNoMatch(ep))
        .filter((e): e is Episode => e !== null)
      return rebuildSeason(season, episodes)
    })
    .filter((s): s is Season => s !== null)
  return rebuildShow(show, seasons)
}

function isEpisodeAssigned(ep: Episode): boolean {
  return ep.files.every((f) => isCleanlyAssigned(f))
}

function episodeHasConflict(ep: Episode): boolean {
  return ep.files.some((f) => hasConflict(f))
}

function filterShowsBy(
  shows: TVShowMigrationPreview[],
  predicate: (ep: Episode) => boolean,
): TVShowMigrationPreview[] {
  return shows
    .map((show) => filterShowByEpisodePredicate(show, predicate))
    .filter((s): s is TVShowMigrationPreview => s !== null)
}

export function filterTvShows(preview: MigrationPreview, filter: FilterType): TVShowMigrationPreview[] {
  if (filter === 'all') { return preview.tvShows }
  if (filter === 'assigned') { return filterShowsBy(preview.tvShows, isEpisodeAssigned) }
  if (filter === 'conflicts') { return filterShowsBy(preview.tvShows, episodeHasConflict) }

  return preview.tvShows
    .map((show) => filterShowNoMatch(show))
    .filter((s): s is TVShowMigrationPreview => s !== null)
}

// --- Visible file ID extraction ---

function extractFileIds(episodes: Episode[]): number[] {
  return episodes.flatMap((e) => e.files.map((f) => f.fileId))
}

function extractSeasonFileIds(seasons: Season[]): number[] {
  return seasons.flatMap((s) => extractFileIds(s.episodes))
}

function getSeasonFilteredFileIds(season: Season, predicate: (ep: Episode) => boolean): number[] {
  return extractFileIds(season.episodes.filter((ep) => predicate(ep)))
}

function getFilteredTvFileIds(shows: TVShowMigrationPreview[], predicate: (ep: Episode) => boolean): number[] {
  return shows.flatMap((show) => show.seasons.flatMap((season) => getSeasonFilteredFileIds(season, predicate)))
}

export function getVisibleMovieFileIds(preview: MigrationPreview, filter: FilterType): number[] {
  if (filter === 'nomatch') {
    return preview.movies.flatMap((m) => m.files.filter((f) => isNoMatch(f)).map((f) => f.fileId))
  }

  let movies = preview.movies
  if (filter === 'assigned') { movies = movies.filter((m) => m.files.every((f) => isCleanlyAssigned(f))) }
  else if (filter === 'conflicts') { movies = movies.filter((m) => m.files.some((f) => hasConflict(f))) }
  return movies.flatMap((m) => m.files.map((f) => f.fileId))
}

function getNoMatchEpisodeFileIds(episodes: Episode[]): number[] {
  return episodes.flatMap((ep) => ep.files.filter((f) => isNoMatch(f)).map((f) => f.fileId))
}

function getNoMatchTvFileIds(shows: TVShowMigrationPreview[]): number[] {
  return shows.flatMap((show) =>
    show.seasons.flatMap((season) => getNoMatchEpisodeFileIds(season.episodes)),
  )
}

export function getVisibleTvFileIds(preview: MigrationPreview, filter: FilterType): number[] {
  if (filter === 'all') { return preview.tvShows.flatMap((s) => extractSeasonFileIds(s.seasons)) }
  if (filter === 'assigned') { return getFilteredTvFileIds(preview.tvShows, isEpisodeAssigned) }
  if (filter === 'conflicts') { return getFilteredTvFileIds(preview.tvShows, episodeHasConflict) }
  return getNoMatchTvFileIds(preview.tvShows)
}
