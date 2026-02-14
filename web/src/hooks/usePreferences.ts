import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { type AddFlowPreferences, preferencesApi } from '@/api/preferences'

export const preferencesKeys = {
  all: ['preferences'] as const,
  addFlow: () => [...preferencesKeys.all, 'addflow'] as const,
}

export function useAddFlowPreferences() {
  return useQuery({
    queryKey: preferencesKeys.addFlow(),
    queryFn: preferencesApi.getAddFlowPreferences,
  })
}

export function useUpdateAddFlowPreferences() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (prefs: Partial<AddFlowPreferences>) => preferencesApi.setAddFlowPreferences(prefs),
    onSuccess: (data) => {
      queryClient.setQueryData(preferencesKeys.addFlow(), data)
    },
  })
}
