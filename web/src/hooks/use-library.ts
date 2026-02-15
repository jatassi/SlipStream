import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { libraryApi } from '@/api/library'

import { movieKeys } from './use-movies'
import { seriesKeys } from './use-series'

export const libraryKeys = {
  all: ['library'] as const,
  scans: () => [...libraryKeys.all, 'scans'] as const,
  scanStatus: (id: number) => [...libraryKeys.scans(), id] as const,
}

/** Get all active scan statuses */
export function useScanStatuses() {
  return useQuery({
    queryKey: libraryKeys.scans(),
    queryFn: () => libraryApi.getScanStatuses(),
    refetchInterval: 2000, // Poll every 2s while scanning
  })
}

/** Get scan status for a specific root folder */
export function useScanStatus(id: number) {
  return useQuery({
    queryKey: libraryKeys.scanStatus(id),
    queryFn: () => libraryApi.getScanStatus(id),
    enabled: !!id,
  })
}

/** Trigger a scan of all root folders */
export function useScanLibrary() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => libraryApi.scanAll(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: libraryKeys.scans() })
    },
    onSettled: () => {
      setTimeout(() => {
        void queryClient.invalidateQueries({ queryKey: movieKeys.all })
        void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      }, 1000)
    },
  })
}

/** Trigger a scan of a specific root folder */
export function useScanRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => libraryApi.scanRootFolder(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: libraryKeys.scans() })
    },
  })
}

/** Cancel a scan for a specific root folder */
export function useCancelScan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => libraryApi.cancelScan(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: libraryKeys.scans() })
    },
  })
}
