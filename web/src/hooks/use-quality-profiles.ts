import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { qualityProfilesApi } from '@/api'
import type { CreateQualityProfileInput, QualityProfile, UpdateQualityProfileInput } from '@/types'

export const qualityProfileKeys = {
  all: ['qualityProfiles'] as const,
  lists: () => [...qualityProfileKeys.all, 'list'] as const,
  list: () => [...qualityProfileKeys.lists()] as const,
  details: () => [...qualityProfileKeys.all, 'detail'] as const,
  detail: (id: number) => [...qualityProfileKeys.details(), id] as const,
  attributes: () => [...qualityProfileKeys.all, 'attributes'] as const,
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
      void queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
    },
  })
}

export function useUpdateQualityProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateQualityProfileInput }) =>
      qualityProfilesApi.update(id, data),
    onSuccess: (profile: QualityProfile) => {
      void queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
      queryClient.setQueryData(qualityProfileKeys.detail(profile.id), profile)
    },
  })
}

export function useDeleteQualityProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => qualityProfilesApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: qualityProfileKeys.all })
    },
  })
}

export function useQualityProfileAttributes() {
  return useQuery({
    queryKey: qualityProfileKeys.attributes(),
    queryFn: () => qualityProfilesApi.getAttributes(),
    staleTime: 24 * 60 * 60 * 1000, // Cache for 24 hours (static data)
  })
}

export function useCheckProfileExclusivity() {
  return useMutation({
    mutationFn: (profileIds: number[]) => qualityProfilesApi.checkExclusivity(profileIds),
  })
}
