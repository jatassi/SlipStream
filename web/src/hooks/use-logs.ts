import { useEffect } from 'react'

import { useMutation, useQuery } from '@tanstack/react-query'

import { logsApi } from '@/api/logs'
import { useLogsStore } from '@/stores/logs'

export const logsKeys = {
  all: ['logs'] as const,
  recent: () => [...logsKeys.all, 'recent'] as const,
}

export function useLogs() {
  const setEntries = useLogsStore((state) => state.setEntries)

  const query = useQuery({
    queryKey: logsKeys.recent(),
    queryFn: () => logsApi.getRecent(),
    refetchOnWindowFocus: false,
    staleTime: Infinity,
  })

  useEffect(() => {
    if (query.data) {
      setEntries(query.data)
    }
  }, [query.data, setEntries])

  return query
}

export function useDownloadLogFile() {
  return useMutation({
    mutationFn: () => logsApi.downloadLogFile(),
  })
}
