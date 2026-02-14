import type {
  AssignFileInput,
  EpisodeSlotAssignment,
  ExecuteMigrationInput,
  GeneratePreviewInput,
  MediaStatus,
  MigrationPreview,
  MigrationResult,
  MovieSlotAssignment,
  MultiVersionSettings,
  ParseReleaseInput,
  ParseReleaseOutput,
  ProfileMatchInput,
  ProfileMatchOutput,
  SetEnabledInput,
  SetMonitoredInput,
  SetProfileInput,
  SimulateImportInput,
  SimulateImportOutput,
  Slot,
  SlotNamingValidation,
  UpdateMultiVersionSettingsInput,
  UpdateSlotInput,
  ValidateConfigurationResponse,
  ValidateNamingInput,
} from '@/types'

import { apiFetch } from './client'

export const slotsApi = {
  // Multi-version settings
  getSettings: () => apiFetch<MultiVersionSettings>('/slots/settings'),

  updateSettings: (data: UpdateMultiVersionSettingsInput) =>
    apiFetch<MultiVersionSettings>('/slots/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Version slots
  list: () => apiFetch<Slot[]>('/slots'),

  get: (id: number) => apiFetch<Slot>(`/slots/${id}`),

  update: (id: number, data: UpdateSlotInput) =>
    apiFetch<Slot>(`/slots/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  setEnabled: (id: number, data: SetEnabledInput) =>
    apiFetch<Slot>(`/slots/${id}/enabled`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  setProfile: (id: number, data: SetProfileInput) =>
    apiFetch<Slot>(`/slots/${id}/profile`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Validation
  validateConfiguration: () =>
    apiFetch<ValidateConfigurationResponse>('/slots/validate', {
      method: 'POST',
    }),

  // Req 4.1.4: Validate naming formats include required differentiator tokens
  validateNaming: (data: ValidateNamingInput) =>
    apiFetch<SlotNamingValidation>('/slots/validate-naming', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Movie slot assignments
  getMovieSlotAssignments: (movieId: number) =>
    apiFetch<MovieSlotAssignment[]>(`/slots/movies/${movieId}/assignments`),

  assignMovieFile: (movieId: number, slotId: number, data: AssignFileInput) =>
    apiFetch<{ status: string }>(`/slots/movies/${movieId}/slots/${slotId}/assign`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  unassignMovieSlot: (movieId: number, slotId: number) =>
    apiFetch<{ status: string }>(`/slots/movies/${movieId}/slots/${slotId}/unassign`, {
      method: 'POST',
    }),

  // Episode slot assignments
  getEpisodeSlotAssignments: (episodeId: number) =>
    apiFetch<EpisodeSlotAssignment[]>(`/slots/episodes/${episodeId}/assignments`),

  assignEpisodeFile: (episodeId: number, slotId: number, data: AssignFileInput) =>
    apiFetch<{ status: string }>(`/slots/episodes/${episodeId}/slots/${slotId}/assign`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  unassignEpisodeSlot: (episodeId: number, slotId: number) =>
    apiFetch<{ status: string }>(`/slots/episodes/${episodeId}/slots/${slotId}/unassign`, {
      method: 'POST',
    }),

  // Movie status (Phase 5: Status & Monitoring)
  getMovieStatus: (movieId: number) => apiFetch<MediaStatus>(`/slots/movies/${movieId}/status`),

  setMovieSlotMonitored: (movieId: number, slotId: number, data: SetMonitoredInput) =>
    apiFetch<{ status: string }>(`/slots/movies/${movieId}/slots/${slotId}/monitored`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Episode status
  getEpisodeStatus: (episodeId: number) =>
    apiFetch<MediaStatus>(`/slots/episodes/${episodeId}/status`),

  setEpisodeSlotMonitored: (episodeId: number, slotId: number, data: SetMonitoredInput) =>
    apiFetch<{ status: string }>(`/slots/episodes/${episodeId}/slots/${slotId}/monitored`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Debug API (gated behind developerMode)
  parseRelease: (data: ParseReleaseInput) =>
    apiFetch<ParseReleaseOutput>('/slots/debug/parse-release', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  profileMatch: (data: ProfileMatchInput) =>
    apiFetch<ProfileMatchOutput>('/slots/debug/profile-match', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  simulateImport: (data: SimulateImportInput) =>
    apiFetch<SimulateImportOutput>('/slots/debug/simulate-import', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  generatePreview: (data: GeneratePreviewInput) =>
    apiFetch<MigrationPreview>('/slots/debug/generate-preview', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Migration/Dry Run (Req 14.1.1-14.2.3)
  getMigrationPreview: () =>
    apiFetch<MigrationPreview>('/slots/migration/preview', {
      method: 'POST',
    }),

  executeMigration: (data?: ExecuteMigrationInput) =>
    apiFetch<MigrationResult>('/slots/migration/execute', {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    }),
}
