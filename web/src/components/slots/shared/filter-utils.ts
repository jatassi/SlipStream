import type {
  FileMigrationPreview,
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
} from './types'

export type FilterType = 'all' | 'assigned' | 'conflicts' | 'nomatch'
type Episode = TVShowMigrationPreview['seasons'][number]['episodes'][number]

function isCleanlyAssigned(file: FileMigrationPreview): boolean {
  return file.proposedSlotId !== null && !file.needsReview && !file.conflict
}

function isUnmatched(file: FileMigrationPreview): boolean {
  return file.needsReview && !file.conflict
}

function hasConflict(file: FileMigrationPreview): boolean {
  return !!file.conflict
}

function episodeAllAssigned(ep: Episode): boolean {
  return ep.files.every((f) => isCleanlyAssigned(f))
}

function episodeHasConflict(ep: Episode): boolean {
  return ep.files.some((f) => hasConflict(f))
}

function filterSeasons(
  show: TVShowMigrationPreview,
  episodePredicate: (episode: Episode) => boolean,
): TVShowMigrationPreview | null {
  const filteredSeasons = show.seasons
    .map((season) => {
      const filteredEpisodes = season.episodes.filter((ep) => episodePredicate(ep))
      if (filteredEpisodes.length === 0) {
        return null
      }
      return {
        ...season,
        episodes: filteredEpisodes,
        totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
      }
    })
    .filter((s): s is NonNullable<typeof s> => s !== null)

  if (filteredSeasons.length === 0) {
    return null
  }

  return {
    ...show,
    seasons: filteredSeasons,
    totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
  }
}

function filterEpisodeFiles(
  episode: Episode,
  fileFilter: (file: FileMigrationPreview) => boolean,
): Episode | null {
  const files = episode.files.filter((f) => fileFilter(f))
  if (files.length === 0) {
    return null
  }
  return { ...episode, files }
}

function filterSeasonsWithFileTransform(
  show: TVShowMigrationPreview,
  fileFilter: (file: FileMigrationPreview) => boolean,
): TVShowMigrationPreview | null {
  const filteredSeasons = show.seasons
    .map((season) => {
      const filteredEpisodes = season.episodes
        .map((ep) => filterEpisodeFiles(ep, fileFilter))
        .filter((e): e is NonNullable<typeof e> => e !== null)

      if (filteredEpisodes.length === 0) {
        return null
      }
      return {
        ...season,
        episodes: filteredEpisodes,
        totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
      }
    })
    .filter((s): s is NonNullable<typeof s> => s !== null)

  if (filteredSeasons.length === 0) {
    return null
  }

  return {
    ...show,
    seasons: filteredSeasons,
    totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
  }
}

export function filterMovies(
  preview: MigrationPreview | null,
  filter: FilterType,
): MovieMigrationPreview[] {
  if (!preview) {
    return []
  }
  if (filter === 'all') {
    return preview.movies
  }
  if (filter === 'assigned') {
    return preview.movies.filter((movie) => movie.files.every((f) => isCleanlyAssigned(f)))
  }
  if (filter === 'conflicts') {
    return preview.movies.filter((movie) => movie.files.some((f) => hasConflict(f)))
  }

  return preview.movies
    .map((movie) => {
      const filteredFiles = movie.files.filter((f) => isUnmatched(f))
      if (filteredFiles.length === 0) {
        return null
      }
      return { ...movie, files: filteredFiles }
    })
    .filter((m): m is MovieMigrationPreview => m !== null)
}

export function filterTvShows(
  preview: MigrationPreview | null,
  filter: FilterType,
): TVShowMigrationPreview[] {
  if (!preview) {
    return []
  }
  if (filter === 'all') {
    return preview.tvShows
  }
  if (filter === 'assigned') {
    return preview.tvShows
      .map((show) => filterSeasons(show, episodeAllAssigned))
      .filter((s): s is TVShowMigrationPreview => s !== null)
  }
  if (filter === 'conflicts') {
    return preview.tvShows
      .map((show) => filterSeasons(show, episodeHasConflict))
      .filter((s): s is TVShowMigrationPreview => s !== null)
  }

  return preview.tvShows
    .map((show) => filterSeasonsWithFileTransform(show, (f) => isUnmatched(f)))
    .filter((s): s is TVShowMigrationPreview => s !== null)
}

function collectMatchingEpisodeFileIds(
  show: TVShowMigrationPreview,
  episodeFilter: (episode: Episode) => boolean,
): number[] {
  const ids: number[] = []
  for (const season of show.seasons) {
    for (const episode of season.episodes) {
      if (!episodeFilter(episode)) {
        continue
      }
      for (const file of episode.files) {
        ids.push(file.fileId)
      }
    }
  }
  return ids
}

function collectFileIdsFromEpisode(
  episode: Episode,
  fileFilter: (file: FileMigrationPreview) => boolean,
  ids: number[],
): void {
  for (const file of episode.files) {
    if (fileFilter(file)) {
      ids.push(file.fileId)
    }
  }
}

function collectMatchingFileIds(
  show: TVShowMigrationPreview,
  fileFilter: (file: FileMigrationPreview) => boolean,
): number[] {
  const ids: number[] = []
  for (const season of show.seasons) {
    for (const episode of season.episodes) {
      collectFileIdsFromEpisode(episode, fileFilter, ids)
    }
  }
  return ids
}

export function getVisibleMovieFileIds(
  preview: MigrationPreview | null,
  filter: FilterType,
): number[] {
  if (!preview) {
    return []
  }
  if (filter === 'nomatch') {
    return preview.movies.flatMap((movie) =>
      movie.files.filter((f) => isUnmatched(f)).map((f) => f.fileId),
    )
  }

  let movies = preview.movies
  if (filter === 'assigned') {
    movies = movies.filter((movie) => movie.files.every((f) => isCleanlyAssigned(f)))
  } else if (filter === 'conflicts') {
    movies = movies.filter((movie) => movie.files.some((f) => hasConflict(f)))
  }

  return movies.flatMap((m) => m.files.map((f) => f.fileId))
}

export function getVisibleTvFileIds(
  preview: MigrationPreview | null,
  filter: FilterType,
): number[] {
  if (!preview) {
    return []
  }

  if (filter === 'assigned') {
    return preview.tvShows.flatMap((show) =>
      collectMatchingEpisodeFileIds(show, episodeAllAssigned),
    )
  }
  if (filter === 'conflicts') {
    return preview.tvShows.flatMap((show) =>
      collectMatchingEpisodeFileIds(show, episodeHasConflict),
    )
  }
  if (filter === 'nomatch') {
    return preview.tvShows.flatMap((show) =>
      collectMatchingFileIds(show, (f) => isUnmatched(f)),
    )
  }

  return preview.tvShows.flatMap((show) =>
    collectMatchingFileIds(show, () => true),
  )
}
