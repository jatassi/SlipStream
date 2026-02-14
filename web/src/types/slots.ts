// Version slot types for multi-version support

export type SlotProfile = {
  id: number
  name: string
  cutoff: number
}

export type SlotRootFolder = {
  id: number
  path: string
  name: string
}

export type Slot = {
  id: number
  slotNumber: number
  name: string
  enabled: boolean
  qualityProfileId: number | null
  displayOrder: number
  createdAt: string
  updatedAt: string
  // Root folder assignments for multi-version mode (Req 22.1.1-22.1.2)
  movieRootFolderId: number | null
  tvRootFolderId: number | null
  qualityProfile?: SlotProfile
  movieRootFolder?: SlotRootFolder
  tvRootFolder?: SlotRootFolder
  fileCount?: number
}

export type MultiVersionSettings = {
  enabled: boolean
  dryRunCompleted: boolean
  lastMigrationAt?: string
  createdAt: string
  updatedAt: string
}

export type UpdateSlotInput = {
  name: string
  enabled: boolean
  qualityProfileId: number | null
  displayOrder: number
  movieRootFolderId?: number | null
  tvRootFolderId?: number | null
}

export type UpdateMultiVersionSettingsInput = {
  enabled: boolean
}

export type SetEnabledInput = {
  enabled: boolean
}

export type SetProfileInput = {
  qualityProfileId: number | null
}

export type AttributeIssue = {
  attribute: string
  message: string
}

export type SlotConflict = {
  slotAName: string
  slotBName: string
  issues: AttributeIssue[]
}

export type ValidateConfigurationResponse = {
  valid: boolean
  errors?: string[]
  conflicts?: SlotConflict[]
}

// Slot Assignment types
export type SlotAssignment = {
  slotId: number
  slotNumber: number
  slotName: string
  matchScore: number
  isUpgrade: boolean
  isNewFill: boolean
  needsUpgrade: boolean
  confidence: number
  currentFileId?: number
  currentQuality?: string
}

export type SlotEvaluation = {
  assignments: SlotAssignment[]
  recommendedSlotId: number
  requiresSelection: boolean
  matchingCount: number
}

export type MovieSlotAssignment = {
  id: number
  movieId: number
  slotId: number
  fileId: number | null
  monitored: boolean
  slotName: string
  slotNumber: number
  qualityProfileId: number | null
}

export type EpisodeSlotAssignment = {
  id: number
  episodeId: number
  slotId: number
  fileId: number | null
  monitored: boolean
  slotName: string
  slotNumber: number
  qualityProfileId: number | null
}

export type AssignFileInput = {
  fileId: number
}

// Slot Status types (Phase 5: Status & Monitoring)

export type SlotStatus = {
  slotId: number
  slotNumber: number
  slotName: string
  monitored: boolean
  status: 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
  statusMessage?: string | null
  activeDownloadId?: string | null
  fileId?: number
  currentQuality?: string
  currentQualityId?: number
  profileCutoff: number
}

export type MediaStatus = {
  mediaType: string
  mediaId: number
  status: string
  slotStatuses: SlotStatus[]
  filledSlots: number
  emptySlots: number
  monitoredSlots: number
}

export type SetMonitoredInput = {
  monitored: boolean
}

// Debug API types (Phase 13: Debug & Testing)

export type ParseReleaseInput = {
  releaseTitle: string
}

export type ParseReleaseOutput = {
  title: string
  year?: number
  season?: number
  episode?: number
  quality?: string
  source?: string
  videoCodec?: string
  audioCodecs?: string[]
  audioChannels?: string[]
  hdrFormats?: string[]
  releaseGroup?: string
  isSeasonPack: boolean
  isCompleteSeries: boolean
  isTv: boolean
  qualityScore: number
}

export type ProfileMatchInput = {
  releaseTitle: string
  qualityProfileId: number
}

export type AttributeMatchResult = {
  mode: string // "acceptable", "required", "preferred"
  profileValues: string[]
  releaseValue: string
  matches: boolean
  score: number
  reason?: string
}

export type ProfileMatchOutput = {
  release: ParseReleaseOutput
  profileId: number
  profileName: string
  allAttributesMatch: boolean
  totalScore: number
  qualityScore: number
  combinedScore: number
  hdrMatch: AttributeMatchResult
  videoCodecMatch: AttributeMatchResult
  audioCodecMatch: AttributeMatchResult
  audioChannelMatch: AttributeMatchResult
}

