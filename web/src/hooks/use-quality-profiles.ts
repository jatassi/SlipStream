import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { qualityProfilesApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type { CreateQualityProfileInput, QualityProfile, UpdateQualityProfileInput } from '@/types'

const baseKeys = createQueryKeys('qualityProfiles')
const qualityProfileKeys = {
  ...baseKeys,
  attributes: () => [...baseKeys.all, 'attributes'] as const,
}

export function useQualityProfiles() {
  return useQuery({
    queryKey: qualityProfileKeys.list(),
    queryFn: () => qualityProfilesApi.list(),
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

