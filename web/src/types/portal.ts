// Portal User types
export interface PortalUser {
  id: number
  username: string
  displayName: string | null
  qualityProfileId: number | null
  autoApprove: boolean
  enabled: boolean
  isAdmin: boolean
  createdAt: string
  updatedAt: string
}

export interface PortalUserWithQuota extends PortalUser {
  quota: UserQuota | null
}

// Auth types
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: PortalUser
  isAdmin: boolean
}

export interface SignupRequest {
  token: string
  password: string
  displayName?: string
}

export interface SignupResponse {
  token: string
  user: PortalUser
}

export interface UpdateProfileRequest {
  username?: string
  displayName?: string
  password?: string
}

// Invitation types
export interface Invitation {
  id: number
  username: string
  token: string
  expiresAt: string
  usedAt: string | null
  createdAt: string
}

export interface CreateInvitationRequest {
  username: string
}

export interface ValidateInvitationResponse {
  valid: boolean
  username: string
  expiresAt: string
}

export interface VerifyPinResponse {
  valid: boolean
}

// Request types
export type RequestStatus = 'pending' | 'approved' | 'denied' | 'downloading' | 'available' | 'cancelled'
export type PortalMediaType = 'movie' | 'series' | 'season' | 'episode'

export interface Request {
  id: number
  userId: number
  mediaType: PortalMediaType
  tmdbId: number | null
  tvdbId: number | null
  title: string
  year: number | null
  seasonNumber: number | null
  episodeNumber: number | null
  status: RequestStatus
  monitorFuture: boolean
  deniedReason: string | null
  approvedAt: string | null
  approvedBy: number | null
  mediaId: number | null
  targetSlotId: number | null
  posterUrl: string | null
  requestedSeasons: number[] | null
  createdAt: string
  updatedAt: string
  user?: PortalUser
  isWatching?: boolean
}

export interface CreateRequestInput {
  mediaType: PortalMediaType
  tmdbId?: number
  tvdbId?: number
  title: string
  year?: number
  seasonNumber?: number
  episodeNumber?: number
  monitorFuture?: boolean
  posterUrl?: string
  requestedSeasons?: number[]
}

export interface RequestListFilters {
  status?: RequestStatus
  mediaType?: PortalMediaType
  userId?: number
}

export interface ApproveRequestInput {
  action: 'approve_only' | 'auto_search' | 'manual_search'
  rootFolderId?: number
}

export interface DenyRequestInput {
  reason?: string
}

export interface BatchApproveInput {
  ids: number[]
  action: 'approve_only' | 'auto_search' | 'manual_search'
  rootFolderId?: number
}

export interface BatchDenyInput {
  ids: number[]
  reason?: string
}

// Quota types
export interface UserQuota {
  userId: number
  moviesLimit: number | null
  seasonsLimit: number | null
  episodesLimit: number | null
  moviesUsed: number
  seasonsUsed: number
  episodesUsed: number
  periodStart: string
}

export interface QuotaLimits {
  moviesLimit?: number | null
  seasonsLimit?: number | null
  episodesLimit?: number | null
}

// User notification types
export interface UserNotification {
  id: number
  userId: number
  type: string
  name: string
  settings: Record<string, unknown>
  onAvailable: boolean
  onApproved: boolean
  onDenied: boolean
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateUserNotificationInput {
  type: string
  name: string
  settings: Record<string, unknown>
  onAvailable: boolean
  onApproved: boolean
  onDenied: boolean
  enabled: boolean
}

// Search with availability
export interface AvailabilityInfo {
  inLibrary: boolean
  existingSlots: SlotInfo[]
  canRequest: boolean
  existingRequestId: number | null
  existingRequestUserId: number | null
  existingRequestStatus: RequestStatus | null
  mediaId: number | null
  addedAt: string | null
}

export interface SlotInfo {
  id: number
  name: string
  quality: string
}

export interface PortalMovieSearchResult {
  id: number
  tmdbId: number
  title: string
  year: number | null
  overview: string | null
  posterUrl: string | null
  backdropUrl: string | null
  availability?: AvailabilityInfo
}

export interface PortalSeriesSearchResult {
  id: number
  tmdbId: number
  tvdbId: number | null
  title: string
  year: number | null
  overview: string | null
  posterUrl: string | null
  backdropUrl: string | null
  availability?: AvailabilityInfo
}

export interface SeasonInfo {
  seasonNumber: number
  name: string
  overview?: string
  posterUrl?: string
  airDate?: string
}

// Request settings
export interface RequestSettings {
  enabled: boolean
  defaultMovieQuota: number
  defaultSeasonQuota: number
  defaultEpisodeQuota: number
  defaultRootFolderId: number | null
  adminNotifyNew: boolean
  searchRateLimit: number
}

// Admin user management
export interface AdminUpdateUserInput {
  qualityProfileId?: number | null
  autoApprove?: boolean
  quotaOverride?: QuotaLimits
}

// Auto-approve result
export interface AutoApproveResult {
  autoApproved: boolean
  quotaExceeded: boolean
  searchStarted: boolean
  searchFound: boolean
  searchError?: string
}

export interface CreateRequestResponse {
  request: Request
  autoApprove: AutoApproveResult
}

// Portal download (queue item filtered to user's requests)
export interface PortalDownload {
  id: string
  clientId: number
  clientName: string
  title: string
  mediaType: 'movie' | 'series' | 'unknown'
  status: 'queued' | 'downloading' | 'paused' | 'completed' | 'failed'
  progress: number
  size: number
  downloadedSize: number
  downloadSpeed: number
  eta: number
  season?: number
  episode?: number
  movieId?: number
  seriesId?: number
  seasonNumber?: number
  isSeasonPack?: boolean
  requestId: number
  requestTitle: string
  requestMediaId?: number
  tmdbId?: number
  tvdbId?: number
}
