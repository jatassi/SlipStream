import { apiFetch } from './client'
import type { BrowseResult } from '@/types'

export const filesystemApi = {
  /**
   * Browse directories at the given path
   * If path is empty, returns root (drives on Windows, / on Unix)
   */
  browse: (path?: string) =>
    apiFetch<BrowseResult>(
      `/filesystem/browse${path ? `?path=${encodeURIComponent(path)}` : ''}`
    ),
}
