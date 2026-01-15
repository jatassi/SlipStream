import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { slotsApi } from '@/api'
import type {
  Slot,
  MultiVersionSettings,
  UpdateSlotInput,
  UpdateMultiVersionSettingsInput,
  SetEnabledInput,
  SetProfileInput,
  AssignFileInput,
  SetMonitoredInput,
  ParseReleaseInput,
  ProfileMatchInput,
  SimulateImportInput,
  ValidateNamingInput,
  ExecuteMigrationInput,
} from '@/types'

export const slotsKeys = {
  all: ['slots'] as const,
  lists: () => [...slotsKeys.all, 'list'] as const,
  list: () => [...slotsKeys.lists()] as const,
  details: () => [...slotsKeys.all, 'detail'] as const,
  detail: (id: number) => [...slotsKeys.details(), id] as const,
  settings: () => [...slotsKeys.all, 'settings'] as const,
  validation: () => [...slotsKeys.all, 'validation'] as const,
  movieAssignments: (movieId: number) => [...slotsKeys.all, 'movie', movieId] as const,
  movieStatus: (movieId: number) => [...slotsKeys.all, 'movie', movieId, 'status'] as const,
  episodeAssignments: (episodeId: number) => [...slotsKeys.all, 'episode', episodeId] as const,
  episodeStatus: (episodeId: number) => [...slotsKeys.all, 'episode', episodeId, 'status'] as const,
}

export function useSlots() {
  return useQuery({
    queryKey: slotsKeys.list(),
    queryFn: () => slotsApi.list(),
  })
}

export function useSlot(id: number) {
  return useQuery({
    queryKey: slotsKeys.detail(id),
    queryFn: () => slotsApi.get(id),
    enabled: !!id,
  })
}

export function useMultiVersionSettings() {
  return useQuery({
    queryKey: slotsKeys.settings(),
    queryFn: () => slotsApi.getSettings(),
  })
}

export function useUpdateMultiVersionSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateMultiVersionSettingsInput) =>
      slotsApi.updateSettings(data),
    onSuccess: (settings: MultiVersionSettings) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.all })
      queryClient.setQueryData(slotsKeys.settings(), settings)
    },
  })
}

export function useUpdateSlot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSlotInput }) =>
      slotsApi.update(id, data),
    onSuccess: (slot: Slot) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.all })
      queryClient.setQueryData(slotsKeys.detail(slot.id), slot)
    },
  })
}

export function useSetSlotEnabled() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: SetEnabledInput }) =>
      slotsApi.setEnabled(id, data),
    onSuccess: (slot: Slot) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.all })
      queryClient.setQueryData(slotsKeys.detail(slot.id), slot)
    },
  })
}

export function useSetSlotProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: SetProfileInput }) =>
      slotsApi.setProfile(id, data),
    onSuccess: (slot: Slot) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.all })
      queryClient.setQueryData(slotsKeys.detail(slot.id), slot)
    },
  })
}

export function useValidateSlotConfiguration() {
  return useMutation({
    mutationFn: () => slotsApi.validateConfiguration(),
  })
}

// Movie slot assignment hooks
export function useMovieSlotAssignments(movieId: number) {
  return useQuery({
    queryKey: slotsKeys.movieAssignments(movieId),
    queryFn: () => slotsApi.getMovieSlotAssignments(movieId),
    enabled: !!movieId,
  })
}

export function useAssignMovieFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      movieId,
      slotId,
      data,
    }: {
      movieId: number
      slotId: number
      data: AssignFileInput
    }) => slotsApi.assignMovieFile(movieId, slotId, data),
    onSuccess: (_, { movieId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.movieAssignments(movieId) })
    },
  })
}

export function useUnassignMovieSlot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      movieId,
      slotId,
    }: {
      movieId: number
      slotId: number
    }) => slotsApi.unassignMovieSlot(movieId, slotId),
    onSuccess: (_, { movieId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.movieAssignments(movieId) })
    },
  })
}

// Episode slot assignment hooks
export function useEpisodeSlotAssignments(episodeId: number) {
  return useQuery({
    queryKey: slotsKeys.episodeAssignments(episodeId),
    queryFn: () => slotsApi.getEpisodeSlotAssignments(episodeId),
    enabled: !!episodeId,
  })
}

export function useAssignEpisodeFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      episodeId,
      slotId,
      data,
    }: {
      episodeId: number
      slotId: number
      data: AssignFileInput
    }) => slotsApi.assignEpisodeFile(episodeId, slotId, data),
    onSuccess: (_, { episodeId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.episodeAssignments(episodeId) })
    },
  })
}

export function useUnassignEpisodeSlot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      episodeId,
      slotId,
    }: {
      episodeId: number
      slotId: number
    }) => slotsApi.unassignEpisodeSlot(episodeId, slotId),
    onSuccess: (_, { episodeId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.episodeAssignments(episodeId) })
    },
  })
}

// Phase 5: Status & Monitoring hooks

// Movie status hooks
export function useMovieSlotStatus(movieId: number) {
  return useQuery({
    queryKey: slotsKeys.movieStatus(movieId),
    queryFn: () => slotsApi.getMovieStatus(movieId),
    enabled: !!movieId,
  })
}

export function useSetMovieSlotMonitored() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      movieId,
      slotId,
      data,
    }: {
      movieId: number
      slotId: number
      data: SetMonitoredInput
    }) => slotsApi.setMovieSlotMonitored(movieId, slotId, data),
    onSuccess: (_, { movieId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.movieStatus(movieId) })
      queryClient.invalidateQueries({ queryKey: slotsKeys.movieAssignments(movieId) })
    },
  })
}

// Episode status hooks
export function useEpisodeSlotStatus(episodeId: number) {
  return useQuery({
    queryKey: slotsKeys.episodeStatus(episodeId),
    queryFn: () => slotsApi.getEpisodeStatus(episodeId),
    enabled: !!episodeId,
  })
}

export function useSetEpisodeSlotMonitored() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      episodeId,
      slotId,
      data,
    }: {
      episodeId: number
      slotId: number
      data: SetMonitoredInput
    }) => slotsApi.setEpisodeSlotMonitored(episodeId, slotId, data),
    onSuccess: (_, { episodeId }) => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.episodeStatus(episodeId) })
      queryClient.invalidateQueries({ queryKey: slotsKeys.episodeAssignments(episodeId) })
    },
  })
}

// Debug hooks (Phase 13: Debug & Testing)

export function useParseRelease() {
  return useMutation({
    mutationFn: (data: ParseReleaseInput) => slotsApi.parseRelease(data),
  })
}

export function useProfileMatch() {
  return useMutation({
    mutationFn: (data: ProfileMatchInput) => slotsApi.profileMatch(data),
  })
}

export function useSimulateImport() {
  return useMutation({
    mutationFn: (data: SimulateImportInput) => slotsApi.simulateImport(data),
  })
}

// File Naming Validation (Req 4.1.1-4.1.5)

export function useValidateNaming() {
  return useMutation({
    mutationFn: (data: ValidateNamingInput) => slotsApi.validateNaming(data),
  })
}

// Migration/Dry Run (Req 14.1.1-14.2.3)

export function useMigrationPreview() {
  return useMutation({
    mutationFn: () => slotsApi.getMigrationPreview(),
  })
}

export function useExecuteMigration() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data?: ExecuteMigrationInput) => slotsApi.executeMigration(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: slotsKeys.all })
      queryClient.invalidateQueries({ queryKey: slotsKeys.settings() })
    },
  })
}
