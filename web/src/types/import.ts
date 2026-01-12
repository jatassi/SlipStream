// Import settings types
export interface ImportSettings {
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

export interface UpdateImportSettingsRequest {
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
export interface PatternPreviewRequest {
  pattern: string
  mediaType?: 'episode' | 'movie' | 'folder'
}

export interface TokenBreakdown {
  token: string
  name: string
  value: string
  empty: boolean
  modified: boolean
}

export interface PatternPreviewResponse {
  pattern: string
  preview: string
  valid: boolean
  error?: string
  tokens?: TokenBreakdown[]
}

export interface PatternValidateResponse {
  pattern: string
  valid: boolean
  error?: string
  tokens?: string[]
}

// Import status types
export interface ImportStatus {
  queueLength: number
  processingCount: number
}

// Pending import types
export interface PendingImport {
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
export interface ManualImportRequest {
  path: string
  mediaType: 'movie' | 'episode'
  mediaId: number
  seriesId?: number
  seasonNum?: number
}

export interface ManualImportResponse {
  success: boolean
  sourcePath: string
  destinationPath?: string
  linkMode?: string
  isUpgrade: boolean
  error?: string
}

// Preview import types
export interface ParsedMediaInfo {
  title?: string
  year?: number
  season?: number
  episode?: number
  endEpisode?: number
  quality?: string
  source?: string
  codec?: string
  audioCodecs?: string[]
  audioChannels?: string[]
  audioEnhancements?: string[]
  attributes?: string[]
  isTV: boolean
  isSeasonPack?: boolean
}

export interface SuggestedMatch {
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

export interface PreviewImportResponse {
  path: string
  fileName: string
  fileSize: number
  valid: boolean
  validationError?: string
  parsedInfo?: ParsedMediaInfo
  suggestedMatch?: SuggestedMatch
}

// Scan directory types
export interface ScannedFile {
  path: string
  fileName: string
  fileSize: number
  valid: boolean
  validationError?: string
  parsedInfo?: ParsedMediaInfo
  suggestedMatch?: SuggestedMatch
}

export interface ScanDirectoryResponse {
  path: string
  files: ScannedFile[]
  total: number
  valid: number
}

// Rename preview types
export interface RenamePreview {
  fileId: number
  mediaType: string
  mediaId: number
  mediaTitle: string
  currentPath: string
  currentFileName: string
  newPath: string
  newFileName: string
  needsRename: boolean
  error?: string
}

export interface RenamePreviewResponse {
  total: number
  previews: RenamePreview[]
}

export interface ExecuteRenameRequest {
  mediaType: string
  fileIds: number[]
}

export interface ExecuteRenameResponse {
  total: number
  succeeded: number
  failed: number
  skipped: number
  results: RenameResult[]
}

export interface RenameResult {
  fileId: number
  success: boolean
  oldPath: string
  newPath: string
  error?: string
}

// Filename parsing types
export interface ParsedTokenDetail {
  name: string
  value: string
  raw?: string
}

export interface ParseFilenameResponse {
  filename: string
  parsedInfo?: ParsedMediaInfo
  tokens: ParsedTokenDetail[]
}
