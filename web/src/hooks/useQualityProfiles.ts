import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { qualityProfilesApi } from '@/api'
import type {
  QualityProfile,
  CreateQualityProfileInput,
  UpdateQualityProfileInput,
} from '@/types'

export const qualityProfileKeys = {
  all: ['qualityProfiles'] as const,
  lists: () => [...qualityProfileKeys.all, 'list'] as const,
  list: () => [...qualityProfileKeys.lists()] as const,
  details: () => [...qualityProfileKeys.all, 'detail'] as const,
  detail: (id: number) => [...qualityProfileKeys.details(), id] as const,
}

export function useQualityProfiles() {
  return useQuery({
    queryKey: qualityProfileKeys.list(),
    queryFn: () => qualityProfilesApi.list(),
  })
}

export function useQualityProfile(id: number) {
  return useQuery({
    queryKey: qualityProfileKeys.detail(id),
    queryFn: () => qualityProfilesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateQualityProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateQualityProfileInput) => qualityProfilesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
    },
  })
}

export function useUpdateQualityProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateQualityProfileInput }) =>
      qualityProfilesApi.update(id, data),
    onSuccess: (profile: QualityProfile) => {
      queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
      queryClient.setQueryData(qualityProfileKeys.detail(profile.id), profile)
    },
  })
}

export function useDeleteQualityProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => qualityProfilesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
    },
  })
}
