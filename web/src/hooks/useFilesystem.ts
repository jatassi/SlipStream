import { useQuery } from '@tanstack/react-query'
import { filesystemApi } from '@/api'

export const filesystemKeys = {
  all: ['filesystem'] as const,
  browse: (path?: string) => [...filesystemKeys.all, 'browse', path] as const,
}

/**
 * Hook to browse directories at the given path
 * @param path - Path to browse. If empty, returns root (drives on Windows)
 * @param enabled - Whether to enable the query
 */
export function useBrowseDirectory(path?: string, enabled = true) {
  return useQuery({
    queryKey: filesystemKeys.browse(path),
    queryFn: () => filesystemApi.browse(path),
    enabled,
    staleTime: 30000, // Cache for 30 seconds
  })
}
