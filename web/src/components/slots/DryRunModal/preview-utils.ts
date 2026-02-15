import type {
  FileMigrationPreview,
  ManualEdit,
  MigrationPreview,
} from './types'

export function applyEditToFile(
  file: FileMigrationPreview,
  manualEdits: Map<number, ManualEdit>,
): FileMigrationPreview {
  const edit = manualEdits.get(file.fileId)
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

function fileHasIssue(file: FileMigrationPreview, edits: Map<number, ManualEdit>): boolean {
  const edited = applyEditToFile(file, edits)
  return !!edited.conflict || edited.needsReview
}

function filesHaveIssue(files: FileMigrationPreview[], edits: Map<number, ManualEdit>): boolean {
  return files.some((f) => fileHasIssue(f, edits))
}

function editMovies(preview: MigrationPreview, edits: Map<number, ManualEdit>) {
  return preview.movies.map((movie) => ({
    ...movie,
    files: movie.files.map((f) => applyEditToFile(f, edits)),
    hasConflict: filesHaveIssue(movie.files, edits),
  }))
}

function editEpisodes(
  episodes: MigrationPreview['tvShows'][0]['seasons'][0]['episodes'],
  edits: Map<number, ManualEdit>,
) {
  return episodes.map((episode) => ({
    ...episode,
    files: episode.files.map((f) => applyEditToFile(f, edits)),
    hasConflict: filesHaveIssue(episode.files, edits),
  }))
}

function editSeasons(seasons: MigrationPreview['tvShows'][0]['seasons'], edits: Map<number, ManualEdit>) {
  return seasons.map((season) => {
    const episodes = editEpisodes(season.episodes, edits)
    return {
      ...season,
      episodes,
      hasConflict: season.episodes.some((e) => filesHaveIssue(e.files, edits)),
    }
  })
}

function seasonHasIssue(season: MigrationPreview['tvShows'][0]['seasons'][0], edits: Map<number, ManualEdit>): boolean {
  return season.episodes.some((e) => filesHaveIssue(e.files, edits))
}

function editTvShows(preview: MigrationPreview, edits: Map<number, ManualEdit>) {
  return preview.tvShows.map((show) => ({
    ...show,
    seasons: editSeasons(show.seasons, edits),
    hasConflict: show.seasons.some((s) => seasonHasIssue(s, edits)),
  }))
}

function collectEpisodeFiles(episodes: MigrationPreview['tvShows'][0]['seasons'][0]['episodes']) {
  return episodes.flatMap((e) => e.files)
}

function computeSummary(
  editedMovies: ReturnType<typeof editMovies>,
  editedTvShows: ReturnType<typeof editTvShows>,
) {
  const allMovieFiles = editedMovies.flatMap((m) => m.files)
  const allTvFiles = editedTvShows.flatMap((s) => s.seasons.flatMap((se) => collectEpisodeFiles(se.episodes)))
  const allFiles = [...allMovieFiles, ...allTvFiles]

  return {
    totalMovies: editedMovies.length,
    totalTvShows: editedTvShows.length,
    totalFiles: allFiles.length,
    filesWithSlots: allFiles.filter((f) => f.proposedSlotId !== null && !f.needsReview && !f.conflict).length,
    filesNeedingReview: allFiles.filter((f) => f.needsReview && !f.conflict).length,
    conflicts: allFiles.filter((f) => !!f.conflict).length,
  }
}

export function computeEditedPreview(
  preview: MigrationPreview,
  manualEdits: Map<number, ManualEdit>,
): MigrationPreview {
  if (manualEdits.size === 0) {
    return preview
  }

  const movies = editMovies(preview, manualEdits)
  const tvShows = editTvShows(preview, manualEdits)

  return { movies, tvShows, summary: computeSummary(movies, tvShows) }
}
