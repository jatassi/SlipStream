// Portal User types
export type PortalUser = {
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

export type PortalUserWithQuota = {
  quota: UserQuota | null
} & PortalUser

// Auth types
export type LoginRequest = {
  username: string
  password: string
}

export type LoginResponse = {
  token: string
  user: PortalUser
  isAdmin: boolean
}

export type SignupRequest = {
  token: string
  password: string
  displayName?: string
}

export type SignupResponse = {
  token: string
  user: PortalUser
}

export type UpdateProfileRequest = {
  username?: string
  displayName?: string
  password?: string
}

// Invitation types
export type Invitation = {
  id: number
  username: string
  token: string
  expiresAt: string
  usedAt: string | null
  createdAt: string
  qualityProfileId: number | null
  autoApprove: boolean
}

export type CreateInvitationRequest = {
  username: string
  qualityProfileId?: number | null
  autoApprove?: boolean
}

export type ValidateInvitationResponse = {
  valid: boolean
  username: string
  expiresAt: string
}

export type VerifyPinResponse = {
  valid: boolean
}

// Request types
export type RequestStatus =
  | 'pending'
  | 'approved'
  | 'denied'
  | 'downloading'
  | 'failed'
  | 'available'
  | 'cancelled'
export type PortalMediaType = 'movie' | 'series' | 'season' | 'episode'

export type Request = {
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

export type CreateRequestInput = {
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

export type RequestListFilters = {
  status?: RequestStatus
  mediaType?: PortalMediaType
  userId?: number
  scope?: 'mine' | 'all'
}

export type ApproveRequestInput = {
  action: 'approve_only' | 'auto_search' | 'manual_search'
  rootFolderId?: number
}

export type DenyRequestInput = {
  reason?: string
}

export type BatchApproveInput = {
  ids: number[]
  action: 'approve_only' | 'auto_search' | 'manual_search'
  rootFolderId?: number
}

export type BatchDenyInput = {
  ids: number[]
  reason?: string
}

// Quota types
export type UserQuota = {
  userId: number
  moviesLimit: number | null
  seasonsLimit: number | null
  episodesLimit: number | null
  moviesUsed: number
  seasonsUsed: number
  episodesUsed: number
  periodStart: string
}

export type QuotaLimits = {
  moviesLimit?: number | null
  seasonsLimit?: number | null
  episodesLimit?: number | null
}

// User notification types
export type UserNotification = {
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

export type CreateUserNotificationInput = {
  type: string
  name: string
  settings: Record<string, unknown>
  onAvailable: boolean
  onApproved: boolean
  onDenied: boolean
  enabled: boolean
}

// Search with availability
export type AvailabilityInfo = {
  inLibrary: boolean
  existingSlots: SlotInfo[]
  canRequest: boolean
  existingRequestId: number | null
  existingRequestUserId: number | null
  existingRequestStatus: RequestStatus | null
  mediaId: number | null
  addedAt: string | null
}

export type SlotInfo = {
  id: number
  name: string
  quality: string
}

export type PortalMovieSearchResult = {
  id: number
  tmdbId: number
  title: string
  year: number | null
  overview: string | null
  posterUrl: string | null
  backdropUrl: string | null
  availability?: AvailabilityInfo
}

export type PortalSeriesSearchResult = {
  id: number
  tmdbId: number
  tvdbId: number | null
  title: string
  year: number | null
  overview: string | null
  posterUrl: string | null
  backdropUrl: string | null
  network?: string
  networkLogoUrl?: string
  availability?: AvailabilityInfo
}

export type SeasonInfo = {
  seasonNumber: number
  name: string
  overview?: string
  posterUrl?: string
  airDate?: string
}

// Request settings
export type RequestSettings = {
  enabled: boolean
  defaultMovieQuota: number
  defaultSeasonQuota: number
  defaultEpisodeQuota: number
  defaultRootFolderId: number | null
  adminNotifyNew: boolean
  searchRateLimit: number
}

// Admin user management
export type AdminUpdateUserInput = {
  username?: string
  qualityProfileId?: number | null
  autoApprove?: boolean
  quotaOverride?: QuotaLimits
}

// Auto-approve result
export type AutoApproveResult = {
  autoApproved: boolean
  quotaExceeded: boolean
  searchStarted: boolean
  searchFound: boolean
  searchError?: string
}

export type CreateRequestResponse = {
  request: Request
  autoApprove: AutoApproveResult
}

// Portal download (queue item filtered to user's requests)
export type PortalDownload = {
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
