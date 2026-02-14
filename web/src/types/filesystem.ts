export type DirectoryEntry = {
  name: string
  path: string
  isDir: boolean
}

export type DriveInfo = {
  letter: string
  label?: string
  type?: string
  freeSpace?: number
}

export type BrowseResult = {
  path: string
  parent?: string
  entries: DirectoryEntry[]
  drives?: DriveInfo[]
}
