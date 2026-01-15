import type {
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
  FileMigrationPreview,
  Slot,
} from '@/types'

// Manual edit types for tracking user overrides
export type ManualEdit =
  | { type: 'ignore' }
  | { type: 'assign'; slotId: number; slotName: string }
  | { type: 'unassign' }

export interface DryRunModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onMigrationComplete: () => void
}

export interface SummaryCardProps {
  label: string
  value: number
  icon: React.ElementType
  variant?: 'default' | 'success' | 'warning' | 'error'
  active?: boolean
  onClick?: () => void
}

export interface MoviesListProps {
  movies: MovieMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export interface MovieItemProps {
  movie: MovieMigrationPreview
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export interface TVShowsListProps {
  shows: TVShowMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export interface TVShowItemProps {
  show: TVShowMigrationPreview
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export interface SeasonItemProps {
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

export interface EpisodeItemProps {
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

export interface FileItemProps {
  file: FileMigrationPreview
  compact?: boolean
  isSelected: boolean
  isIgnored?: boolean
  onToggleSelection: () => void
}

export interface AssignModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  slots: Slot[]
  selectedCount: number
  onAssign: (slotId: number, slotName: string) => void
}

export interface AggregatedFileTooltipProps {
  files: FileMigrationPreview[]
}

// Re-export types that are used across components
export type {
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
  FileMigrationPreview,
  Slot,
}
