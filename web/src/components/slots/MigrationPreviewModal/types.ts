import type {
  FileMigrationPreview,
  MovieMigrationPreview,
  Slot,
  TVShowMigrationPreview,
} from '@/types'

// Manual edit types for tracking user overrides
export type ManualEdit =
  | { type: 'ignore' }
  | { type: 'assign'; slotId: number; slotName: string }
  | { type: 'unassign' }

export type DryRunModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onMigrationComplete: () => void
}

export type SummaryCardProps = {
  label: string
  value: number
  icon: React.ElementType
  variant?: 'default' | 'success' | 'warning' | 'error'
  active?: boolean
  onClick?: () => void
}

export type MoviesListProps = {
  movies: MovieMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type MovieItemProps = {
  movie: MovieMigrationPreview
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type TVShowsListProps = {
  shows: TVShowMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type TVShowItemProps = {
  show: TVShowMigrationPreview
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type SeasonItemProps = {
  season: {
    seasonNumber: number
    episodes: {
      episodeId: number
      episodeNumber: number
      title?: string
      files: FileMigrationPreview[]
      hasConflict: boolean
    }[]
    totalFiles: number
    hasConflict: boolean
  }
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type EpisodeItemProps = {
  episode: {
    episodeId: number
    episodeNumber: number
    title?: string
    files: FileMigrationPreview[]
    hasConflict: boolean
  }
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export type FileItemProps = {
  file: FileMigrationPreview
  compact?: boolean
  isSelected: boolean
  isIgnored?: boolean
  onToggleSelection: () => void
}

export type AssignModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  slots: Slot[]
  selectedCount: number
  onAssign: (slotId: number, slotName: string) => void
}

export type AggregatedFileTooltipProps = {
  files: FileMigrationPreview[]
}

// Re-export types that are used across components

export {
  type FileMigrationPreview,
  type MigrationPreview,
  type MovieMigrationPreview,
  type Slot,
  type TVShowMigrationPreview,
} from '@/types'