export type SimulateImportInput = {
  releaseTitle: string
  mediaType: string // "movie" or "episode"
  mediaId: number
}

export type SlotEvaluationDetail = {
  slotId: number
  slotNumber: number
  slotName: string
  profileId?: number
  profileName: string
  matchScore: number
  attributeScore: number
  qualityScore: number
  isEmpty: boolean
  isUpgrade: boolean
  currentQuality?: string
  confidence: number
  attributesPassed: boolean
}

export type SimulateImportOutput = {
  release: ParseReleaseOutput
  slotEvaluations: SlotEvaluationDetail[]
  recommendedSlot?: SlotEvaluationDetail
  requiresSelection: boolean
  matchingCount: number
  importAction: string // "accept", "reject", "user_choice"
  importReason: string
}

// File Naming Validation types (Req 4.1.1-4.1.5)

export type DifferentiatorAttribute = 'HDR' | 'Video Codec' | 'Audio Codec' | 'Audio Channels'

export type MissingTokenInfo = {
  attribute: DifferentiatorAttribute
  tokenName: string
  description: string
  suggestedToken: string
}

export type NamingValidationResult = {
  valid: boolean
  missingTokens?: MissingTokenInfo[]
  requiredAttributes?: DifferentiatorAttribute[]
  warnings?: string[]
}

export type ValidateNamingInput = {
  movieFileFormat: string
  episodeFileFormat: string
}

export type SlotNamingValidation = {
  movieFormatValid: boolean
  episodeFormatValid: boolean
  movieValidation: NamingValidationResult
  episodeValidation: NamingValidationResult
  requiredAttributes: DifferentiatorAttribute[]
  canProceed: boolean
  warnings: string[]
  qualityTierExclusive?: boolean // Profiles are exclusive via quality tiers only
  noEnabledSlots?: boolean // No enabled slots with profiles found
}

// Migration/Dry Run Types (Req 14.1.1-14.2.3)

export type SlotRejectionInfo = {
  slotId: number
  slotName: string
  reasons: string[]
}

export type FileMigrationPreview = {
  fileId: number
  path: string
  quality: string
  size: number
  proposedSlotId: number | null
  proposedSlotName?: string
  matchScore: number
  needsReview: boolean
  conflict?: string
  slotRejections?: SlotRejectionInfo[]
}

export type MovieMigrationPreview = {
  movieId: number
  title: string
  year?: number
  files: FileMigrationPreview[]
  hasConflict: boolean
  conflicts?: string[]
}

export type EpisodeMigrationPreview = {
  episodeId: number
  episodeNumber: number
  title?: string
  files: FileMigrationPreview[]
  hasConflict: boolean
}

export type SeasonMigrationPreview = {
  seasonNumber: number
  episodes: EpisodeMigrationPreview[]
  totalFiles: number
  hasConflict: boolean
}

export type TVShowMigrationPreview = {
  seriesId: number
  title: string
  seasons: SeasonMigrationPreview[]
  totalFiles: number
  hasConflict: boolean
}

export type MigrationSummary = {
  totalMovies: number
  totalTvShows: number
  totalFiles: number
  filesWithSlots: number
  filesNeedingReview: number
  conflicts: number
}

export type MigrationPreview = {
  movies: MovieMigrationPreview[]
  tvShows: TVShowMigrationPreview[]
  summary: MigrationSummary
}

export type MigrationResult = {
  success: boolean
  filesAssigned: number
  filesQueued: number
  errors?: string[]
  completedAt: string
}

// File override for manual migration adjustments
export type FileOverride = {
  fileId: number
  type: 'ignore' | 'assign' | 'unassign'
  slotId?: number // Required when type is 'assign'
}

// Input for execute migration with optional overrides
export type ExecuteMigrationInput = {
  overrides?: FileOverride[]
}

// Debug Preview Generation Types

export type MockFile = {
  fileId: number
  path: string
  quality: string
  size: number
}

export type MockMovie = {
  movieId: number
  title: string
  year?: number
  files: MockFile[]
}

export type MockEpisode = {
  episodeId: number
  episodeNumber: number
  title?: string
  files: MockFile[]
}

export type MockSeason = {
  seasonNumber: number
  episodes: MockEpisode[]
}

export type MockTVShow = {
  seriesId: number
  title: string
  seasons: MockSeason[]
}

export type GeneratePreviewInput = {
  movies: MockMovie[]
  tvShows: MockTVShow[]
}
