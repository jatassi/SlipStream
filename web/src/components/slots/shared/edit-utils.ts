import type {
  FileMigrationPreview,
  ManualEdit,
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
} from './types'

function applyEditToFile(
  file: FileMigrationPreview,
  edits: Map<number, ManualEdit>,
): FileMigrationPreview {
  const edit = edits.get(file.fileId)
  if (!edit) {
    return file
  }

  switch (edit.type) {
    case 'ignore': {
      return {
        ...file,
        proposedSlotId: null,
        proposedSlotName: undefined,
        needsReview: false,
        conflict: undefined,
        matchScore: 0,
      }
    }
    case 'assign': {
      return {
        ...file,
        proposedSlotId: edit.slotId,
        proposedSlotName: edit.slotName,
        needsReview: false,
        conflict: undefined,
        matchScore: 100,
      }
    }
    case 'unassign': {
      return {
        ...file,
        proposedSlotId: null,
        proposedSlotName: undefined,
        needsReview: true,
        conflict: undefined,
        matchScore: 0,
      }
    }
    default: {
      return file
    }
  }
}

function hasFileIssue(file: FileMigrationPreview, edits: Map<number, ManualEdit>): boolean {
  const edited = applyEditToFile(file, edits)
  return !!edited.conflict || edited.needsReview
}

function seasonHasIssue(
  season: TVShowMigrationPreview['seasons'][number],
  edits: Map<number, ManualEdit>,
): boolean {
  return season.episodes.some((e) => e.files.some((f) => hasFileIssue(f, edits)))
}

function buildEditedEpisode(
  episode: TVShowMigrationPreview['seasons'][number]['episodes'][number],
  edits: Map<number, ManualEdit>,
) {
  return {
    ...episode,
    files: episode.files.map((f) => applyEditToFile(f, edits)),
    hasConflict: episode.files.some((f) => hasFileIssue(f, edits)),
  }
}

function buildEditedSeason(
  season: TVShowMigrationPreview['seasons'][number],
  edits: Map<number, ManualEdit>,
) {
  return {
    ...season,
    episodes: season.episodes.map((ep) => buildEditedEpisode(ep, edits)),
    hasConflict: seasonHasIssue(season, edits),
  }
}

function buildEditedMovies(
  movies: MovieMigrationPreview[],
  edits: Map<number, ManualEdit>,
): MovieMigrationPreview[] {
  return movies.map((movie) => ({
    ...movie,
    files: movie.files.map((f) => applyEditToFile(f, edits)),
    hasConflict: movie.files.some((f) => hasFileIssue(f, edits)),
  }))
}

function buildEditedTvShows(
  shows: TVShowMigrationPreview[],
  edits: Map<number, ManualEdit>,
): TVShowMigrationPreview[] {
  return shows.map((show) => ({
    ...show,
    seasons: show.seasons.map((s) => buildEditedSeason(s, edits)),
    hasConflict: show.seasons.some((s) => seasonHasIssue(s, edits)),
  }))
}

function collectSeasonFiles(season: TVShowMigrationPreview['seasons'][number]): FileMigrationPreview[] {
  return season.episodes.flatMap((e) => e.files)
}

function collectAllTvFiles(tvShows: TVShowMigrationPreview[]): FileMigrationPreview[] {
  return tvShows.flatMap((s) => s.seasons.flatMap((se) => collectSeasonFiles(se)))
}

export function computeEditedPreview(
  preview: MigrationPreview | null,
  edits: Map<number, ManualEdit>,
): MigrationPreview | null {
  if (!preview) {
    return null
  }
  if (edits.size === 0) {
    return preview
  }

  const movies = buildEditedMovies(preview.movies, edits)
  const tvShows = buildEditedTvShows(preview.tvShows, edits)

  const allFiles = [...movies.flatMap((m) => m.files), ...collectAllTvFiles(tvShows)]

  const summary = {
    totalMovies: movies.length,
    totalTvShows: tvShows.length,
    totalFiles: allFiles.length,
    filesWithSlots: allFiles.filter(
      (f) => f.proposedSlotId !== null && !f.needsReview && !f.conflict,
    ).length,
    filesNeedingReview: allFiles.filter((f) => f.needsReview && !f.conflict).length,
    conflicts: allFiles.filter((f) => !!f.conflict).length,
  }

  return { movies, tvShows, summary }
}
