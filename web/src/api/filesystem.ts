import type { BrowseResult } from '@/types'

import { apiFetch } from './client'

export type ImportBrowseResult = {
  path: string
  parent?: string
  directories: { name: string; path: string; isDir: boolean }[]
  files: { name: string; path: string; size: number; modTime: number }[]
  drives?: { letter: string; label?: string; type?: string }[]
}

export const filesystemApi = {
  /**
   * Browse directories at the given path
   * If path is empty, returns root (drives on Windows, / on Unix)
   */
  browse: (path?: string) =>
    apiFetch<BrowseResult>(`/filesystem/browse${path ? `?path=${encodeURIComponent(path)}` : ''}`),

  /**
   * Browse directories and video files for import
   */
  browseForImport: (path?: string) =>
    apiFetch<ImportBrowseResult>(
      `/filesystem/browse/import${path ? `?path=${encodeURIComponent(path)}` : ''}`,
    ),
}
