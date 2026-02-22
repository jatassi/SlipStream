// Import settings types
export type ImportSettings = {
  validationLevel: 'basic' | 'standard' | 'full'
  minimumFileSizeMB: number
  videoExtensions: string[]
  matchConflictBehavior: 'trust_queue' | 'trust_parse' | 'fail'
  unknownMediaBehavior: 'ignore' | 'auto_add'
  renameEpisodes: boolean
  replaceIllegalCharacters: boolean
  colonReplacement: 'delete' | 'dash' | 'space_dash' | 'space_dash_space' | 'smart' | 'custom'
  customColonReplacement?: string
  standardEpisodeFormat: string
  dailyEpisodeFormat: string
  animeEpisodeFormat: string
  seriesFolderFormat: string
  seasonFolderFormat: string
  specialsFolderFormat: string
  multiEpisodeStyle: 'extend' | 'duplicate' | 'repeat' | 'scene' | 'range' | 'prefixed_range'
  renameMovies: boolean
  movieFolderFormat: string
  movieFileFormat: string
}

export type UpdateImportSettingsRequest = {
  validationLevel?: string
  minimumFileSizeMB?: number
  videoExtensions?: string[]
  matchConflictBehavior?: string
  unknownMediaBehavior?: string
  renameEpisodes?: boolean
  replaceIllegalCharacters?: boolean
  colonReplacement?: string
  customColonReplacement?: string
  standardEpisodeFormat?: string
  dailyEpisodeFormat?: string
  animeEpisodeFormat?: string
  seriesFolderFormat?: string
  seasonFolderFormat?: string
  specialsFolderFormat?: string
  multiEpisodeStyle?: string
  renameMovies?: boolean
  movieFolderFormat?: string
  movieFileFormat?: string
}

// Pattern preview types
export type PatternPreviewRequest = {
  pattern: string
  mediaType?: 'episode' | 'movie' | 'folder'
}

export type TokenBreakdown = {
  token: string
  name: string
  value: string
  empty: boolean
  modified: boolean
}

export type PatternPreviewResponse = {
  pattern: string
  preview: string
  valid: boolean
  error?: string
  tokens?: TokenBreakdown[]
}

// Pending import types
export type PendingImport = {
  id?: number
  filePath: string
  fileName: string
  fileSize: number
  status: string
  mediaType?: string
  mediaId?: number
  mediaTitle?: string
  error?: string
  attempts: number
  isProcessing: boolean
}

// Manual import types
export type ManualImportRequest = {
  path: string
  mediaType: 'movie' | 'episode'
  mediaId: number
  seriesId?: number
  seasonNum?: number
  targetSlotId?: number
}

export type ImportSlotAssignment = {
  slotId: number
  slotNumber: number
  slotName: string
  matchScore: number
  isUpgrade: boolean
  isNewFill: boolean
}

export type ManualImportResponse = {
  success: boolean
  sourcePath: string
  destinationPath?: string
  linkMode?: string
  isUpgrade: boolean
  error?: string
  requiresSlotSelection: boolean
  slotAssignments: ImportSlotAssignment[]
  recommendedSlotId?: number
  assignedSlotId?: number
}

// Preview import types
export type ParsedMediaInfo = {
  title?: string
  year?: number
  season: number
  episode: number
  endEpisode?: number
  quality?: string
  source?: string
  codec?: string
  audioCodecs?: string[]
  audioChannels?: string[]
  audioEnhancements?: string[]
  attributes?: string[]
  hdrFormats?: string[]
  isTV: boolean
  isSeasonPack?: boolean
}

export type SuggestedMatch = {
  mediaType: string
  mediaId: number
  mediaTitle: string
  confidence: number
  year?: number
  seasonNum?: number
  episodeNum?: number
  seriesId?: number
  seriesTitle?: string
}

// Scan directory types
export type ScannedFile = {
  path: string
  fileName: string
  fileSize: number
  valid: boolean
  validationError?: string
  parsedInfo?: ParsedMediaInfo
  suggestedMatch?: SuggestedMatch
}

export type ScanDirectoryResponse = {
  path: string
  files: ScannedFile[]
  total: number
  valid: number
}

// Filename parsing types
export type ParsedTokenDetail = {
  name: string
  value: string
  raw?: string
}

export type ParseFilenameResponse = {
  filename: string
  parsedInfo?: ParsedMediaInfo
  tokens: ParsedTokenDetail[]
}
