export interface DirectoryEntry {
  name: string
  path: string
  isDir: boolean
}

export interface DriveInfo {
  letter: string
  label?: string
  type?: string
  freeSpace?: number
}

export interface BrowseResult {
  path: string
  parent?: string
  entries: DirectoryEntry[]
  drives?: DriveInfo[]
}
