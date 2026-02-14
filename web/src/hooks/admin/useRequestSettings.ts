import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as adminApi from '@/api/admin'
import type { RequestSettings } from '@/types'

export const requestSettingsKeys = {
  all: ['admin', 'requestSettings'] as const,
  settings: () => [...requestSettingsKeys.all, 'settings'] as const,
}

export function useRequestSettings() {
  return useQuery({
    queryKey: requestSettingsKeys.settings(),
    queryFn: () => adminApi.getRequestSettings(),
  })
}

export function useUpdateRequestSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (settings: Partial<RequestSettings>) => adminApi.updateRequestSettings(settings),
    onSuccess: (settings) => {
      queryClient.setQueryData(requestSettingsKeys.settings(), settings)
    },
  })
}
