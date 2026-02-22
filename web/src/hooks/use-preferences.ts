import { useQuery } from '@tanstack/react-query'

import { preferencesApi } from '@/api/preferences'

const preferencesKeys = {
  all: ['preferences'] as const,
  addFlow: () => [...preferencesKeys.all, 'addflow'] as const,
}

export function useAddFlowPreferences() {
  return useQuery({
    queryKey: preferencesKeys.addFlow(),
    queryFn: preferencesApi.getAddFlowPreferences,
  })
}

