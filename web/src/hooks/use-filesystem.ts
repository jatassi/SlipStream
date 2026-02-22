import { useQuery } from '@tanstack/react-query'

import { filesystemApi } from '@/api'

const filesystemKeys = {
  all: ['filesystem'] as const,
  browse: (path?: string, extensions?: string[]) =>
    [...filesystemKeys.all, 'browse', path, extensions] as const,
  browseImport: (path?: string) => [...filesystemKeys.all, 'browseImport', path] as const,
}

/**
 * Hook to browse directories at the given path
 * @param path - Path to browse. If empty, returns root (drives on Windows)
 * @param enabled - Whether to enable the query
 * @param extensions - Optional file extensions to include (e.g., ['.db', '.sqlite'])
 */
export function useBrowseDirectory(path?: string, enabled = true, extensions?: string[]) {
  return useQuery({
    queryKey: filesystemKeys.browse(path, extensions),
    queryFn: () => filesystemApi.browse(path, extensions),
    enabled,
    staleTime: 30_000, // Cache for 30 seconds
  })
}

/**
 * Hook to browse directories and video files for import
 * @param path - Path to browse. If empty, returns root
 * @param enabled - Whether to enable the query
 */
export function useBrowseForImport(path?: string, enabled = true) {
  return useQuery({
    queryKey: filesystemKeys.browseImport(path),
    queryFn: () => filesystemApi.browseForImport(path),
    enabled,
    staleTime: 30_000,
  })
}
